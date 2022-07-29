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

package podschedulername_test

import (
	"context"
	"testing"

	"github.com/gardener/gardener/cmd/utils"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/logger"
	"github.com/gardener/gardener/pkg/operation/botanist/component/gardenerkubescheduler"
	"github.com/gardener/gardener/pkg/operation/botanist/component/seedadmissioncontroller"
	"github.com/gardener/gardener/pkg/seedadmissioncontroller/webhooks/admission/extensionresources"
	"github.com/gardener/gardener/pkg/seedadmissioncontroller/webhooks/admission/podschedulername"
	"github.com/gardener/gardener/test/framework"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestPodSchedulerName(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SeedAdmissionController PodSchedulerName Webhook Integration Test Suite")
}

const testID = "gsac-podschedulername-webhook-test"

var (
	ctx        context.Context
	ctxCancel  context.CancelFunc
	err        error
	log        logr.Logger
	testEnv    *envtest.Environment
	restConfig *rest.Config
)

var _ = BeforeSuite(func() {
	utils.DeduplicateWarnings()
	ctx, ctxCancel = context.WithCancel(context.Background())

	logf.SetLogger(logger.MustNewZapLogger(logger.DebugLevel, logger.FormatJSON, zap.WriteTo(GinkgoWriter)))
	log = logf.Log.WithName("test")

	By("starting test environment")
	testEnv = &envtest.Environment{
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			ValidatingWebhooks: []*admissionregistrationv1.ValidatingWebhookConfiguration{getValidatingWebhookConfig()},
			MutatingWebhooks:   []*admissionregistrationv1.MutatingWebhookConfiguration{getMutatingWebhookConfig()},
		},
	}
	restConfig, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(restConfig).ToNot(BeNil())

	By("setting up manager")
	// setup manager in order to leverage dependency injection
	mgr, err := manager.New(restConfig, manager.Options{
		Scheme:  kubernetes.SeedScheme,
		Port:    testEnv.WebhookInstallOptions.LocalServingPort,
		Host:    testEnv.WebhookInstallOptions.LocalServingHost,
		CertDir: testEnv.WebhookInstallOptions.LocalServingCertDir,

		MetricsBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	By("setting up webhook server")
	server := mgr.GetWebhookServer()
	Expect(extensionresources.AddWebhooks(mgr)).To(Succeed())
	server.Register(podschedulername.WebhookPath, &webhook.Admission{Handler: admission.HandlerFunc(podschedulername.DefaultShootControlPlanePodsSchedulerName)})

	go func() {
		defer GinkgoRecover()
		Expect(server.Start(ctx)).To(Succeed())
	}()
})

var _ = AfterSuite(func() {
	By("running cleanup actions")
	framework.RunCleanupActions()

	By("stopping manager")
	ctxCancel()

	By("stopping test environment")
	Expect(testEnv.Stop()).To(Succeed())
})

func getValidatingWebhookConfig() *admissionregistrationv1.ValidatingWebhookConfiguration {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service-name",
			Namespace: "service-ns",
		},
	}

	return seedadmissioncontroller.GetValidatingWebhookConfig(nil, service)
}

func getMutatingWebhookConfig() *admissionregistrationv1.MutatingWebhookConfiguration {
	clientConfig := admissionregistrationv1.WebhookClientConfig{
		Service: &admissionregistrationv1.ServiceReference{
			Path: pointer.String(podschedulername.WebhookPath),
		},
	}

	return gardenerkubescheduler.GetMutatingWebhookConfig(clientConfig)
}
