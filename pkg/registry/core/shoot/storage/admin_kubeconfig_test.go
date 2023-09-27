/*
Copyright 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

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

package storage

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"
	registryrest "k8s.io/apiserver/pkg/registry/rest"
	kubecorev1listers "k8s.io/client-go/listers/core/v1"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	testclock "k8s.io/utils/clock/testing"

	authenticationapi "github.com/gardener/gardener/pkg/apis/authentication"
	gardencore "github.com/gardener/gardener/pkg/apis/core"
	gardencorelisters "github.com/gardener/gardener/pkg/client/core/listers/core/internalversion"
	secretsutils "github.com/gardener/gardener/pkg/utils/secrets"
	"github.com/gardener/gardener/pkg/utils/test"
)

var _ = Describe("Admin Kubeconfig", func() {
	var (
		ctx context.Context
		obj *authenticationapi.AdminKubeconfigRequest

		shoot           *gardencore.Shoot
		caClusterSecret *corev1.Secret
		caClientSecret  *gardencore.InternalSecret

		akcREST          *AdminKubeconfigREST
		createValidation registryrest.ValidateObjectFunc

		shootGetter          *fakeGetter
		secretLister         *fakeSecretLister
		internalSecretLister *fakeInternalSecretLister

		clusterCACert = []byte("cluster-ca-cert1")

		clientCACertName = "minikubeCA"
		clientCACert     = []byte(`-----BEGIN CERTIFICATE-----
MIIDBjCCAe6gAwIBAgIBATANBgkqhkiG9w0BAQsFADAVMRMwEQYDVQQDEwptaW5p
a3ViZUNBMB4XDTIxMDMyNTE0MjczN1oXDTMxMDMyNDE0MjczN1owFTETMBEGA1UE
AxMKbWluaWt1YmVDQTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALsW
8jU6AUP1t9Wp6xOTAYhjrEPGixP+iCj9cSX5XkShpVNYNemwCqpDNOetKAAtFQMk
pco1isfuB876bNY+/bC5YCrYprzljS+EYAb+/eD/ahURnXXy09yfBrGTMvr6ti8B
L5DqlDqhHu3sekIMSedrcCs10dDckgl4lghoRSoCad3/LLqOYTPDD7VLKJup4JgS
3J1s6AxvBeeRAh94avTP+4MP4PBIewrq0CODA+rf9xfGlOrRYU5ZJnIPFCM6uEIA
xpYJl9tKuyN23kZ1BJtlenHYiR4fouXE05S0U5pw+z3WvOyNRsVQ2BViZOsVnmD6
wVrPBuZRG2NMCfEzjAECAwEAAaNhMF8wDgYDVR0PAQH/BAQDAgKkMB0GA1UdJQQW
MBQGCCsGAQUFBwMCBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQW
BBQwmHrSlJ/ytlShbhPeeMmKGnsneDANBgkqhkiG9w0BAQsFAAOCAQEABeF0WNol
mSS/hnbMFIfI8Fe90uefiO3hryBUJVBb9eaDXRRjCh9Dhj5pwxUBRyKbPHFQLQMe
YWq2Vg6vWEjDEISnthcK6n5oPIwzV5zNWek7sW3DSzFdYru8KDQReVnzBdMNIDZI
OnM7+5534rkP8/eIX58QFcVibjM34BfqNQgHW5vFXobYoIX2wfMysLZVESYQdU9P
14S7fj3Ui4IrBqElF30QUmAe6bgjBu+xsZHFaImJ+yJXuPjPEuIWoKMoiH9fDrJ0
C3KRaS8COePkaiH/NUjuIjyTXzhvJqmFbH730vABpcKi01eQMMjtRkPlWIEqUHoG
QbU6uberp2QAQA==
-----END CERTIFICATE-----`)
		clientCAKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAuxbyNToBQ/W31anrE5MBiGOsQ8aLE/6IKP1xJfleRKGlU1g1
6bAKqkM0560oAC0VAySlyjWKx+4Hzvps1j79sLlgKtimvOWNL4RgBv794P9qFRGd
dfLT3J8GsZMy+vq2LwEvkOqUOqEe7ex6QgxJ52twKzXR0NySCXiWCGhFKgJp3f8s
uo5hM8MPtUsom6ngmBLcnWzoDG8F55ECH3hq9M/7gw/g8Eh7CurQI4MD6t/3F8aU
6tFhTlkmcg8UIzq4QgDGlgmX20q7I3beRnUEm2V6cdiJHh+i5cTTlLRTmnD7Pda8
7I1GxVDYFWJk6xWeYPrBWs8G5lEbY0wJ8TOMAQIDAQABAoIBAHZMrBq78tDmLrgM
GXjnG7ECVYsFoCukZrSEjWdVpyX+kGuC+5QonJXMqUdVVlXGK+Mw6SRTds201Xsr
Hmbarc9xaD2vgL8w53WEXrQNyLrcxldMLCTIxu5aIAFo8nOA1HIkbc9UhSYNe2E2
hpf87T5H0UWBYoqO7kjO1w+53wIQL8gSCysHfO/72LwHhob1E89lyUN4bemr++eU
IgwuUxvCdiKr3in5nvbRwhLNO+K7TipKZgIj5J0SUqtiLZZ4QLNvnGbGzgoyRzoU
OgQ02qAZ8oJW0P9xal9OhWWSVRESo6D+HWMJM6Y3GdPt36oFqSnrpDh9n9L9Bf0R
SS0VXYECgYEA4DAwNPlPdiNbg8GHBouBTWW2dBGhhWvyWzZOs7Q97JW/Cs1B1ruM
42+1/ZNyNdr+buWqhDGr1QtM4UEK1nBkRHuV6kqZw8z/hKhC3r0D2AhP01yI4sGF
Bm3QFlmQJTYz9wOPFJDINkgCG2KH60p+PXBIeULA5MtYEC6hMZNe1mMCgYEA1aMf
Tlu4DIZ3Trh1ow+XtJPbwwcjcdXmMfwU+jQr3pSz6ySxXuCSBgJ9z8RbcELwDmNg
9MW8u+XMH6VSw8X6Fv1Fy7+npObz7UMW0Ij0cW/FFJ9vKOSYYET/YpFh5D0/QsWi
zLmg8iYQEjo4DlXVh8mfz0ishm0H6dVwGDp0X0sCgYAF5379hitfkyLP34Ls2zO2
lB0wBV7ZorQpTs7X0MFov7DeWfWH8DyPqNuEKCPz4yacSRQqkxxRahDGRe5BI4ig
fRi/qONP0tBP8BaCwzucrutbR66bOjmEp9O5Iva25CyOLtvP0NhVBaR4kCnAOqAE
gjaGawmlfO1+z5uTMKxovQKBgQDNJGVEZhhWlqxr//6eBLQFJ1IIdYtYnS/9YXV3
SK+zfRFDQ6m6VGSDttK+tmujYfOHrXAFuvbfautWm/bcnPfoKW5jFvdRBqDGfPyk
ZE5tuwkBI5OnLdMP5lFhgf8BHrrnUEZi1gExZNFb32HCijOPv1FgxwU70+icZmLM
MR1b/wKBgHyhTEIz3YDAG7O/y3U6JWGnxqlr8i8+FobZWMbVSGDtgRZpDcDGyhFb
AIOz/jD6sCJ6KPr1L6mJ5w4mDX1UmjCKy3Kz4xfqxPEbMvPDTL+9TWFSlAuNtHGC
lIwEl8tStnO9u1JUK4w1e+lC37zI2v5k4WMQmJcolUEMwmZjnCR/
-----END RSA PRIVATE KEY-----`)
	)

	const (
		name      = "test"
		userName  = "foo"
		namespace = "baz"
	)

	BeforeEach(func() {
		caClusterSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name + ".ca-cluster", Namespace: namespace},
			Data: map[string][]byte{
				"ca.crt": clusterCACert,
			},
		}
		caClientSecret = &gardencore.InternalSecret{
			ObjectMeta: metav1.ObjectMeta{Name: name + ".ca-client", Namespace: namespace},
			Data: map[string][]byte{
				"ca.crt": clientCACert,
				"ca.key": clientCAKey,
			},
		}

		createValidation = func(ctx context.Context, obj runtime.Object) error { return nil }
		shoot = &gardencore.Shoot{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Status: gardencore.ShootStatus{
				AdvertisedAddresses: []gardencore.ShootAdvertisedAddress{
					{
						Name: "external",
						URL:  "https://foo.bar.external:9443",
					},
					{
						Name: "internal",
						URL:  "https://foo.bar.internal:9443",
					},
				},
			},
		}

		shootGetter = &fakeGetter{obj: shoot}

		secretLister = &fakeSecretLister{obj: caClusterSecret}
		internalSecretLister = &fakeInternalSecretLister{obj: caClientSecret}

		obj = &authenticationapi.AdminKubeconfigRequest{
			Spec: authenticationapi.AdminKubeconfigRequestSpec{
				ExpirationSeconds: int64(time.Minute.Seconds() * 11),
			},
		}

		akcREST = &AdminKubeconfigREST{
			secretLister:         secretLister,
			internalSecretLister: internalSecretLister,
			shootStorage:         shootGetter,
		}

		ctx = request.WithUser(context.Background(), &user.DefaultInfo{
			Name: userName,
		})

		DeferCleanup(test.WithVar(&secretsutils.Clock, testclock.NewFakeClock(time.Unix(10, 0))))
	})

	Context("request fails", func() {
		var (
			actual runtime.Object
			err    error
		)

		AfterEach(func() {
			actual, err = akcREST.Create(ctx, name, obj, createValidation, nil)

			Expect(err).To(HaveOccurred())
			Expect(actual).To(BeNil())
		})

		It("returns an error if create validation fails", func() {
			createValidation = func(ctx context.Context, obj runtime.Object) error {
				return errors.New("some error")
			}
		})

		It("returns an error if validation fails", func() {
			obj.Spec.ExpirationSeconds = -1
		})

		It("returns an error if there is no user in the context", func() {
			ctx = context.TODO()
		})

		It("returns an error if it cannot get the ca-client secret", func() {
			internalSecretLister.err = errors.New("fake")
		})

		It("returns an error if the ca-client secret doesn't exist", func() {
			internalSecretLister.err = apierrors.NewNotFound(gardencore.Resource("internalsecrets"), caClientSecret.Name)
		})

		It("returns an error if the ca-client secret is missing the public key", func() {
			delete(caClientSecret.Data, "ca.crt")
		})

		It("returns an error if the ca-client secret is missing the private key", func() {
			delete(caClientSecret.Data, "ca.key")
		})

		It("returns an error if it cannot get the ca-cluster secret", func() {
			secretLister.err = errors.New("fake")
		})

		It("returns an error if the ca-cluster secret doesn't exist", func() {
			secretLister.err = apierrors.NewNotFound(gardencore.Resource("secrets"), caClusterSecret.Name)
		})

		It("returns an error if the ca-cluster secret is missing the public key", func() {
			delete(caClusterSecret.Data, "ca.crt")
		})

		It("returns an error if it cannot get the shoot", func() {
			shootGetter.err = errors.New("can't get shoot")
		})

		It("returns an error if it cannot convert the object to a shoot", func() {
			shootGetter.obj = &corev1.Pod{}
		})

		It("returns an error if there are no advertised addresses in shoot status", func() {
			shoot.Status.AdvertisedAddresses = nil
		})
	})

	Context("request succeeds", func() {
		It("should successfully issue admin kubeconfig", func() {
			actual, err := akcREST.Create(ctx, name, obj, nil, nil)

			Expect(err).ToNot(HaveOccurred())
			Expect(actual).ToNot(BeNil())
			Expect(actual).To(BeAssignableToTypeOf(&authenticationapi.AdminKubeconfigRequest{}))

			akcr := actual.(*authenticationapi.AdminKubeconfigRequest)

			Expect(akcr.Status.ExpirationTimestamp.Time).To(Equal(time.Unix(10, 0).Add(time.Minute * 11)))

			config := &clientcmdv1.Config{}
			Expect(runtime.DecodeInto(clientcmdlatest.Codec, akcr.Status.Kubeconfig, config)).To(Succeed())

			Expect(config.Clusters).To(ConsistOf(
				clientcmdv1.NamedCluster{
					Name: "baz--test-external",
					Cluster: clientcmdv1.Cluster{
						Server:                   "https://foo.bar.external:9443",
						CertificateAuthorityData: clusterCACert,
					},
				},
				clientcmdv1.NamedCluster{
					Name: "baz--test-internal",
					Cluster: clientcmdv1.Cluster{
						Server:                   "https://foo.bar.internal:9443",
						CertificateAuthorityData: clusterCACert,
					},
				},
			))

			Expect(config.Contexts).To(ConsistOf(
				clientcmdv1.NamedContext{
					Name: "baz--test-external",
					Context: clientcmdv1.Context{
						Cluster:  "baz--test-external",
						AuthInfo: "baz--test-external",
					},
				},
				clientcmdv1.NamedContext{
					Name: "baz--test-internal",
					Context: clientcmdv1.Context{
						Cluster:  "baz--test-internal",
						AuthInfo: "baz--test-external",
					},
				},
			))
			Expect(config.CurrentContext).To(Equal("baz--test-external"))

			Expect(config.AuthInfos).To(HaveLen(1))
			Expect(config.AuthInfos[0].Name).To(Equal("baz--test-external"))
			Expect(config.AuthInfos[0].AuthInfo.ClientCertificateData).ToNot(BeEmpty())
			Expect(config.AuthInfos[0].AuthInfo.ClientKeyData).ToNot(BeEmpty())

			certPem, _ := pem.Decode(config.AuthInfos[0].AuthInfo.ClientCertificateData)
			cert, err := x509.ParseCertificate(certPem.Bytes)
			Expect(err).ToNot(HaveOccurred())

			Expect(cert.Subject.CommonName).To(Equal(userName))
			Expect(cert.Subject.Organization).To(ConsistOf("system:masters"))
			Expect(cert.NotAfter.Unix()).To(Equal(akcr.Status.ExpirationTimestamp.Time.Unix())) // certificates do not have nano seconds in them
			Expect(cert.NotBefore.UTC()).To(Equal(time.Unix(10, 0).UTC()))
			Expect(cert.Issuer.CommonName).To(Equal(clientCACertName))
		})
	})
})

type fakeGetter struct {
	obj runtime.Object
	err error
}

func (f *fakeGetter) Get(_ context.Context, _ string, _ *metav1.GetOptions) (runtime.Object, error) {
	return f.obj, f.err
}

type fakeSecretLister struct {
	kubecorev1listers.SecretLister
	obj *corev1.Secret
	err error
}

func (f fakeSecretLister) Secrets(string) kubecorev1listers.SecretNamespaceLister {
	return f
}

func (f fakeSecretLister) Get(_ string) (*corev1.Secret, error) {
	return f.obj, f.err
}

type fakeInternalSecretLister struct {
	gardencorelisters.InternalSecretLister
	obj *gardencore.InternalSecret
	err error
}

func (f fakeInternalSecretLister) InternalSecrets(string) gardencorelisters.InternalSecretNamespaceLister {
	return f
}

func (f fakeInternalSecretLister) Get(_ string) (*gardencore.InternalSecret, error) {
	return f.obj, f.err
}
