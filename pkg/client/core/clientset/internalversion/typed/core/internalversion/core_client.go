/*
Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package internalversion

import (
	"github.com/gardener/gardener/pkg/client/core/clientset/internalversion/scheme"
	rest "k8s.io/client-go/rest"
)

type CoreInterface interface {
	RESTClient() rest.Interface
	BackupBucketsGetter
	BackupEntriesGetter
	CloudProfilesGetter
	ControllerDeploymentsGetter
	ControllerInstallationsGetter
	ControllerRegistrationsGetter
	ExposureClassesGetter
	PlantsGetter
	ProjectsGetter
	QuotasGetter
	SecretBindingsGetter
	SeedsGetter
	ShootsGetter
	ShootExtensionStatusesGetter
	ShootLeftoversGetter
	ShootStatesGetter
}

// CoreClient is used to interact with features provided by the core.gardener.cloud group.
type CoreClient struct {
	restClient rest.Interface
}

func (c *CoreClient) BackupBuckets() BackupBucketInterface {
	return newBackupBuckets(c)
}

func (c *CoreClient) BackupEntries(namespace string) BackupEntryInterface {
	return newBackupEntries(c, namespace)
}

func (c *CoreClient) CloudProfiles() CloudProfileInterface {
	return newCloudProfiles(c)
}

func (c *CoreClient) ControllerDeployments() ControllerDeploymentInterface {
	return newControllerDeployments(c)
}

func (c *CoreClient) ControllerInstallations() ControllerInstallationInterface {
	return newControllerInstallations(c)
}

func (c *CoreClient) ControllerRegistrations() ControllerRegistrationInterface {
	return newControllerRegistrations(c)
}

func (c *CoreClient) ExposureClasses() ExposureClassInterface {
	return newExposureClasses(c)
}

func (c *CoreClient) Plants(namespace string) PlantInterface {
	return newPlants(c, namespace)
}

func (c *CoreClient) Projects() ProjectInterface {
	return newProjects(c)
}

func (c *CoreClient) Quotas(namespace string) QuotaInterface {
	return newQuotas(c, namespace)
}

func (c *CoreClient) SecretBindings(namespace string) SecretBindingInterface {
	return newSecretBindings(c, namespace)
}

func (c *CoreClient) Seeds() SeedInterface {
	return newSeeds(c)
}

func (c *CoreClient) Shoots(namespace string) ShootInterface {
	return newShoots(c, namespace)
}

func (c *CoreClient) ShootExtensionStatuses(namespace string) ShootExtensionStatusInterface {
	return newShootExtensionStatuses(c, namespace)
}

func (c *CoreClient) ShootLeftovers(namespace string) ShootLeftoverInterface {
	return newShootLeftovers(c, namespace)
}

func (c *CoreClient) ShootStates(namespace string) ShootStateInterface {
	return newShootStates(c, namespace)
}

// NewForConfig creates a new CoreClient for the given config.
func NewForConfig(c *rest.Config) (*CoreClient, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &CoreClient{client}, nil
}

// NewForConfigOrDie creates a new CoreClient for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *CoreClient {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new CoreClient for the given RESTClient.
func New(c rest.Interface) *CoreClient {
	return &CoreClient{c}
}

func setConfigDefaults(config *rest.Config) error {
	config.APIPath = "/apis"
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}
	if config.GroupVersion == nil || config.GroupVersion.Group != scheme.Scheme.PrioritizedVersionsForGroup("core.gardener.cloud")[0].Group {
		gv := scheme.Scheme.PrioritizedVersionsForGroup("core.gardener.cloud")[0]
		config.GroupVersion = &gv
	}
	config.NegotiatedSerializer = scheme.Codecs

	if config.QPS == 0 {
		config.QPS = 5
	}
	if config.Burst == 0 {
		config.Burst = 10
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *CoreClient) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
