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

package podschedulername_test

import (
	"context"
	"testing"

	"github.com/gardener/gardener/pkg/operation/botanist/component/resourcemanager"
	"github.com/gardener/gardener/pkg/resourcemanager/webhook/podschedulername"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	logzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestPodSchedulerName(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PodSchedulerName Integration Test Suite")
}

var (
	ctx       = context.Background()
	mgrCancel context.CancelFunc

	logger     logr.Logger
	testEnv    *envtest.Environment
	restConfig *rest.Config
	testClient client.Client
)

var _ = BeforeSuite(func() {
	logger = logzap.New(logzap.UseDevMode(true), logzap.WriteTo(GinkgoWriter), logzap.Level(zapcore.Level(1)))
	logf.SetLogger(logger)

	By("starting test environment")
	testEnv = &envtest.Environment{
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			MutatingWebhooks: getMutatingWebhookConfigurations(),
		},
	}
	var err error
	restConfig, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(restConfig).ToNot(BeNil())

	testClient, err = client.New(restConfig, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())

	By("setting up manager")
	mgr, err := manager.New(restConfig, manager.Options{
		Port:               testEnv.WebhookInstallOptions.LocalServingPort,
		Host:               testEnv.WebhookInstallOptions.LocalServingHost,
		CertDir:            testEnv.WebhookInstallOptions.LocalServingCertDir,
		MetricsBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	By("registering webhook")
	conf := podschedulername.WebhookConfig{SchedulerName: "bin-packing-scheduler"}
	Expect(podschedulername.AddToManagerWithOptions(mgr, conf)).To(Succeed())

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

func getMutatingWebhookConfigurations() []*admissionregistrationv1.MutatingWebhookConfiguration {
	return []*admissionregistrationv1.MutatingWebhookConfiguration{
		{
			TypeMeta: metav1.TypeMeta{
				APIVersion: admissionregistrationv1.SchemeGroupVersion.String(),
				Kind:       "MutatingWebhookConfiguration",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener-resource-manager",
			},
			Webhooks: []admissionregistrationv1.MutatingWebhook{
				resourcemanager.GetPodSchedulerNameMutatingWebhook(nil, nil, func(_ *corev1.Secret, path string) admissionregistrationv1.WebhookClientConfig {
					return admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Path: &path,
						},
					}
				}),
			},
		},
	}
}
