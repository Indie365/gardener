// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardener_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetesscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	. "github.com/gardener/gardener/pkg/utils/gardener"
)

var _ = Describe("Secret", func() {
	var (
		ctx        = context.TODO()
		fakeClient client.Client

		secret *corev1.Secret
	)

	BeforeEach(func() {
		fakeClient = fakeclient.NewClientBuilder().WithScheme(kubernetesscheme.Scheme).Build()

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-1",
				Namespace: "test-namespace",
			},
		}
	})

	Describe("#FetchKubeconfigFromSecret", func() {
		It("should return an error because the secret does not exist", func() {
			_, err := FetchKubeconfigFromSecret(ctx, fakeClient, client.ObjectKeyFromObject(secret))
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("secrets \"secret-1\" not found")))
		})

		It("should return an error because the secret does not contain kubeconfig", func() {
			Expect(fakeClient.Create(ctx, secret)).To(Succeed())
			_, err := FetchKubeconfigFromSecret(ctx, fakeClient, client.ObjectKeyFromObject(secret))
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(("the secret's field 'kubeconfig' is either not present or empty")))
		})

		It("should return an error because the kubeconfig data is empty", func() {
			secret.Data = map[string][]byte{kubernetes.KubeConfig: {}}
			Expect(fakeClient.Create(ctx, secret)).To(Succeed())
			_, err := FetchKubeconfigFromSecret(ctx, fakeClient, client.ObjectKeyFromObject(secret))
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(("the secret's field 'kubeconfig' is either not present or empty")))
		})

		It("should return kubeconfig data if secret is prensent and contains valid kubeconfig", func() {
			secret.Data = map[string][]byte{kubernetes.KubeConfig: []byte("secret-data")}
			Expect(fakeClient.Create(ctx, secret)).To(Succeed())
			kubeConfig, err := FetchKubeconfigFromSecret(ctx, fakeClient, client.ObjectKeyFromObject(secret))
			Expect(err).ToNot(HaveOccurred())
			Expect(kubeConfig).To(Equal([]byte("secret-data")))
		})
	})
})
