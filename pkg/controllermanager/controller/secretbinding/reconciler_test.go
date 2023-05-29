// Copyright 2021 SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package secretbinding

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
)

var _ = Describe("SecretBindingControl", func() {
	var (
		fakeClient client.Client
		ctx        = context.TODO()
	)

	BeforeEach(func() {
		testScheme := runtime.NewScheme()
		Expect(kubernetes.AddGardenSchemeToScheme(testScheme)).To(Succeed())

		fakeClient = fakeclient.NewClientBuilder().WithScheme(testScheme).Build()
	})

	Describe("#mayReleaseSecret", func() {
		var (
			reconciler *Reconciler

			secretBinding1Namespace = "foo"
			secretBinding1Name      = "bar"
			secretBinding2Namespace = "baz"
			secretBinding2Name      = "bax"
			secretNamespace         = "foo"
			secretName              = "bar"
		)

		BeforeEach(func() {
			reconciler = &Reconciler{Client: fakeClient}
		})

		It("should return true as no other secretbinding exists", func() {
			allowed, err := reconciler.mayReleaseSecret(ctx, secretBinding1Namespace, secretBinding1Name, secretNamespace, secretName)

			Expect(allowed).To(BeTrue())
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return true as no other secretbinding references the secret", func() {
			secretBinding := &gardencorev1beta1.SecretBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretBinding1Name,
					Namespace: secretBinding1Namespace,
				},
				SecretRef: corev1.SecretReference{
					Namespace: secretNamespace,
					Name:      secretName,
				},
			}

			Expect(fakeClient.Create(ctx, secretBinding)).To(Succeed())

			allowed, err := reconciler.mayReleaseSecret(ctx, secretBinding1Namespace, secretBinding1Name, secretNamespace, secretName)

			Expect(allowed).To(BeTrue())
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return false as another secretbinding references the secret", func() {
			secretBinding := &gardencorev1beta1.SecretBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretBinding2Name,
					Namespace: secretBinding2Namespace,
				},
				SecretRef: corev1.SecretReference{
					Namespace: secretNamespace,
					Name:      secretName,
				},
			}

			Expect(fakeClient.Create(ctx, secretBinding)).To(Succeed())

			allowed, err := reconciler.mayReleaseSecret(ctx, secretBinding1Namespace, secretBinding1Name, secretNamespace, secretName)

			Expect(allowed).To(BeFalse())
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("SecretBinding label for Secrets", func() {
		var (
			reconciler *Reconciler
			request    reconcile.Request

			secretBindingNamespace = "foo"
			secretBindingName      = "bar"

			secret        *corev1.Secret
			secretBinding *gardencorev1beta1.SecretBinding
		)

		BeforeEach(func() {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "namespace",
				},
			}

			secretBinding = &gardencorev1beta1.SecretBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretBindingName,
					Namespace: secretBindingNamespace,
				},
				SecretRef: corev1.SecretReference{
					Namespace: secret.Namespace,
					Name:      secret.Name,
				},
			}

			Expect(fakeClient.Create(ctx, secret)).To(Succeed())
			Expect(fakeClient.Create(ctx, secretBinding)).To(Succeed())

			reconciler = &Reconciler{Client: fakeClient}
			request = reconcile.Request{NamespacedName: types.NamespacedName{Namespace: secretBindingNamespace, Name: secretBindingName}}
		})

		It("should add the label to the secret referred by the secretbinding", func() {
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(secret), secret)).To(Succeed())
			Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue(
				"reference.gardener.cloud/secretbinding", "true",
			))
		})

		It("should remove the label from the secret when there are no secretbindings referring it", func() {
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(secretBinding), secretBinding)).To(Succeed())
			secretBinding.DeletionTimestamp = &metav1.Time{Time: time.Date(1, 1, 1, 1, 1, 1, 1, time.UTC)}
			// Add dummy finalizer to prevent deletion
			secretBinding.Finalizers = append(secretBinding.Finalizers, "finalizer")
			Expect(fakeClient.Update(ctx, secretBinding)).To(Succeed())

			_, err = reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(secret), secret)).To(Succeed())
			Expect(secret.ObjectMeta.Labels).To(BeEmpty())
		})
	})

	Describe("SecretBinding label for Quotas", func() {
		var (
			reconciler *Reconciler
			request    reconcile.Request

			secretBindingNamespace1 = "sb-ns-1"
			secretBindingName1      = "sb-1"
			secretBindingNamespace2 = "sb-ns-2"
			secretBindingName2      = "sb-2"
			quotaNamespace1         = "quota-ns-1"
			quotaName1              = "quota-1"
			quotaNamespace2         = "quota-ns-2"
			quotaName2              = "quota-2"

			secret                         *corev1.Secret
			secretBinding1, secretBinding2 *gardencorev1beta1.SecretBinding
			quota1, quota2                 *gardencorev1beta1.Quota
		)

		BeforeEach(func() {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "namespace",
				},
			}

			quota1 = &gardencorev1beta1.Quota{
				ObjectMeta: metav1.ObjectMeta{
					Name:      quotaName1,
					Namespace: quotaNamespace1,
				},
			}
			quota2 = &gardencorev1beta1.Quota{
				ObjectMeta: metav1.ObjectMeta{
					Name:      quotaName2,
					Namespace: quotaNamespace2,
				},
			}

			secretBinding1 = &gardencorev1beta1.SecretBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretBindingName1,
					Namespace: secretBindingNamespace1,
				},
				Quotas: []corev1.ObjectReference{
					{
						Name:      quotaName1,
						Namespace: quotaNamespace1,
					},
					{
						Name:      quotaName2,
						Namespace: quotaNamespace2,
					},
				},
				SecretRef: corev1.SecretReference{
					Name:      secret.Name,
					Namespace: secret.Namespace,
				},
			}

			secretBinding2 = &gardencorev1beta1.SecretBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:       secretBindingName2,
					Namespace:  secretBindingNamespace2,
					Finalizers: []string{"gardener"},
				},
				Quotas: []corev1.ObjectReference{
					{
						Name:      quotaName2,
						Namespace: quotaNamespace2,
					},
				},
				SecretRef: corev1.SecretReference{
					Name:      secret.Name,
					Namespace: secret.Namespace,
				},
			}

			reconciler = &Reconciler{Client: fakeClient}
			request = reconcile.Request{NamespacedName: types.NamespacedName{Namespace: secretBindingNamespace1, Name: secretBindingName1}}

			Expect(fakeClient.Create(ctx, secret)).To(Succeed())
			Expect(fakeClient.Create(ctx, quota1)).To(Succeed())
			Expect(fakeClient.Create(ctx, quota2)).To(Succeed())
			Expect(fakeClient.Create(ctx, secretBinding1)).To(Succeed())
			Expect(fakeClient.Create(ctx, secretBinding2)).To(Succeed())
		})

		It("should add the label to the quota referred by the secretbinding", func() {
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(quota1), quota1)).To(Succeed())
			Expect(quota1.ObjectMeta.Labels).To(HaveKeyWithValue(
				"reference.gardener.cloud/secretbinding", "true",
			))
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(quota2), quota2)).To(Succeed())
			Expect(quota2.ObjectMeta.Labels).To(HaveKeyWithValue(
				"reference.gardener.cloud/secretbinding", "true",
			))
		})

		It("should remove the label from the quotas when there are no secretbindings referring it", func() {
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(secretBinding1), secretBinding1)).To(Succeed())
			secretBinding1.DeletionTimestamp = &metav1.Time{Time: time.Date(1, 1, 1, 1, 1, 1, 1, time.UTC)}
			// Add dummy finalizer to prevent deletion
			secretBinding1.Finalizers = append(secretBinding1.Finalizers, "finalizer")
			Expect(fakeClient.Update(ctx, secretBinding1)).To(Succeed())

			_, err = reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(quota1), quota1)).To(Succeed())
			Expect(quota1.ObjectMeta.Labels).To(BeEmpty())

			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(quota2), quota2)).To(Succeed())
			Expect(quota2.ObjectMeta.Labels).To(HaveKeyWithValue(
				"reference.gardener.cloud/secretbinding", "true",
			))

			// Remove the finalizer from secretBinding1 so that the deletion can proceed
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(secretBinding1), secretBinding1)).To(Succeed())
			secretBinding1.Finalizers = nil
			Expect(fakeClient.Update(ctx, secretBinding1)).To(Succeed())

			// Now delete the other secretbinding referencing the quota
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(secretBinding2), secretBinding2)).To(Succeed())
			secretBinding2.DeletionTimestamp = &metav1.Time{Time: time.Date(1, 1, 1, 1, 1, 1, 1, time.UTC)}
			secretBinding2.Finalizers = append(secretBinding2.Finalizers, "finalizer")
			Expect(fakeClient.Update(ctx, secretBinding2)).To(Succeed())

			request = reconcile.Request{NamespacedName: types.NamespacedName{Namespace: secretBindingNamespace2, Name: secretBindingName2}}
			_, err = reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(quota2), quota2)).To(Succeed())
			Expect(quota2.ObjectMeta.Labels).To(BeEmpty())
		})
	})
})
