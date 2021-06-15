// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package envtest

import (
	"fmt"
	"io/ioutil"
	"os"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("gardener.test-env")

// GardenerTestEnvironment wraps envtest.Environment and additionally starts, registers and stops an instance of
// gardener-apiserver in order to work with gardener resources in the test.
type GardenerTestEnvironment struct {
	*envtest.Environment

	// certDir contains the kube-apiserver certs (generated by controller-runtime's pkg/envtest) and the front-proxy
	// certs for the API server aggregation layer
	certDir          string
	aggregatorConfig AggregatorConfig

	// GardenerAPIServer knows how to start, register and stop a temporary gardener-apiserver instance.
	GardenerAPIServer *GardenerAPIServer
}

// Start starts the underlying envtest.Environment and the GardenerAPIServer.
func (e *GardenerTestEnvironment) Start() (*rest.Config, error) {
	if e.Environment == nil {
		e.Environment = &envtest.Environment{}
	}
	kubeAPIServer := e.Environment.ControlPlane.GetAPIServer()

	// manage k-api cert dir by ourselves, we will add aggregator certs to it
	var err error
	e.certDir, err = ioutil.TempDir("", "k8s_test_framework_")
	if err != nil {
		return nil, err
	}
	kubeAPIServer.CertDir = e.certDir

	// configure kube-aggregator
	if err := e.aggregatorConfig.ConfigureAPIServerArgs(e.certDir, kubeAPIServer.Configure()); err != nil {
		return nil, err
	}

	// start kube control plane
	log.V(1).Info("starting envtest control plane")
	adminRestConfig, err := e.Environment.Start()
	if err != nil {
		return nil, err
	}

	// TODO: respect Environment.UseExistingCluster / USE_EXISTING_CLUSTER for running tests against a local setup
	// instead of running gardener-apiserver and spinning up a test environment.
	if e.GardenerAPIServer == nil {
		e.GardenerAPIServer = &GardenerAPIServer{}
	}

	// add gardener-apiserver user
	gardenerAPIServerUser, err := e.Environment.ControlPlane.AddUser(envtest.User{
		Name: "gardener-apiserver",
		// TODO: bootstrap gardener RBAC and bind to ClusterRole/gardener.cloud:system:apiserver
		Groups: []string{"system:masters"},
	}, &rest.Config{
		// gotta go fast during tests -- we don't really care about overwhelming our test API server
		QPS:   1000.0,
		Burst: 2000.0,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to provision gardener-apiserver user: %w", err)
	}
	e.GardenerAPIServer.user = gardenerAPIServerUser

	// default GardenerAPIServer settings to the envtest ControlPlane settings
	if e.GardenerAPIServer.StartTimeout.Milliseconds() == 0 {
		e.GardenerAPIServer.StartTimeout = e.ControlPlaneStartTimeout
	}
	if e.GardenerAPIServer.StopTimeout.Milliseconds() == 0 {
		e.GardenerAPIServer.StopTimeout = e.ControlPlaneStopTimeout
	}
	if e.GardenerAPIServer.Out == nil && e.AttachControlPlaneOutput {
		e.GardenerAPIServer.Out = os.Stdout
	}
	if e.GardenerAPIServer.Err == nil && e.AttachControlPlaneOutput {
		e.GardenerAPIServer.Err = os.Stderr
	}
	// reuse etcd from envtest ControlPlane if not overwritten
	if e.GardenerAPIServer.EtcdURL == nil {
		e.GardenerAPIServer.EtcdURL = e.Environment.ControlPlane.Etcd.URL
	}

	if err := e.GardenerAPIServer.Start(); err != nil {
		return nil, fmt.Errorf("failed to start gardener-apiserver: %w", err)
	}

	return adminRestConfig, nil
}

// Stop stops the underlying envtest.Environment and the GardenerAPIServer.
func (e *GardenerTestEnvironment) Stop() error {
	if err := e.GardenerAPIServer.Stop(); err != nil {
		return err
	}
	if err := e.Environment.Stop(); err != nil {
		return err
	}

	if e.certDir != "" {
		return os.RemoveAll(e.certDir)
	}
	return nil
}
