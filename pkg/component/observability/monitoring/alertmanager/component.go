// Copyright 2024 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package alertmanager

import (
	"context"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/component"
	"github.com/gardener/gardener/pkg/component/observability/monitoring"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
)

const (
	port = 9093
	// PortNameMetrics is the name of the metrics port.
	PortNameMetrics = "metrics"
)

// Interface contains functions for an alertmanager deployer.
type Interface interface {
	component.DeployWaiter
	// SetIngressAuthSecret sets the ingress auth secret name.
	SetIngressAuthSecret(*corev1.Secret)
	// SetIngressWildcardCertSecret sets the ingress wildcard cert secret name.
	SetIngressWildcardCertSecret(*corev1.Secret)
}

// Values contains configuration values for the AlertManager resources.
type Values struct {
	// Name is the name of the AlertManager. It will be used for the resource names of AlertManager and ManagedResource.
	Name string
	// Image defines the container image of AlertManager.
	Image string
	// Version is the version of AlertManager.
	Version string
	// ClusterType is the type of the cluster.
	ClusterType component.ClusterType
	// PriorityClassName is the name of the priority class for the StatefulSet.
	PriorityClassName string
	// StorageCapacity is the storage capacity of AlertManager.
	StorageCapacity resource.Quantity
	// Replicas is the number of replicas.
	Replicas int32
	// RuntimeVersion is the Kubernetes version of the runtime cluster.
	RuntimeVersion *semver.Version
	// AlertingSMTPSecret is the alerting SMTP secret.
	AlertingSMTPSecret *corev1.Secret
	// EmailReceivers is a list of email addresses to which alerts should be sent. If this list is empty, the alerts
	// will be sent to the email address in `.data.to` in the alerting SMTP secret.
	EmailReceivers []string
	// Ingress contains configuration for exposing this AlertManager instance via an Ingress resource.
	Ingress *IngressValues

	// DataMigration is a struct for migrating data from existing disks.
	// TODO(rfranzke): Remove this as soon as the PV migration code is removed.
	DataMigration monitoring.DataMigration
}

// IngressValues contains configuration for exposing this AlertManager instance via an Ingress resource.
type IngressValues struct {
	// AuthSecretName is the name of the auth secret.
	AuthSecretName string
	// Host is the hostname under which the AlertManager instance should be exposed.
	Host string
	// SecretsManager is the secrets manager used for generating the TLS certificate if no wildcard certificate is
	// provided.
	SecretsManager secretsmanager.Interface
	// WildcardCertSecretName is name of a secret containing the wildcard TLS certificate which is issued for the
	// ingress domain. If not provided, a self-signed server certificate will be created.
	WildcardCertSecretName *string
}

// New creates a new instance of DeployWaiter for the AlertManager.
func New(log logr.Logger, client client.Client, namespace string, values Values) Interface {
	return &alertManager{
		log:       log,
		client:    client,
		namespace: namespace,
		values:    values,
	}
}

type alertManager struct {
	log       logr.Logger
	client    client.Client
	namespace string
	values    Values
}

func (a *alertManager) Deploy(ctx context.Context) error {
	var (
		log      = a.log.WithName("alertmanager-deployer").WithValues("name", a.values.Name)
		registry = managedresources.NewRegistry(kubernetes.SeedScheme, kubernetes.SeedCodec, kubernetes.SeedSerializer)
	)

	// TODO(rfranzke): Remove this migration code after all AlertManagers have been migrated.
	takeOverExistingPV, pv, oldPVC, err := a.values.DataMigration.ExistingPVTakeOverPrerequisites(ctx, log)
	if err != nil {
		return err
	}

	ingress, err := a.ingress(ctx)
	if err != nil {
		return err
	}

	resources, err := registry.AddAllAndSerialize(
		a.service(),
		a.alertManager(takeOverExistingPV),
		a.vpa(),
		a.podDisruptionBudget(),
		a.config(),
		a.smtpSecret(),
		ingress,
	)
	if err != nil {
		return err
	}

	if takeOverExistingPV {
		if err := a.values.DataMigration.PrepareExistingPVTakeOver(ctx, log, pv, oldPVC); err != nil {
			return err
		}

		log.Info("Deploy new AlertManager (with init container for renaming the data directory)")
	}

	if err := managedresources.CreateForSeed(ctx, a.client, a.namespace, a.name(), false, resources); err != nil {
		return err
	}

	if takeOverExistingPV {
		if err := a.values.DataMigration.FinalizeExistingPVTakeOver(ctx, log, pv); err != nil {
			return err
		}

		log.Info("Deploy new AlertManager again (to remove the migration init container)")
		return a.Deploy(ctx)
	}

	return nil
}

func (a *alertManager) Destroy(ctx context.Context) error {
	return managedresources.DeleteForSeed(ctx, a.client, a.namespace, a.name())
}

// TimeoutWaitForManagedResource is the timeout used while waiting for the ManagedResources to become healthy or
// deleted.
var TimeoutWaitForManagedResource = 5 * time.Minute

func (a *alertManager) Wait(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, TimeoutWaitForManagedResource)
	defer cancel()

	return managedresources.WaitUntilHealthy(timeoutCtx, a.client, a.namespace, a.name())
}

func (a *alertManager) WaitCleanup(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, TimeoutWaitForManagedResource)
	defer cancel()

	return managedresources.WaitUntilDeleted(timeoutCtx, a.client, a.namespace, a.name())
}

func (a *alertManager) SetIngressAuthSecret(secret *corev1.Secret) {
	if a.values.Ingress != nil && secret != nil {
		a.values.Ingress.AuthSecretName = secret.Name
	}
}

func (a *alertManager) SetIngressWildcardCertSecret(secret *corev1.Secret) {
	if a.values.Ingress != nil && secret != nil {
		a.values.Ingress.WildcardCertSecretName = &secret.Name
	}
}

func (a *alertManager) name() string {
	return "alertmanager-" + a.values.Name
}

func (a *alertManager) getLabels() map[string]string {
	return map[string]string{
		"component":                "alertmanager",
		v1beta1constants.LabelRole: v1beta1constants.LabelMonitoring,
		"alertmanager":             a.values.Name,
	}
}
