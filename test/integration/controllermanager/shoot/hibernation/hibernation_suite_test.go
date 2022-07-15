// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package hibernation_test

import (
	"context"
	"testing"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/envtest"
	"github.com/gardener/gardener/pkg/logger"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	testclock "k8s.io/utils/clock/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestShootRetry(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Shoot Hibernation Controller Integration Test Suite")
}

var (
	ctx        = context.Background()
	testEnv    *envtest.GardenerTestEnvironment
	restConfig *rest.Config
	mgrCancel  context.CancelFunc

	testClient client.Client
	fakeClock  *testclock.FakeClock
)

var _ = BeforeSuite(func() {
	logf.SetLogger(logger.MustNewZapLogger(logger.InfoLevel, logger.FormatJSON, zap.WriteTo(GinkgoWriter)).WithName("test"))

	By("starting test environment")
	testEnv = &envtest.GardenerTestEnvironment{
		GardenerAPIServer: &envtest.GardenerAPIServer{
			Args: []string{"--disable-admission-plugins=ResourceReferenceManager,ExtensionValidator,ShootQuotaValidator,ShootValidator,ShootTolerationRestriction"},
		},
	}
	var err error
	restConfig, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())

	DeferCleanup(func() {
		By("stopping test environment")
		Expect(testEnv.Stop()).To(Succeed())
	})

	testClient, err = client.New(restConfig, client.Options{Scheme: kubernetes.GardenScheme})
	Expect(err).ToNot(HaveOccurred())
	fakeClock = &testclock.FakeClock{}

	By("setup manager")
	mgr, err := manager.New(restConfig, manager.Options{
		Scheme:             kubernetes.GardenScheme,
		MetricsBindAddress: "0",
	})
	Expect(err).ToNot(HaveOccurred())

	Expect(addShootHibernationControllerToManager(mgr)).To(Succeed())

	var mgrContext context.Context
	mgrContext, mgrCancel = context.WithCancel(ctx)

	By("start manager")
	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(mgrContext)).ToNot(HaveOccurred())
	}()

	DeferCleanup(func() {
		By("stopping manager")
		mgrCancel()
	})
})
