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

package tokenrequestor_test

import (
	"context"
	"testing"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/logger"
	"github.com/gardener/gardener/pkg/resourcemanager/controller/tokenrequestor"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestTokenInvalidator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TokenInvalidator Integration Test Suite")
}

var (
	ctx       = context.Background()
	mgrCancel context.CancelFunc

	testEnv    *envtest.Environment
	restConfig *rest.Config
	testClient client.Client

	err error
)

var _ = BeforeSuite(func() {
	logf.SetLogger(logger.MustNewZapLogger(logger.InfoLevel, logger.FormatJSON, zap.WriteTo(GinkgoWriter)).WithName("test"))

	By("starting test environment")
	testEnv = &envtest.Environment{}
	testEnv.ControlPlane.GetAPIServer().Configure().Set("api-audiences", v1beta1constants.GardenerAudience)

	restConfig, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(restConfig).ToNot(BeNil())

	testClient, err = client.New(restConfig, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())

	By("setting up manager")
	mgr, err := manager.New(restConfig, manager.Options{MetricsBindAddress: "0"})
	Expect(err).NotTo(HaveOccurred())

	By("registering controllers and webhooks")
	Expect(tokenrequestor.AddToManagerWithOptions(mgr, tokenrequestor.ControllerConfig{
		MaxConcurrentWorkers: 5,
		TargetCluster:        mgr,
	})).To(Succeed())

	By("starting manager")
	var mgrContext context.Context
	mgrContext, mgrCancel = context.WithCancel(ctx)
	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(mgrContext)).To(Succeed())
	}()
})

var _ = AfterSuite(func() {
	By("stopping manager")
	mgrCancel()

	By("stopping test environment")
	Expect(testEnv.Stop()).To(Succeed())
})
