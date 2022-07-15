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

package scheduler

import (
	"context"
	"testing"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenversionedcoreclientset "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/envtest"
	"github.com/gardener/gardener/pkg/logger"
	schedulerfeatures "github.com/gardener/gardener/pkg/scheduler/features"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestScheduler(t *testing.T) {
	schedulerfeatures.RegisterFeatureGates()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scheduler Test Suite")
}

const (
	namespace    = "garden-dev"
	providerType = "provider-type"
)

var (
	ctx                 = context.Background()
	testEnv             *envtest.GardenerTestEnvironment
	restConfig          *rest.Config
	err                 error
	mgrContext          context.Context
	mgrCancel           context.CancelFunc
	testClient          client.Client
	versionedTestClient *gardenversionedcoreclientset.Clientset
)

var _ = BeforeSuite(func() {
	runtimelog.SetLogger(logger.MustNewZapLogger(logger.InfoLevel, logger.FormatJSON, zap.WriteTo(GinkgoWriter)).WithName("test"))

	By("starting test environment")
	testEnv = &envtest.GardenerTestEnvironment{
		GardenerAPIServer: &envtest.GardenerAPIServer{
			Args: []string{"--disable-admission-plugins=ResourceReferenceManager,ExtensionValidator,ShootQuotaValidator"},
		},
	}
	restConfig, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(restConfig).NotTo(BeNil())

	testClient, err = client.New(restConfig, client.Options{Scheme: kubernetes.GardenScheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(testClient).NotTo(BeNil())

	versionedTestClient, err = gardenversionedcoreclientset.NewForConfig(restConfig)
	Expect(err).ToNot(HaveOccurred())
	Expect(versionedTestClient).NotTo(BeNil())

	By("create shoot namespace")
	shootNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespace},
	}
	Expect(testClient.Create(ctx, shootNamespace)).To(Or(Succeed(), BeAlreadyExistsError()))

	project := &gardencorev1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "dev"},
		Spec: gardencorev1beta1.ProjectSpec{
			Namespace: pointer.String(namespace),
		},
	}
	Expect(testClient.Create(ctx, project)).To(Or(Succeed(), BeAlreadyExistsError()))

	By("create secret binding")
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: namespace,
		},
	}
	Expect(testClient.Create(ctx, secret)).To(Succeed())

	secretBinding := &gardencorev1beta1.SecretBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secretbinding",
			Namespace: namespace,
		},
		Provider: &gardencorev1beta1.SecretBindingProvider{
			Type: providerType,
		},
		SecretRef: corev1.SecretReference{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
	}
	Expect(testClient.Create(ctx, secretBinding)).To(Succeed())
})

var _ = AfterSuite(func() {
	By("Stopping Test Environment")
	Expect(testEnv.Stop()).To(Succeed())
})
