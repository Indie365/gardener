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

package validation_test

import (
	testutils "github.com/gardener/gardener/landscaper/common/test-utils"
	"github.com/gardener/gardener/landscaper/pkg/controlplane/apis/imports"
	. "github.com/gardener/gardener/landscaper/pkg/controlplane/apis/imports/validation"
	landscaperv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("ValidateImports", func() {
	var (
		landscaperImport *imports.Imports
		caEtcdTLS        = testutils.GenerateCACertificate("gardener.cloud:system:etcd-virtual")
		caEtcdString     = string(caEtcdTLS.CertificatePEM)
		etcdClientCert   = testutils.GenerateClientCertificate(&caEtcdTLS)
		etcdCertString   = string(etcdClientCert.CertificatePEM)
		etcdKeyString    = string(etcdClientCert.PrivateKeyPEM)
	)

	BeforeEach(func() {
		landscaperImport = &imports.Imports{
			RuntimeCluster: landscaperv1alpha1.Target{Spec: landscaperv1alpha1.TargetSpec{
				Configuration: landscaperv1alpha1.AnyJSON{RawMessage: []byte("dummy")},
			}},
			InternalDomain: imports.DNS{
				Domain:      "default.domain",
				Provider:    "abc",
				Credentials: []byte("credentials"),
			},
			Etcd: imports.Etcd{
				EtcdUrl:        "virtual-garden-etcd-main-client.garden.svc:2379",
				EtcdCABundle:   &caEtcdString,
				EtcdClientCert: &etcdCertString,
				EtcdClientKey:  &etcdKeyString,
			},
			GardenerAPIServer: &imports.GardenerAPIServer{
				ComponentConfiguration: &imports.APIServerComponentConfiguration{
					CA:  &imports.CA{},
					TLS: &imports.TLSServer{},
				},
			},
		}

	})

	Describe("#Validate Mandatory Configuration", func() {
		It("should allow valid configurations", func() {
			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(BeEmpty())
		})

		It("should fail: missing DNS config", func() {
			landscaperImport.InternalDomain = imports.DNS{}
			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(3))
		})

		It("should fail: etcd Url has length 0", func() {
			landscaperImport.EtcdUrl = ""
			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})
	})

	Context("Etcd", func() {
		It("should forbid invalid TLS configuration - etcd url is not set", func() {
			landscaperImport.EtcdUrl = ""
			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("etcdUrl"),
				})),
			))
		})

		It("should forbid invalid TLS configuration - etcd CA is invalid", func() {
			landscaperImport.EtcdCABundle = pointer.String("invalid")
			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("etcdCABundle"),
				})),
			))
		})

		It("should forbid invalid TLS configuration - etcd client certificate is invalid", func() {
			landscaperImport.EtcdClientCert = pointer.String("invalid")
			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("etcdClientCert"),
				})),
			))
		})

		It("should forbid invalid TLS configuration - etcd client key is invalid", func() {
			landscaperImport.EtcdClientKey = pointer.String("invalid")
			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("etcdClientKey"),
				})),
			))
		})

		It("should forbid providing both the etcd secret reference as well as supply the certificate values directly", func() {
			landscaperImport.EtcdSecretRef = &corev1.SecretReference{}
			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("etcdSecretRef"),
					"Detail": ContainSubstring("cannot configure both the secret reference as well as supply the certificate values directly"),
				})),
			))
		})
	})

	Describe("Validate Optional Configuration", func() {
		It("validate that the virtual garden kubeconfig is provided when virtual garden is enabled", func() {
			landscaperImport.VirtualGarden = &imports.VirtualGarden{Enabled: true}
			landscaperImport.VirtualGardenCluster = nil
			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})

		It("validate default domain - missing credentials", func() {
			landscaperImport.DefaultDomains = append(landscaperImport.DefaultDomains, imports.DNS{
				Domain:   "xyz",
				Provider: "sdsd",
			})
			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})

		It("validate alerting: no auth type", func() {
			landscaperImport.Alerting = []imports.Alerting{
				{
					AuthType: "",
				},
			}

			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})

		It("validate alerting: auth type: none requires url", func() {
			landscaperImport.Alerting = []imports.Alerting{
				{
					AuthType: "node",
					Url:      nil,
				},
			}

			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})

		It("validate alerting: smtp: To email address not set", func() {
			landscaperImport.Alerting = []imports.Alerting{
				{
					AuthType:         "smtp",
					ToEmailAddress:   pointer.String(""),
					FromEmailAddress: pointer.String("xy"),
					Smarthost:        pointer.String("xy"),
					AuthUsername:     pointer.String("xy"),
					AuthIdentity:     pointer.String("xy"),
					AuthPassword:     pointer.String("xy"),
				},
			}

			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})

		It("validate alerting: smtp: From email address not set", func() {
			landscaperImport.Alerting = []imports.Alerting{
				{
					AuthType:         "smtp",
					ToEmailAddress:   pointer.String("xy"),
					FromEmailAddress: pointer.String(""),
					Smarthost:        pointer.String("xy"),
					AuthUsername:     pointer.String("xy"),
					AuthIdentity:     pointer.String("xy"),
					AuthPassword:     pointer.String("xy"),
				},
			}

			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})

		It("validate alerting: smtp: smarthost not set", func() {
			landscaperImport.Alerting = []imports.Alerting{
				{
					AuthType:         "smtp",
					ToEmailAddress:   pointer.String("xy"),
					FromEmailAddress: pointer.String("xy"),
					Smarthost:        pointer.String(""),
					AuthUsername:     pointer.String("xy"),
					AuthIdentity:     pointer.String("xy"),
					AuthPassword:     pointer.String("xy"),
				},
			}

			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})

		It("validate alerting: smtp: auth username not set", func() {
			landscaperImport.Alerting = []imports.Alerting{
				{
					AuthType:         "smtp",
					ToEmailAddress:   pointer.String("xy"),
					FromEmailAddress: pointer.String("xy"),
					Smarthost:        pointer.String("xy"),
					AuthUsername:     pointer.String(""),
					AuthIdentity:     pointer.String("xy"),
					AuthPassword:     pointer.String("xy"),
				},
			}

			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})

		It("validate alerting: smtp: auth identity not set", func() {
			landscaperImport.Alerting = []imports.Alerting{
				{
					AuthType:         "smtp",
					ToEmailAddress:   pointer.String("xy"),
					FromEmailAddress: pointer.String("xy"),
					Smarthost:        pointer.String("xy"),
					AuthUsername:     pointer.String("xy"),
					AuthIdentity:     pointer.String(""),
					AuthPassword:     pointer.String("xy"),
				},
			}

			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})

		It("validate alerting: smtp: auth password not set", func() {
			landscaperImport.Alerting = []imports.Alerting{
				{
					AuthType:         "smtp",
					ToEmailAddress:   pointer.String("xy"),
					FromEmailAddress: pointer.String("xy"),
					Smarthost:        pointer.String("xy"),
					AuthUsername:     pointer.String("xy"),
					AuthIdentity:     pointer.String("xy"),
					AuthPassword:     pointer.String(""),
				},
			}

			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})

		It("validate alerting: basic: url not set", func() {
			landscaperImport.Alerting = []imports.Alerting{
				{
					AuthType: "basic",
					Username: pointer.String("user"),
					Password: pointer.String("pw"),
				},
			}

			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})

		It("validate alerting: basic: user not set", func() {
			landscaperImport.Alerting = []imports.Alerting{
				{
					AuthType: "basic",
					Url:      pointer.String("xy"),
					Password: pointer.String("pw"),
				},
			}

			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})

		It("validate alerting: basic: password not set", func() {
			landscaperImport.Alerting = []imports.Alerting{
				{
					AuthType: "basic",
					Url:      pointer.String("xy"),
					Username: pointer.String("user"),
				},
			}

			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})

		It("validate alerting: certificate: url not set", func() {
			landscaperImport.Alerting = []imports.Alerting{
				{
					AuthType: "certificate",
					Url:      pointer.String(""),
					CaCert:   pointer.String("x"),
					TlsCert:  pointer.String("x"),
					TlsKey:   pointer.String("x"),
				},
			}

			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})

		It("validate alerting: certificate: CaCert not set", func() {
			landscaperImport.Alerting = []imports.Alerting{
				{
					AuthType: "certificate",
					Url:      pointer.String("url"),
					CaCert:   pointer.String(""),
					TlsCert:  pointer.String("x"),
					TlsKey:   pointer.String("x"),
				},
			}

			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})

		It("validate alerting: certificate: TlsCert not set", func() {
			landscaperImport.Alerting = []imports.Alerting{
				{
					AuthType: "certificate",
					Url:      pointer.String("url"),
					CaCert:   pointer.String("x"),
					TlsCert:  pointer.String(""),
					TlsKey:   pointer.String("x"),
				},
			}

			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})

		It("validate alerting: certificate: TlsKey not set", func() {
			landscaperImport.Alerting = []imports.Alerting{
				{
					AuthType: "certificate",
					Url:      pointer.String("url"),
					CaCert:   pointer.String("x"),
					TlsCert:  pointer.String("x"),
					TlsKey:   pointer.String(""),
				},
			}

			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})

		It("validate that the seed restriction is enabled when the seed authorizer option is enabled", func() {
			landscaperImport.Rbac = &imports.Rbac{SeedAuthorizer: &imports.SeedAuthorizer{
				Enabled: pointer.Bool(true),
			}}

			landscaperImport.GardenerAdmissionController = &imports.GardenerAdmissionController{
				Enabled:         true,
				SeedRestriction: &imports.SeedRestriction{Enabled: false},
			}

			errorList := ValidateLandscaperImports(landscaperImport)
			Expect(errorList).To(HaveLen(1))
		})
	})

})
