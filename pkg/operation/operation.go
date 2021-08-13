// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package operation

import (
	"context"
	"fmt"
	"strings"

	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	gardencorev1alpha1helper "github.com/gardener/gardener/pkg/apis/core/v1alpha1/helper"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gardencoreinformers "github.com/gardener/gardener/pkg/client/core/informers/externalversions/core/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/client/kubernetes/clientmap"
	"github.com/gardener/gardener/pkg/client/kubernetes/clientmap/keys"
	"github.com/gardener/gardener/pkg/gardenlet/apis/config"
	"github.com/gardener/gardener/pkg/operation/common"
	"github.com/gardener/gardener/pkg/operation/etcdencryption"
	"github.com/gardener/gardener/pkg/operation/garden"
	"github.com/gardener/gardener/pkg/operation/seed"
	shootpkg "github.com/gardener/gardener/pkg/operation/shoot"
	"github.com/gardener/gardener/pkg/utils/chart"
	"github.com/gardener/gardener/pkg/utils/flow"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	"github.com/gardener/gardener/pkg/utils/imagevector"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewBuilder returns a new Builder.
func NewBuilder() *Builder {
	return &Builder{
		configFunc: func() (*config.GardenletConfiguration, error) {
			return nil, fmt.Errorf("config is required but not set")
		},
		gardenFunc: func(context.Context, map[string]*corev1.Secret) (*garden.Garden, error) {
			return nil, fmt.Errorf("garden object is required but not set")
		},
		gardenerInfoFunc: func() (*gardencorev1beta1.Gardener, error) {
			return nil, fmt.Errorf("gardener info is required but not set")
		},
		gardenClusterIdentityFunc: func() (string, error) {
			return "", fmt.Errorf("garden cluster identity is required but not set")
		},
		imageVectorFunc: func() (imagevector.ImageVector, error) {
			return nil, fmt.Errorf("image vector is required but not set")
		},
		loggerFunc: func() (*logrus.Entry, error) {
			return nil, fmt.Errorf("logger is required but not set")
		},
		secretsFunc: func() (map[string]*corev1.Secret, error) {
			return nil, fmt.Errorf("secrets map is required but not set")
		},
		seedFunc: func(context.Context) (*seed.Seed, error) {
			return nil, fmt.Errorf("seed object is required but not set")
		},
		shootFunc: func(context.Context, client.Client, *garden.Garden, *seed.Seed) (*shootpkg.Shoot, error) {
			return nil, fmt.Errorf("shoot object is required but not set")
		},
		exposureClassFunc: func(string) (*config.ExposureClassHandler, error) {
			return nil, nil
		},
	}
}

// WithConfig sets the configFunc attribute at the Builder.
func (b *Builder) WithConfig(cfg *config.GardenletConfiguration) *Builder {
	b.configFunc = func() (*config.GardenletConfiguration, error) { return cfg, nil }
	return b
}

// WithGarden sets the gardenFunc attribute at the Builder.
func (b *Builder) WithGarden(g *garden.Garden) *Builder {
	b.gardenFunc = func(context.Context, map[string]*corev1.Secret) (*garden.Garden, error) { return g, nil }
	return b
}

// WithGardenFrom sets the gardenFunc attribute at the Builder which will build a new Garden object.
func (b *Builder) WithGardenFrom(reader client.Reader, namespace string) *Builder {
	b.gardenFunc = func(ctx context.Context, secrets map[string]*corev1.Secret) (*garden.Garden, error) {
		return garden.
			NewBuilder().
			WithProjectFromReader(reader, namespace).
			WithInternalDomainFromSecrets(secrets).
			WithDefaultDomainsFromSecrets(secrets).
			Build(ctx)
	}
	return b
}

// WithGardenerInfo sets the gardenerInfoFunc attribute at the Builder.
func (b *Builder) WithGardenerInfo(gardenerInfo *gardencorev1beta1.Gardener) *Builder {
	b.gardenerInfoFunc = func() (*gardencorev1beta1.Gardener, error) { return gardenerInfo, nil }
	return b
}

// WithGardenClusterIdentity sets the identity of the Garden cluster as attribute at the Builder.
func (b *Builder) WithGardenClusterIdentity(gardenClusterIdentity string) *Builder {
	b.gardenClusterIdentityFunc = func() (string, error) { return gardenClusterIdentity, nil }
	return b
}

// WithImageVector sets the imageVectorFunc attribute at the Builder.
func (b *Builder) WithImageVector(imageVector imagevector.ImageVector) *Builder {
	b.imageVectorFunc = func() (imagevector.ImageVector, error) { return imageVector, nil }
	return b
}

// WithLogger sets the loggerFunc attribute at the Builder.
func (b *Builder) WithLogger(logger *logrus.Entry) *Builder {
	b.loggerFunc = func() (*logrus.Entry, error) { return logger, nil }
	return b
}

// WithSecrets sets the secretsFunc attribute at the Builder.
func (b *Builder) WithSecrets(secrets map[string]*corev1.Secret) *Builder {
	b.secretsFunc = func() (map[string]*corev1.Secret, error) { return secrets, nil }
	return b
}

// WithSeed sets the seedFunc attribute at the Builder.
func (b *Builder) WithSeed(s *seed.Seed) *Builder {
	b.seedFunc = func(_ context.Context) (*seed.Seed, error) { return s, nil }
	return b
}

// WithSeedFrom sets the seedFunc attribute at the Builder which will build a new Seed object.
func (b *Builder) WithSeedFrom(k8sGardenCoreInformers gardencoreinformers.Interface, seedName string) *Builder {
	b.seedFunc = func(ctx context.Context) (*seed.Seed, error) {
		return seed.
			NewBuilder().
			WithSeedObjectFromLister(k8sGardenCoreInformers.Seeds().Lister(), seedName).
			Build()
	}
	return b
}

// WithShoot sets the shootFunc attribute at the Builder.
func (b *Builder) WithShoot(s *shootpkg.Shoot) *Builder {
	b.shootFunc = func(_ context.Context, _ client.Client, _ *garden.Garden, _ *seed.Seed) (*shootpkg.Shoot, error) {
		return s, nil
	}
	return b
}

// WithShootFrom sets the shootFunc attribute at the Builder which will build a new Shoot object.
func (b *Builder) WithShootFrom(k8sGardenCoreInformers gardencoreinformers.Interface, gardenClient kubernetes.Interface, s *gardencorev1beta1.Shoot) *Builder {
	b.shootFunc = func(ctx context.Context, c client.Client, gardenObj *garden.Garden, seedObj *seed.Seed) (*shootpkg.Shoot, error) {
		return shootpkg.
			NewBuilder().
			WithShootObject(s).
			WithCloudProfileObjectFromReader(gardenClient.Client()).
			WithShootSecretFromReader(gardenClient.Client()).
			WithProjectName(gardenObj.Project.Name).
			WithExposureClassFromReader(gardenClient.Client()).
			WithDisableDNS(!seedObj.Info.Spec.Settings.ShootDNS.Enabled).
			WithInternalDomain(gardenObj.InternalDomain).
			WithDefaultDomains(gardenObj.DefaultDomains).
			Build(ctx, c)
	}
	return b
}

// WithShootFromCluster sets the shootFunc attribute at the Builder which will build a new Shoot object constructed from the cluster resource.
// The shoot status is still taken from the passed `shoot`, though.
func (b *Builder) WithShootFromCluster(gardenClient, seedClient kubernetes.Interface, s *gardencorev1beta1.Shoot) *Builder {
	b.shootFunc = func(ctx context.Context, c client.Client, gardenObj *garden.Garden, seedObj *seed.Seed) (*shootpkg.Shoot, error) {
		shootNamespace := shootpkg.ComputeTechnicalID(gardenObj.Project.Name, s)

		shoot, err := shootpkg.
			NewBuilder().
			WithShootObjectFromCluster(seedClient, shootNamespace).
			WithCloudProfileObjectFromCluster(seedClient, shootNamespace).
			WithShootSecretFromReader(gardenClient.Client()).
			WithProjectName(gardenObj.Project.Name).
			WithExposureClassFromReader(gardenClient.Client()).
			WithDisableDNS(!seedObj.Info.Spec.Settings.ShootDNS.Enabled).
			WithInternalDomain(gardenObj.InternalDomain).
			WithDefaultDomains(gardenObj.DefaultDomains).
			Build(ctx, c)
		if err != nil {
			return nil, err
		}
		shoot.Info.Status = s.Status
		return shoot, nil
	}
	return b
}

// WithExposureClassHandlerFromConfig sets the exposureClassFunc attribute at the Builder which will find the
// the required exposure class handler in the passed Gardenlet config.
func (b *Builder) WithExposureClassHandlerFromConfig(cfg *config.GardenletConfiguration) *Builder {
	b.exposureClassFunc = func(handlerName string) (*config.ExposureClassHandler, error) {
		for _, handler := range cfg.ExposureClassHandlers {
			if handler.Name == handlerName {
				return &handler, nil
			}
		}
		return nil, fmt.Errorf("No exposure class handler with name %q found", handlerName)
	}
	return b
}

// Build initializes a new Operation object.
func (b *Builder) Build(ctx context.Context, clientMap clientmap.ClientMap) (*Operation, error) {
	operation := &Operation{
		ClientMap: clientMap,
	}

	gardenClient, err := clientMap.GetClient(ctx, keys.ForGarden())
	if err != nil {
		return nil, fmt.Errorf("failed to get garden client: %w", err)
	}
	operation.K8sGardenClient = gardenClient

	config, err := b.configFunc()
	if err != nil {
		return nil, err
	}
	operation.Config = config

	secretsMap, err := b.secretsFunc()
	if err != nil {
		return nil, err
	}
	secrets := make(map[string]*corev1.Secret)
	for k, v := range secretsMap {
		secrets[k] = v
	}
	operation.Secrets = secrets

	garden, err := b.gardenFunc(ctx, secrets)
	if err != nil {
		return nil, err
	}
	operation.Garden = garden

	gardenerInfo, err := b.gardenerInfoFunc()
	if err != nil {
		return nil, err
	}
	operation.GardenerInfo = gardenerInfo

	gardenClusterIdentity, err := b.gardenClusterIdentityFunc()
	if err != nil {
		return nil, err
	}
	operation.GardenClusterIdentity = gardenClusterIdentity

	imageVector, err := b.imageVectorFunc()
	if err != nil {
		return nil, err
	}
	operation.ImageVector = imageVector

	logger, err := b.loggerFunc()
	if err != nil {
		return nil, err
	}
	operation.Logger = logger

	seed, err := b.seedFunc(ctx)
	if err != nil {
		return nil, err
	}
	operation.Seed = seed

	shoot, err := b.shootFunc(ctx, gardenClient.Client(), garden, seed)
	if err != nil {
		return nil, err
	}
	operation.Shoot = shoot

	if shoot.ExposureClass != nil {
		exposureClassHandler, err := b.exposureClassFunc(shoot.ExposureClass.Handler)
		if err != nil {
			return nil, err
		}
		operation.ExposureClassHandler = exposureClassHandler
	}

	// Get the ManagedSeed object for this shoot, if it exists.
	// Also read the managed seed API server settings from the managed-seed-api-server annotation.
	operation.ManagedSeed, err = kutil.GetManagedSeed(ctx, gardenClient.GardenSeedManagement(), shoot.Info.Namespace, shoot.Info.Name)
	if err != nil {
		return nil, fmt.Errorf("could not get managed seed for shoot %s/%s: %w", shoot.Info.Namespace, shoot.Info.Name, err)
	}
	operation.ManagedSeedAPIServer, err = gardencorev1beta1helper.ReadManagedSeedAPIServer(shoot.Info)
	if err != nil {
		return nil, fmt.Errorf("could not read managed seed API server settings of shoot %s/%s: %+v", shoot.Info.Namespace, shoot.Info.Name, err)
	}

	// If the managed-seed-api-server annotation is not present, try to read the managed seed API server settings
	// from the use-as-seed annotation. This is done to avoid re-annotating a shoot annotated with the use-as-seed annotation
	// by the shooted seed registration controller.
	if operation.ManagedSeedAPIServer == nil {
		shootedSeed, err := gardencorev1beta1helper.ReadShootedSeed(shoot.Info)
		if err != nil {
			return nil, fmt.Errorf("could not read managed seed API server settings of shoot %s/%s: %+v", shoot.Info.Namespace, shoot.Info.Name, err)
		}
		if shootedSeed != nil {
			operation.ManagedSeedAPIServer = shootedSeed.APIServer
		}
	}

	return operation, nil
}

// InitializeSeedClients will use the Garden Kubernetes client to read the Seed Secret in the Garden
// cluster which contains a Kubeconfig that can be used to authenticate against the Seed cluster. With it,
// a Kubernetes client as well as a Chart renderer for the Seed cluster will be initialized and attached to
// the already existing Operation object.
func (o *Operation) InitializeSeedClients(ctx context.Context) error {
	if o.K8sSeedClient != nil {
		return nil
	}

	seedClient, err := o.ClientMap.GetClient(ctx, keys.ForSeed(o.Seed.Info))
	if err != nil {
		return fmt.Errorf("failed to get seed client: %w", err)
	}
	o.K8sSeedClient = seedClient
	return nil
}

// InitializeShootClients will use the Seed Kubernetes client to read the gardener Secret in the Seed
// cluster which contains a Kubeconfig that can be used to authenticate against the Shoot cluster. With it,
// a Kubernetes client as well as a Chart renderer for the Shoot cluster will be initialized and attached to
// the already existing Operation object.
func (o *Operation) InitializeShootClients(ctx context.Context) error {
	if o.K8sShootClient != nil {
		return nil
	}

	if o.Shoot.HibernationEnabled {
		// Don't initialize clients for Shoots, that are currently hibernated and their API server is not running
		apiServerRunning, err := o.IsAPIServerRunning(ctx)
		if err != nil {
			return err
		}
		if !apiServerRunning {
			return nil
		}
	}

	shootClient, err := o.ClientMap.GetClient(ctx, keys.ForShoot(o.Shoot.Info))
	if err != nil {
		return err
	}
	o.K8sShootClient = shootClient

	return nil
}

// IsAPIServerRunning checks if the API server of the Shoot currently running (not scaled-down/deleted).
func (o *Operation) IsAPIServerRunning(ctx context.Context) (bool, error) {
	deployment := &appsv1.Deployment{}
	// use API reader here to make sure, we're not reading from a stale cache, when checking if we should initialize a shoot client (e.g. from within the care controller)
	if err := o.K8sSeedClient.APIReader().Get(ctx, kutil.Key(o.Shoot.SeedNamespace, v1beta1constants.DeploymentNameKubeAPIServer), deployment); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	if deployment.GetDeletionTimestamp() != nil {
		return false, nil
	}

	if deployment.Spec.Replicas == nil {
		return false, nil
	}
	return *deployment.Spec.Replicas > 0, nil
}

// GetSecretKeysOfRole returns a list of keys which are present in the Garden Secrets map and which
// are prefixed with <kind>.
func (o *Operation) GetSecretKeysOfRole(kind string) []string {
	return common.GetSecretKeysWithPrefix(kind, o.Secrets)
}

func makeDescription(stats *flow.Stats) string {
	if stats.ProgressPercent() == 0 {
		return "Starting " + stats.FlowName
	}
	if stats.ProgressPercent() == 100 {
		return stats.FlowName + " finished"
	}
	return strings.Join(stats.Running.StringList(), ", ")
}

// ReportShootProgress will update the last operation object in the Shoot manifest `status` section
// by the current progress of the Flow execution.
func (o *Operation) ReportShootProgress(ctx context.Context, stats *flow.Stats) {
	var (
		description    = makeDescription(stats)
		progress       = stats.ProgressPercent()
		lastUpdateTime = metav1.Now()
		shoot          = o.Shoot.Info
	)

	if err := func() error {
		statusPatch := client.StrategicMergeFrom(shoot.DeepCopy())

		if shoot.Status.LastOperation == nil {
			return fmt.Errorf("last operation of Shoot %s/%s is unset", shoot.Namespace, shoot.Name)
		}
		if shoot.Status.LastOperation.LastUpdateTime.After(lastUpdateTime.Time) {
			return fmt.Errorf("last operation of Shoot %s/%s was updated mid-air", shoot.Namespace, shoot.Name)
		}
		if description != "" {
			shoot.Status.LastOperation.Description = description
		}
		shoot.Status.LastOperation.Progress = progress
		shoot.Status.LastOperation.LastUpdateTime = lastUpdateTime

		return o.K8sGardenClient.Client().Status().Patch(ctx, shoot, statusPatch)
	}(); err != nil {
		o.Logger.Errorf("Could not report shoot progress: %v", err)
		return
	}
}

// CleanShootTaskErrorAndUpdateStatusLabel removes the error with taskID from the Shoot's status.LastErrors array.
// If the status.LastErrors array is empty then status.LastErrors is also removed. It also re-evaluates the shoot status
// in case the last error list is empty now, and if necessary, updates the status label on the shoot.
func (o *Operation) CleanShootTaskErrorAndUpdateStatusLabel(ctx context.Context, taskID string) {
	shoot := o.Shoot.Info

	statusPatch := client.MergeFrom(shoot.DeepCopy())
	// LastErrors doesn't have patchMergeKey as TaskID is optional, so strategic merge wouldn't make a difference.
	// This will effectively overwrite the whole LastErrors array. Though, this should be fine, as this controller is
	// supposed to be the exclusive owner of this field.
	shoot.Status.LastErrors = gardencorev1beta1helper.DeleteLastErrorByTaskID(shoot.Status.LastErrors, taskID)
	if err := o.K8sGardenClient.Client().Status().Patch(ctx, shoot, statusPatch); err != nil {
		o.Logger.Errorf("Could not update shoot's %s/%s last errors: %v", shoot.Namespace, shoot.Name, err)
		return
	}

	if len(shoot.Status.LastErrors) == 0 {
		metaPatch := client.MergeFrom(shoot.DeepCopy())
		kutil.SetMetaDataLabel(&shoot.ObjectMeta, v1beta1constants.ShootStatus, string(shootpkg.ComputeStatus(
			shoot.Status.LastOperation,
			shoot.Status.LastErrors,
			shoot.Status.Conditions...,
		)))
		if err := o.K8sGardenClient.Client().Patch(ctx, shoot, metaPatch); err != nil {
			o.Logger.Errorf("Could not update shoot's %s/%s status label after removing an erroneous task: %v", shoot.Namespace, shoot.Name, err)
			return
		}
	}
}

// SeedVersion is a shorthand for the kubernetes version of the K8sSeedClient.
func (o *Operation) SeedVersion() string {
	return o.K8sSeedClient.Version()
}

// ShootVersion is a shorthand for the desired kubernetes version of the operation's shoot.
func (o *Operation) ShootVersion() string {
	return o.Shoot.Info.Spec.Kubernetes.Version
}

// InjectSeedSeedImages injects images that shall run on the Seed and target the Seed's Kubernetes version.
func (o *Operation) InjectSeedSeedImages(values map[string]interface{}, names ...string) (map[string]interface{}, error) {
	return chart.InjectImages(values, o.ImageVector, names, imagevector.RuntimeVersion(o.SeedVersion()), imagevector.TargetVersion(o.SeedVersion()))
}

// InjectSeedShootImages injects images that shall run on the Seed but target the Shoot's Kubernetes version.
func (o *Operation) InjectSeedShootImages(values map[string]interface{}, names ...string) (map[string]interface{}, error) {
	return chart.InjectImages(values, o.ImageVector, names, imagevector.RuntimeVersion(o.SeedVersion()), imagevector.TargetVersion(o.ShootVersion()))
}

// InjectShootShootImages injects images that shall run on the Shoot and target the Shoot's Kubernetes version.
func (o *Operation) InjectShootShootImages(values map[string]interface{}, names ...string) (map[string]interface{}, error) {
	return chart.InjectImages(values, o.ImageVector, names, imagevector.RuntimeVersion(o.ShootVersion()), imagevector.TargetVersion(o.ShootVersion()))
}

// EnsureShootStateExists creates the ShootState resource for the corresponding shoot and updates the operations object
func (o *Operation) EnsureShootStateExists(ctx context.Context) error {
	var (
		err        error
		shootState = &gardencorev1alpha1.ShootState{
			ObjectMeta: metav1.ObjectMeta{
				Name:      o.Shoot.Info.Name,
				Namespace: o.Shoot.Info.Namespace,
			},
		}
	)

	if err = o.K8sGardenClient.Client().Create(ctx, shootState); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
	}

	if err = o.K8sGardenClient.Client().Get(ctx, client.ObjectKeyFromObject(shootState), shootState); err != nil {
		return err
	}
	o.ShootState = shootState
	gardenerResourceList := gardencorev1alpha1helper.GardenerResourceDataList(shootState.Spec.Gardener)
	o.Shoot.ETCDEncryption, err = etcdencryption.GetEncryptionConfig(gardenerResourceList)
	return err
}

func (o *Operation) DeleteShootState(ctx context.Context) error {
	shootState := &gardencorev1alpha1.ShootState{
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.Shoot.Info.Name,
			Namespace: o.Shoot.Info.Namespace,
		},
	}

	if err := gutil.ConfirmDeletion(ctx, o.K8sGardenClient.Client(), shootState); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return client.IgnoreNotFound(o.K8sGardenClient.Client().Delete(ctx, shootState))
}

// SaveGardenerResourcesInShootState saves the provided GardenerResourcesDataList in the ShootState's `gardener` field
func (o *Operation) SaveGardenerResourcesInShootState(ctx context.Context, resourceList gardencorev1alpha1helper.GardenerResourceDataList) error {
	shootState := o.ShootState.DeepCopy()
	shootState.Spec.Gardener = resourceList
	if err := o.K8sGardenClient.Client().Patch(ctx, shootState, client.MergeFrom(o.ShootState)); err != nil {
		return err
	}
	o.ShootState = shootState
	return nil
}

// DeleteClusterResourceFromSeed deletes the `Cluster` extension resource for the shoot in the seed cluster.
func (o *Operation) DeleteClusterResourceFromSeed(ctx context.Context) error {
	if err := o.InitializeSeedClients(ctx); err != nil {
		o.Logger.Errorf("Could not initialize a new Kubernetes client for the seed cluster: %s", err.Error())
		return err
	}

	return client.IgnoreNotFound(o.K8sSeedClient.Client().Delete(ctx, &extensionsv1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: o.Shoot.SeedNamespace}}))
}

// ComputeGrafanaHosts computes the host for both grafanas.
func (o *Operation) ComputeGrafanaHosts() []string {
	return []string{
		o.ComputeGrafanaOperatorsHost(),
		o.ComputeGrafanaUsersHost(),
	}
}

// ComputePrometheusHosts computes the hosts for prometheus.
func (o *Operation) ComputePrometheusHosts() []string {
	return []string{
		o.ComputePrometheusHost(),
	}
}

// ComputeAlertManagerHosts computes the host for alert manager.
func (o *Operation) ComputeAlertManagerHosts() []string {
	return []string{
		o.ComputeAlertManagerHost(),
	}
}

// ComputeGrafanaOperatorsHost computes the host for users Grafana.
func (o *Operation) ComputeGrafanaOperatorsHost() string {
	return o.ComputeIngressHost(common.GrafanaOperatorsPrefix)
}

// ComputeLokiHosts computes the host for loki.
func (o *Operation) ComputeLokiHosts() []string {
	return []string{
		o.ComputeLokiHost(),
	}
}

// ComputeGrafanaUsersHost computes the host for operators Grafana.
func (o *Operation) ComputeGrafanaUsersHost() string {
	return o.ComputeIngressHost(common.GrafanaUsersPrefix)
}

// ComputeAlertManagerHost computes the host for alert manager.
func (o *Operation) ComputeAlertManagerHost() string {
	return o.ComputeIngressHost(common.AlertManagerPrefix)
}

// ComputePrometheusHost computes the host for prometheus.
func (o *Operation) ComputePrometheusHost() string {
	return o.ComputeIngressHost(common.PrometheusPrefix)
}

// ComputeLokiHost computes the host for loki.
func (o *Operation) ComputeLokiHost() string {
	return o.ComputeIngressHost(common.LokiPrefix)
}

// ComputeIngressHost computes the host for a given prefix.
func (o *Operation) ComputeIngressHost(prefix string) string {
	shortID := strings.Replace(o.Shoot.Info.Status.TechnicalID, shootpkg.TechnicalIDPrefix, "", 1)
	return fmt.Sprintf("%s-%s.%s", prefix, shortID, o.Seed.IngressDomain())
}

// ToAdvertisedAddresses returns list of advertised addresses on a Shoot cluster.
func (o *Operation) ToAdvertisedAddresses() []gardencorev1beta1.ShootAdvertisedAddress {
	var addresses []gardencorev1beta1.ShootAdvertisedAddress

	if o.Shoot == nil {
		return addresses
	}

	if o.Shoot.ExternalClusterDomain != nil && len(*o.Shoot.ExternalClusterDomain) > 0 {
		addresses = append(addresses, gardencorev1beta1.ShootAdvertisedAddress{
			Name: "external",
			URL:  "https://" + gutil.GetAPIServerDomain(*o.Shoot.ExternalClusterDomain),
		})
	}

	if len(o.Shoot.InternalClusterDomain) > 0 {
		addresses = append(addresses, gardencorev1beta1.ShootAdvertisedAddress{
			Name: "internal",
			URL:  "https://" + gutil.GetAPIServerDomain(o.Shoot.InternalClusterDomain),
		})
	}

	if len(o.APIServerAddress) > 0 && len(addresses) == 0 {
		addresses = append(addresses, gardencorev1beta1.ShootAdvertisedAddress{
			Name: "unmanaged",
			URL:  "https://" + o.APIServerAddress,
		})
	}

	return addresses
}

// StoreCheckSum stores the passed checksum under the given key from the operation. Calling this function is thread-safe.
func (o *Operation) StoreCheckSum(key, value string) {
	o.checkSumsMutex.Lock()
	defer o.checkSumsMutex.Unlock()

	if o.checkSums == nil {
		o.checkSums = make(map[string]string)
	}

	o.checkSums[key] = value
}

// LoadCheckSum loads the checksum value under the given key from the operation. Calling this function is thread-safe.
func (o *Operation) LoadCheckSum(key string) string {
	o.checkSumsMutex.RLock()
	defer o.checkSumsMutex.RUnlock()

	val := o.checkSums[key]
	return val
}

// DeleteCheckSum deletes the checksum entry under the given key from the operation. Calling this function is thread-safe.
func (o *Operation) DeleteCheckSum(key string) {
	o.checkSumsMutex.Lock()
	defer o.checkSumsMutex.Unlock()

	delete(o.checkSums, key)
}
