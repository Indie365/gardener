// Copyright 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package apiserver_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	. "github.com/gardener/gardener/pkg/component/apiserver"
	kubernetesutils "github.com/gardener/gardener/pkg/utils/kubernetes"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
)

var _ = Describe("AuditWebhook", func() {
	var (
		ctx        = context.TODO()
		namespace  = "some-namespace"
		kubeconfig = []byte("some-kubeconfig")

		fakeClient client.Client
	)

	BeforeEach(func() {
		fakeClient = fakeclient.NewClientBuilder().WithScheme(kubernetes.SeedScheme).Build()
	})

	Describe("#ReconcileSecretAuditWebhookKubeconfig", func() {
		It("should do nothing because config is nil", func() {
			Expect(ReconcileSecretAuditWebhookKubeconfig(ctx, fakeClient, nil, nil)).To(Succeed())

			secretList := &corev1.SecretList{}
			Expect(fakeClient.List(ctx, secretList)).To(Succeed())
			Expect(secretList.Items).To(BeEmpty())
		})

		It("should do nothing because webhook config is nil", func() {
			Expect(ReconcileSecretAuditWebhookKubeconfig(ctx, fakeClient, nil, &AuditConfig{})).To(Succeed())

			secretList := &corev1.SecretList{}
			Expect(fakeClient.List(ctx, secretList)).To(Succeed())
			Expect(secretList.Items).To(BeEmpty())
		})

		It("should do nothing because webhook kubeconfig is nil", func() {
			Expect(ReconcileSecretAuditWebhookKubeconfig(ctx, fakeClient, nil, &AuditConfig{Webhook: &AuditWebhook{}})).To(Succeed())

			secretList := &corev1.SecretList{}
			Expect(fakeClient.List(ctx, secretList)).To(Succeed())
			Expect(secretList.Items).To(BeEmpty())
		})

		It("should successfully deploy the audit webhook kubeconfig secret resource", func() {
			expectedSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "apiserver-audit-webhook-kubeconfig", Namespace: namespace},
				Data:       map[string][]byte{"kubeconfig.yaml": kubeconfig},
			}
			Expect(kubernetesutils.MakeUnique(expectedSecret)).To(Succeed())

			actualSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "apiserver-audit-webhook-kubeconfig", Namespace: namespace}}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedSecret), actualSecret)).To(BeNotFoundError())

			Expect(ReconcileSecretAuditWebhookKubeconfig(ctx, fakeClient, actualSecret, &AuditConfig{Webhook: &AuditWebhook{Kubeconfig: kubeconfig}})).To(Succeed())

			Expect(actualSecret).To(DeepEqual(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            expectedSecret.Name,
					Namespace:       expectedSecret.Namespace,
					Labels:          map[string]string{"resources.gardener.cloud/garbage-collectable-reference": "true"},
					ResourceVersion: "1",
				},
				Immutable: pointer.Bool(true),
				Data:      expectedSecret.Data,
			}))
		})
	})

	Describe("#ReconcileSecretWebhookKubeconfig", func() {
		It("should successully delpoy the kubeconfig secret and make it unique", func() {
			expectedSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "apiserver-kubeconfig", Namespace: namespace},
				Data:       map[string][]byte{"kubeconfig.yaml": kubeconfig},
			}
			Expect(kubernetesutils.MakeUnique(expectedSecret)).To(Succeed())

			actualSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "apiserver-kubeconfig", Namespace: namespace}}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(expectedSecret), actualSecret)).To(BeNotFoundError())

			Expect(ReconcileSecretAuditWebhookKubeconfig(ctx, fakeClient, actualSecret, &AuditConfig{Webhook: &AuditWebhook{Kubeconfig: kubeconfig}})).To(Succeed())

			Expect(actualSecret).To(DeepEqual(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            expectedSecret.Name,
					Namespace:       expectedSecret.Namespace,
					Labels:          map[string]string{"resources.gardener.cloud/garbage-collectable-reference": "true"},
					ResourceVersion: "1",
				},
				Immutable: pointer.Bool(true),
				Data:      expectedSecret.Data,
			}))
		})
	})
})
