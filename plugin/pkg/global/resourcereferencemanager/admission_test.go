// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package resourcereferencemanager_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/utils/ptr"

	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/apis/security"
	securityv1alpha1 "github.com/gardener/gardener/pkg/apis/security/v1alpha1"
	"github.com/gardener/gardener/pkg/apis/seedmanagement"
	seedmanagementv1alpha1 "github.com/gardener/gardener/pkg/apis/seedmanagement/v1alpha1"
	internalclientset "github.com/gardener/gardener/pkg/client/core/clientset/versioned/fake"
	gardencoreinformers "github.com/gardener/gardener/pkg/client/core/informers/externalversions"
	gardensecurityinformers "github.com/gardener/gardener/pkg/client/security/informers/externalversions"
	seedmanagementinformers "github.com/gardener/gardener/pkg/client/seedmanagement/informers/externalversions"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	. "github.com/gardener/gardener/plugin/pkg/global/resourcereferencemanager"
)

type fakeAuthorizerType struct{}

func (fakeAuthorizerType) Authorize(_ context.Context, a authorizer.Attributes) (authorizer.Decision, string, error) {
	username := a.GetUser().GetName()

	if username == "allowed-user" {
		return authorizer.DecisionAllow, "", nil
	}

	return authorizer.DecisionDeny, "", nil
}

var _ = Describe("resourcereferencemanager", func() {
	Describe("#Admit", func() {
		var (
			admissionHandler              *ReferenceManager
			kubeInformerFactory           kubeinformers.SharedInformerFactory
			kubeClient                    *fake.Clientset
			gardenCoreClient              *internalclientset.Clientset
			gardenCoreInformerFactory     gardencoreinformers.SharedInformerFactory
			seedManagementInformerFactory seedmanagementinformers.SharedInformerFactory
			gardenSecurityInformerFactory gardensecurityinformers.SharedInformerFactory
			fakeAuthorizer                fakeAuthorizerType
			scheme                        *runtime.Scheme
			dynamicClient                 *dynamicfake.FakeDynamicClient

			backupBucket gardencorev1beta1.BackupBucket
			backupEntry  gardencorev1beta1.BackupEntry
			coreShoot    core.Shoot

			namespace                  = "default"
			cloudProfileName           = "profile-1"
			seedName                   = "seed-1"
			bindingName                = "binding-1"
			credentialsBindingName     = "credentials-binding-1"
			quotaName                  = "quota-1"
			secretName                 = "secret-1"
			configMapName              = "config-map-1"
			controllerDeploymentName   = "controller-deployment-1"
			controllerRegistrationName = "controller-reg-1"
			shootName                  = "shoot-1"
			projectName                = "project-1"
			allowedUser                = "allowed-user"
			resourceVersion            = "123456"
			finalizers                 = []string{core.GardenerName}

			defaultUserName = "test-user"
			defaultUserInfo = &user.DefaultInfo{Name: defaultUserName}

			secret = corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:       secretName,
					Namespace:  namespace,
					Finalizers: finalizers,
				},
			}

			configMap = corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:            configMapName,
					Namespace:       namespace,
					Finalizers:      finalizers,
					ResourceVersion: resourceVersion,
				},
			}

			controllerDeployment = gardencorev1beta1.ControllerDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: controllerDeploymentName,
				},
			}

			cloudProfile = gardencorev1beta1.CloudProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name: cloudProfileName,
				},
			}
			controllerRegistration = core.ControllerRegistration{
				ObjectMeta: metav1.ObjectMeta{
					Name: controllerRegistrationName,
				},
				Spec: core.ControllerRegistrationSpec{
					Deployment: &core.ControllerRegistrationDeployment{
						DeploymentRefs: []core.DeploymentRef{
							{Name: controllerDeploymentName},
						},
					},
				},
			}
			project     = gardencorev1beta1.Project{}
			coreProject = core.Project{}
			seed        = gardencorev1beta1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name:       seedName,
					Finalizers: finalizers,
				}}
			quota = gardencorev1beta1.Quota{
				ObjectMeta: metav1.ObjectMeta{
					Name:      quotaName,
					Namespace: namespace,
				},
				Spec: gardencorev1beta1.QuotaSpec{
					Scope: corev1.ObjectReference{
						APIVersion: "core.gardener.cloud/v1beta1",
						Kind:       "Project",
					},
				},
			}
			coreSecretBinding = core.SecretBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:       bindingName,
					Namespace:  namespace,
					Finalizers: finalizers,
				},
				SecretRef: corev1.SecretReference{
					Name:      secretName,
					Namespace: namespace,
				},
				Quotas: []corev1.ObjectReference{
					{
						Name:      quotaName,
						Namespace: namespace,
					},
				},
			}
			secretBinding = gardencorev1beta1.SecretBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:       bindingName,
					Namespace:  namespace,
					Finalizers: finalizers,
				},
				SecretRef: corev1.SecretReference{
					Name:      secretName,
					Namespace: namespace,
				},
				Quotas: []corev1.ObjectReference{
					{
						Name:      quotaName,
						Namespace: namespace,
					},
				},
			}
			coreCredentialsBinding = security.CredentialsBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:       credentialsBindingName,
					Namespace:  namespace,
					Finalizers: finalizers,
				},
				CredentialsRef: corev1.ObjectReference{
					Name:      secretName,
					Namespace: namespace,
				},
				Quotas: []corev1.ObjectReference{
					{
						Name:      quotaName,
						Namespace: namespace,
					},
				},
			}
			credentialsBinding = securityv1alpha1.CredentialsBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:       credentialsBindingName,
					Namespace:  namespace,
					Finalizers: finalizers,
				},
				CredentialsRef: corev1.ObjectReference{
					Name:      secretName,
					Namespace: namespace,
				},
				Quotas: []corev1.ObjectReference{
					{
						Name:      quotaName,
						Namespace: namespace,
					},
				},
			}
			projectBase = core.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: projectName,
				},
			}
			shoot     = gardencorev1beta1.Shoot{}
			shootBase = core.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:       shootName,
					Namespace:  namespace,
					Finalizers: finalizers,
				},
				Spec: core.ShootSpec{
					CloudProfileName:       cloudProfileName,
					SeedName:               &seedName,
					SecretBindingName:      ptr.To(bindingName),
					CredentialsBindingName: ptr.To(credentialsBindingName),
					Kubernetes: core.Kubernetes{
						KubeAPIServer: &core.KubeAPIServerConfig{
							AuditConfig: &core.AuditConfig{
								AuditPolicy: &core.AuditPolicy{
									ConfigMapRef: &corev1.ObjectReference{
										Name: configMapName,
									},
								},
							},
						},
					},
					Resources: []core.NamedResourceReference{
						{
							Name: secretName,
							ResourceRef: autoscalingv1.CrossVersionObjectReference{
								Kind:       "Secret",
								Name:       secretName,
								APIVersion: "v1",
							},
						},
					},
				},
			}

			coreBackupBucket = core.BackupBucket{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bucket",
				},
				Spec: core.BackupBucketSpec{
					SeedName: &seedName,
					SecretRef: corev1.SecretReference{
						Name:      secretName,
						Namespace: namespace,
					},
				},
			}
			backupBucketBase = gardencorev1beta1.BackupBucket{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bucket",
				},
				Spec: gardencorev1beta1.BackupBucketSpec{
					SeedName: &seedName,
					SecretRef: corev1.SecretReference{
						Name:      secretName,
						Namespace: namespace,
					},
				},
			}

			coreBackupEntry = core.BackupEntry{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "entry",
					Namespace: namespace,
				},
				Spec: core.BackupEntrySpec{
					BucketName: backupBucketBase.Name,
					SeedName:   &seedName,
				},
			}
			backupEntryBase = gardencorev1beta1.BackupEntry{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "entry",
					Namespace: namespace,
				},
				Spec: gardencorev1beta1.BackupEntrySpec{
					BucketName: backupBucketBase.Name,
					SeedName:   &seedName,
				},
			}

			discoveryClientResources = []*metav1.APIResourceList{
				{
					GroupVersion: "v1",
					APIResources: []metav1.APIResource{
						{
							Name:       "secrets",
							Namespaced: true,
							Group:      "core",
							Version:    "v1",
							Kind:       "Secret",
						},
					},
				},
			}

			discoveryGardenClientResources = []*metav1.APIResourceList{
				{
					GroupVersion: "",
					APIResources: []metav1.APIResource{
						{
							Name:       "controllerdeployments",
							Namespaced: false,
							Group:      "core.gardener.cloud",
							Version:    "",
							Kind:       "ControllerDeployment",
						},
					},
				},
			}
		)

		BeforeEach(func() {
			admissionHandler, _ = New()
			admissionHandler.AssignReadyFunc(func() bool { return true })

			kubeInformerFactory = kubeinformers.NewSharedInformerFactory(nil, 0)
			admissionHandler.SetKubeInformerFactory(kubeInformerFactory)

			kubeClient = fake.NewSimpleClientset()
			kubeClient.Fake = testing.Fake{Resources: discoveryClientResources}
			admissionHandler.SetKubeClientset(kubeClient)

			gardenCoreClient = internalclientset.NewSimpleClientset()
			gardenCoreClient.Fake = testing.Fake{Resources: discoveryGardenClientResources}
			admissionHandler.SetCoreClientSet(gardenCoreClient)

			gardenCoreInformerFactory = gardencoreinformers.NewSharedInformerFactory(nil, 0)
			admissionHandler.SetCoreInformerFactory(gardenCoreInformerFactory)
			seedManagementInformerFactory = seedmanagementinformers.NewSharedInformerFactory(nil, 0)
			admissionHandler.SetSeedManagementInformerFactory(seedManagementInformerFactory)

			gardenSecurityInformerFactory = gardensecurityinformers.NewSharedInformerFactory(nil, 0)
			admissionHandler.SetSecurityInformerFactory(gardenSecurityInformerFactory)

			fakeAuthorizer = fakeAuthorizerType{}
			admissionHandler.SetAuthorizer(fakeAuthorizer)

			scheme = runtime.NewScheme()
			Expect(corev1.AddToScheme(scheme)).To(Succeed())

			dynamicClient = dynamicfake.NewSimpleDynamicClient(scheme, &secret)
			admissionHandler.SetDynamicClient(dynamicClient)

			MissingSecretWait = 0

			coreProject = projectBase
			coreShoot = shootBase
			backupBucket = backupBucketBase
			backupEntry = backupEntryBase

			err := gardencorev1beta1.Convert_core_Shoot_To_v1beta1_Shoot(&coreShoot, &shoot, nil)
			Expect(err).To(Succeed())

			err = gardencorev1beta1.Convert_core_Project_To_v1beta1_Project(&coreProject, &project, nil)
			Expect(err).To(Succeed())
		})

		It("should return nil because the resource is not BackupBucket and operation is delete", func() {
			attrs := admission.NewAttributesRecord(&controllerRegistration, nil, core.Kind("ControllerRegistration").WithVersion("version"), "", controllerRegistration.Name, core.Resource("controllerregistrations").WithVersion("version"), "", admission.Delete, &metav1.DeleteOptions{}, false, nil)

			err := admissionHandler.Admit(context.TODO(), attrs, nil)

			Expect(err).NotTo(HaveOccurred())

			attrs = admission.NewAttributesRecord(&shoot, nil, core.Kind("Shoot").WithVersion("version"), "", controllerRegistration.Name, core.Resource("shoots").WithVersion("version"), "", admission.Delete, &metav1.DeleteOptions{}, false, nil)

			err = admissionHandler.Admit(context.TODO(), attrs, nil)

			Expect(err).NotTo(HaveOccurred())
		})

		Context("tests for ControllerRegistration objects", func() {
			It("should accept because all referenced objects have been found (controller deployment found in cache)", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().ControllerDeployments().Informer().GetStore().Add(&controllerDeployment)).To(Succeed())

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&controllerRegistration, nil, core.Kind("ControllerRegistration").WithVersion("version"), "", controllerRegistration.Name, core.Resource("controllerregistrations").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should accept because all referenced objects have been found (controller deployment looked up live)", func() {
				gardenCoreClient.AddReactor("get", "controllerdeployments", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, &controllerDeployment, nil
				})

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&controllerRegistration, nil, core.Kind("ControllerRegistration").WithVersion("version"), "", controllerRegistration.Name, core.Resource("controllerregistrations").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject because the referenced secret does not exist", func() {
				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&controllerRegistration, nil, core.Kind("ControllerRegistration").WithVersion("version"), "", controllerRegistration.Name, core.Resource("controllerregistrations").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)
				gardenCoreClient.AddReactor("get", "controllerdeployments", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("nope, out of luck")
				})

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("nope, out of luck"))
			})
		})

		Context("tests for SecretBinding objects", func() {
			It("should accept because all referenced objects have been found (secret found in cache)", func() {
				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota)).To(Succeed())

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreSecretBinding, nil, core.Kind("SecretBinding").WithVersion("version"), coreSecretBinding.Namespace, coreSecretBinding.Name, core.Resource("secretbindings").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should accept because all referenced objects have been found (secret looked up live)", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota)).To(Succeed())
				kubeClient.AddReactor("get", "secrets", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: secret.Namespace,
							Name:      secret.Name,
						},
					}, nil
				})

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreSecretBinding, nil, core.Kind("SecretBinding").WithVersion("version"), coreSecretBinding.Namespace, coreSecretBinding.Name, core.Resource("secretbindings").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject because the referenced secret does not exist", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota)).To(Succeed())
				kubeClient.AddReactor("get", "secrets", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("nope, out of luck")
				})

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreSecretBinding, nil, core.Kind("SecretBinding").WithVersion("version"), coreSecretBinding.Namespace, coreSecretBinding.Name, core.Resource("secretbindings").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})

			It("should reject because the user is not allowed to read the referenced secret", func() {
				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota)).To(Succeed())

				user := &user.DefaultInfo{Name: "disallowed-user"}
				attrs := admission.NewAttributesRecord(&coreSecretBinding, nil, core.Kind("SecretBinding").WithVersion("version"), coreSecretBinding.Namespace, coreSecretBinding.Name, core.Resource("secretbindings").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})

			It("should reject because one of the referenced quotas does not exist", func() {
				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreSecretBinding, nil, core.Kind("SecretBinding").WithVersion("version"), coreSecretBinding.Namespace, coreSecretBinding.Name, core.Resource("secretbindings").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})

			It("should reject because the user is not allowed to read the referenced quota", func() {
				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota)).To(Succeed())

				user := &user.DefaultInfo{Name: "disallowed-user"}
				attrs := admission.NewAttributesRecord(&coreSecretBinding, nil, core.Kind("SecretBinding").WithVersion("version"), coreSecretBinding.Namespace, coreSecretBinding.Name, core.Resource("secretbindings").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})

			It("should pass because exact one quota per scope is referenced", func() {
				quotaName2 := "quota-2"
				quota2 := gardencorev1beta1.Quota{
					ObjectMeta: metav1.ObjectMeta{
						Name:      quotaName2,
						Namespace: namespace,
					},
					Spec: gardencorev1beta1.QuotaSpec{
						Scope: corev1.ObjectReference{
							APIVersion: "v1",
							Kind:       "Secret",
						},
					},
				}

				quota2Ref := corev1.ObjectReference{
					Name:      quotaName2,
					Namespace: namespace,
				}
				quotaRefList := coreSecretBinding.Quotas
				quotaRefList = append(quotaRefList, quota2Ref)
				coreSecretBinding.Quotas = quotaRefList

				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota2)).To(Succeed())

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreSecretBinding, nil, core.Kind("SecretBinding").WithVersion("version"), coreSecretBinding.Namespace, coreSecretBinding.Name, core.Resource("secretbindings").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject because more than one quota of the same scope is referenced", func() {
				quotaName2 := "quota-2"
				quota2 := gardencorev1beta1.Quota{
					ObjectMeta: metav1.ObjectMeta{
						Name:      quotaName2,
						Namespace: namespace,
					},
					Spec: gardencorev1beta1.QuotaSpec{
						Scope: corev1.ObjectReference{
							APIVersion: "core.gardener.cloud/v1beta1",
							Kind:       "Project",
						},
					},
				}

				quota2Ref := corev1.ObjectReference{
					Name:      quotaName2,
					Namespace: namespace,
				}
				quotaRefList := coreSecretBinding.Quotas
				quotaRefList = append(quotaRefList, quota2Ref)
				coreSecretBinding.Quotas = quotaRefList

				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota2)).To(Succeed())

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreSecretBinding, nil, core.Kind("SecretBinding").WithVersion("version"), coreSecretBinding.Namespace, coreSecretBinding.Name, core.Resource("secretbindings").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})
		})

		Context("tests for CredentialsBinding objects", func() {
			It("should accept because all referenced objects have been found (secret found in cache)", func() {
				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota)).To(Succeed())

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreCredentialsBinding, nil, security.Kind("CredentialsBinding").WithVersion("version"), coreCredentialsBinding.Namespace, coreCredentialsBinding.Name, security.Resource("credentialsbindings").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should accept because all referenced objects have been found (secret looked up live)", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota)).To(Succeed())
				kubeClient.AddReactor("get", "secrets", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: secret.Namespace,
							Name:      secret.Name,
						},
					}, nil
				})

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreCredentialsBinding, nil, security.Kind("CredentialsBinding").WithVersion("version"), coreCredentialsBinding.Namespace, coreCredentialsBinding.Name, security.Resource("credentialsbindings").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject because the referenced secret does not exist", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota)).To(Succeed())
				kubeClient.AddReactor("get", "secrets", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("nope, out of luck")
				})

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreCredentialsBinding, nil, security.Kind("CredentialsBinding").WithVersion("version"), coreCredentialsBinding.Namespace, coreCredentialsBinding.Name, security.Resource("credentialsbindings").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})

			It("should reject because the user is not allowed to read the referenced secret", func() {
				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota)).To(Succeed())

				user := &user.DefaultInfo{Name: "disallowed-user"}
				attrs := admission.NewAttributesRecord(&coreCredentialsBinding, nil, security.Kind("CredentialsBinding").WithVersion("version"), coreCredentialsBinding.Namespace, coreCredentialsBinding.Name, security.Resource("credentialsbindings").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})

			It("should reject because one of the referenced quotas does not exist", func() {
				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreCredentialsBinding, nil, security.Kind("CredentialsBinding").WithVersion("version"), coreCredentialsBinding.Namespace, coreCredentialsBinding.Name, security.Resource("credentialsbindings").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})

			It("should reject because the user is not allowed to read the referenced quota", func() {
				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota)).To(Succeed())

				user := &user.DefaultInfo{Name: "disallowed-user"}
				attrs := admission.NewAttributesRecord(&coreCredentialsBinding, nil, security.Kind("CredentialsBinding").WithVersion("version"), coreCredentialsBinding.Namespace, coreCredentialsBinding.Name, security.Resource("credentialsbindings").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})

			It("should pass because exact one quota per scope is referenced", func() {
				quotaName2 := "quota-2"
				quota2 := gardencorev1beta1.Quota{
					ObjectMeta: metav1.ObjectMeta{
						Name:      quotaName2,
						Namespace: namespace,
					},
					Spec: gardencorev1beta1.QuotaSpec{
						Scope: corev1.ObjectReference{
							APIVersion: "v1",
							Kind:       "Secret",
						},
					},
				}

				quota2Ref := corev1.ObjectReference{
					Name:      quotaName2,
					Namespace: namespace,
				}
				quotaRefList := coreCredentialsBinding.Quotas
				quotaRefList = append(quotaRefList, quota2Ref)
				coreCredentialsBinding.Quotas = quotaRefList

				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota2)).To(Succeed())

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreCredentialsBinding, nil, security.Kind("CredentialsBinding").WithVersion("version"), coreCredentialsBinding.Namespace, coreCredentialsBinding.Name, security.Resource("credentialsbindings").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject because more than one quota of the same scope is referenced", func() {
				quotaName2 := "quota-2"
				quota2 := gardencorev1beta1.Quota{
					ObjectMeta: metav1.ObjectMeta{
						Name:      quotaName2,
						Namespace: namespace,
					},
					Spec: gardencorev1beta1.QuotaSpec{
						Scope: corev1.ObjectReference{
							APIVersion: "core.gardener.cloud/v1beta1",
							Kind:       "Project",
						},
					},
				}

				quota2Ref := corev1.ObjectReference{
					Name:      quotaName2,
					Namespace: namespace,
				}
				quotaRefList := coreCredentialsBinding.Quotas
				quotaRefList = append(quotaRefList, quota2Ref)
				coreCredentialsBinding.Quotas = quotaRefList

				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Quotas().Informer().GetStore().Add(&quota2)).To(Succeed())

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreCredentialsBinding, nil, security.Kind("CredentialsBinding").WithVersion("version"), coreCredentialsBinding.Namespace, coreCredentialsBinding.Name, security.Resource("credentialsbindings").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})
		})

		Context("tests for Shoot objects", func() {
			It("should add the created-by annotation", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().CloudProfiles().Informer().GetStore().Add(&cloudProfile)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().SecretBindings().Informer().GetStore().Add(&secretBinding)).To(Succeed())
				Expect(gardenSecurityInformerFactory.Security().V1alpha1().CredentialsBindings().Informer().GetStore().Add(&credentialsBinding)).To(Succeed())
				Expect(kubeInformerFactory.Core().V1().ConfigMaps().Informer().GetStore().Add(&configMap)).To(Succeed())

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreShoot, nil, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				Expect(coreShoot.Annotations).NotTo(HaveKeyWithValue(v1beta1constants.GardenCreatedBy, user.Name))

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(coreShoot.Annotations).To(HaveKeyWithValue(v1beta1constants.GardenCreatedBy, user.Name))
			})

			It("should accept because all referenced objects have been found", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().CloudProfiles().Informer().GetStore().Add(&cloudProfile)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().SecretBindings().Informer().GetStore().Add(&secretBinding)).To(Succeed())
				Expect(gardenSecurityInformerFactory.Security().V1alpha1().CredentialsBindings().Informer().GetStore().Add(&credentialsBinding)).To(Succeed())
				Expect(kubeInformerFactory.Core().V1().ConfigMaps().Informer().GetStore().Add(&configMap)).To(Succeed())

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreShoot, nil, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should accept because spec was not changed", func() {
				oldShoot := coreShoot.DeepCopy()
				coreShoot.Annotations = map[string]string{
					"delete": "me",
				}
				coreShoot.Labels = map[string]string{
					"nice": "label",
				}
				coreShoot.Status.TechnicalID = "should-never-change"
				attrs := admission.NewAttributesRecord(&coreShoot, oldShoot, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject because the referenced cloud profile does not exist (create)", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().SecretBindings().Informer().GetStore().Add(&secretBinding)).To(Succeed())
				Expect(gardenSecurityInformerFactory.Security().V1alpha1().CredentialsBindings().Informer().GetStore().Add(&credentialsBinding)).To(Succeed())
				Expect(kubeInformerFactory.Core().V1().ConfigMaps().Informer().GetStore().Add(&configMap)).To(Succeed())

				attrs := admission.NewAttributesRecord(&coreShoot, nil, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})

			It("should reject because the referenced cloud profile does not exist (update)", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().SecretBindings().Informer().GetStore().Add(&secretBinding)).To(Succeed())
				Expect(kubeInformerFactory.Core().V1().ConfigMaps().Informer().GetStore().Add(&configMap)).To(Succeed())

				oldShoot := coreShoot.DeepCopy()
				oldShoot.Spec.CloudProfileName = ""

				attrs := admission.NewAttributesRecord(&coreShoot, oldShoot, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})

			It("should reject because the referenced seed does not exist (create)", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().CloudProfiles().Informer().GetStore().Add(&cloudProfile)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().SecretBindings().Informer().GetStore().Add(&secretBinding)).To(Succeed())
				Expect(kubeInformerFactory.Core().V1().ConfigMaps().Informer().GetStore().Add(&configMap)).To(Succeed())

				attrs := admission.NewAttributesRecord(&coreShoot, nil, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})

			It("should reject because the referenced seed does not exist (update)", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().CloudProfiles().Informer().GetStore().Add(&cloudProfile)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().SecretBindings().Informer().GetStore().Add(&secretBinding)).To(Succeed())
				Expect(kubeInformerFactory.Core().V1().ConfigMaps().Informer().GetStore().Add(&configMap)).To(Succeed())

				attrs := admission.NewAttributesRecord(&coreShoot, nil, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})

			It("should reject because the referenced secret binding does not exist (create)", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().CloudProfiles().Informer().GetStore().Add(&cloudProfile)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
				Expect(kubeInformerFactory.Core().V1().ConfigMaps().Informer().GetStore().Add(&configMap)).To(Succeed())

				attrs := admission.NewAttributesRecord(&coreShoot, nil, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})

			It("should reject because the referenced secret binding does not exist (update)", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().CloudProfiles().Informer().GetStore().Add(&cloudProfile)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
				Expect(kubeInformerFactory.Core().V1().ConfigMaps().Informer().GetStore().Add(&configMap)).To(Succeed())

				oldShoot := coreShoot.DeepCopy()
				oldShoot.Spec.SecretBindingName = ptr.To("")

				attrs := admission.NewAttributesRecord(&coreShoot, nil, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})

			Context("exposure class reference", func() {
				var exposureClassName = "test-exposureclass"

				BeforeEach(func() {
					Expect(gardenCoreInformerFactory.Core().V1beta1().CloudProfiles().Informer().GetStore().Add(&cloudProfile)).To(Succeed())
					Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
					Expect(gardenCoreInformerFactory.Core().V1beta1().SecretBindings().Informer().GetStore().Add(&secretBinding)).To(Succeed())
					Expect(kubeInformerFactory.Core().V1().ConfigMaps().Informer().GetStore().Add(&configMap)).To(Succeed())

					shoot.Spec.ExposureClassName = &exposureClassName
				})

				It("should reject because the referenced exposure class does not exists", func() {
					attrs := admission.NewAttributesRecord(&coreShoot, nil, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

					err := admissionHandler.Admit(context.TODO(), attrs, nil)
					Expect(err).To(HaveOccurred())
				})

				It("should accept because the referenced exposure class exists", func() {
					var exposureClass = core.ExposureClass{
						ObjectMeta: metav1.ObjectMeta{
							Name: exposureClassName,
						},
					}

					Expect(gardenCoreInformerFactory.Core().V1beta1().ExposureClasses().Informer().GetStore().Add(&exposureClass)).To(Succeed())
					attrs := admission.NewAttributesRecord(&coreShoot, nil, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

					err := admissionHandler.Admit(context.TODO(), attrs, nil)
					Expect(err).To(HaveOccurred())
				})
			})

			It("should reject because the referenced config map does not exist", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().CloudProfiles().Informer().GetStore().Add(&cloudProfile)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().SecretBindings().Informer().GetStore().Add(&secretBinding)).To(Succeed())

				attrs := admission.NewAttributesRecord(&coreShoot, nil, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})

			It("should reject because the user is not allowed to read the referenced resource (create)", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().CloudProfiles().Informer().GetStore().Add(&cloudProfile)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().SecretBindings().Informer().GetStore().Add(&secretBinding)).To(Succeed())
				Expect(gardenSecurityInformerFactory.Security().V1alpha1().CredentialsBindings().Informer().GetStore().Add(&credentialsBinding)).To(Succeed())
				Expect(kubeInformerFactory.Core().V1().ConfigMaps().Informer().GetStore().Add(&configMap)).To(Succeed())

				user := &user.DefaultInfo{Name: "disallowed-user"}
				attrs := admission.NewAttributesRecord(&coreShoot, nil, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(MatchError("shoots.core.gardener.cloud \"shoot-1\" is forbidden: shoot cannot reference a resource you are not allowed to read"))
			})

			It("should reject because the user is not allowed to read the referenced resource (update)", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().CloudProfiles().Informer().GetStore().Add(&cloudProfile)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().SecretBindings().Informer().GetStore().Add(&secretBinding)).To(Succeed())
				Expect(kubeInformerFactory.Core().V1().ConfigMaps().Informer().GetStore().Add(&configMap)).To(Succeed())

				coreShoot.Spec.Resources = append(coreShoot.Spec.Resources, core.NamedResourceReference{
					Name: "foo",
					ResourceRef: autoscalingv1.CrossVersionObjectReference{
						Kind:       "Secret",
						Name:       "foo",
						APIVersion: "v1",
					},
				})

				oldShoot := coreShoot.DeepCopy()
				coreShoot.Spec.Resources[0] = oldShoot.Spec.Resources[1]
				coreShoot.Spec.Resources[1] = oldShoot.Spec.Resources[0]

				user := &user.DefaultInfo{Name: "disallowed-user"}
				attrs := admission.NewAttributesRecord(&coreShoot, oldShoot, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(MatchError("shoots.core.gardener.cloud \"shoot-1\" is forbidden: shoot cannot reference a resource you are not allowed to read"))
			})

			It("should allow because the user is not allowed to read the referenced resource but resource spec hasn't changed", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().CloudProfiles().Informer().GetStore().Add(&cloudProfile)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().SecretBindings().Informer().GetStore().Add(&secretBinding)).To(Succeed())
				Expect(kubeInformerFactory.Core().V1().ConfigMaps().Informer().GetStore().Add(&configMap)).To(Succeed())

				oldShoot := coreShoot.DeepCopy()

				user := &user.DefaultInfo{Name: "disallowed-user"}
				attrs := admission.NewAttributesRecord(&coreShoot, oldShoot, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(Not(HaveOccurred()))
			})

			It("should reject because the referenced resource does not exist", func() {
				dynamicClient = dynamicfake.NewSimpleDynamicClient(scheme)
				admissionHandler.SetDynamicClient(dynamicClient)
				Expect(gardenCoreInformerFactory.Core().V1beta1().CloudProfiles().Informer().GetStore().Add(&cloudProfile)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().SecretBindings().Informer().GetStore().Add(&secretBinding)).To(Succeed())
				Expect(gardenSecurityInformerFactory.Security().V1alpha1().CredentialsBindings().Informer().GetStore().Add(&credentialsBinding)).To(Succeed())
				Expect(kubeInformerFactory.Core().V1().ConfigMaps().Informer().GetStore().Add(&configMap)).To(Succeed())

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreShoot, nil, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(MatchError(ContainSubstring("failed to resolve shoot resource reference")))
			})

			It("should reject because the referenced DNS provider secret does not exist (create)", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().CloudProfiles().Informer().GetStore().Add(&cloudProfile)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().SecretBindings().Informer().GetStore().Add(&secretBinding)).To(Succeed())
				Expect(gardenSecurityInformerFactory.Security().V1alpha1().CredentialsBindings().Informer().GetStore().Add(&credentialsBinding)).To(Succeed())
				Expect(kubeInformerFactory.Core().V1().ConfigMaps().Informer().GetStore().Add(&configMap)).To(Succeed())

				coreShoot.Spec.DNS = &core.DNS{
					Providers: []core.DNSProvider{
						{SecretName: ptr.To("foo")},
					},
				}

				kubeClient.AddReactor("get", "secrets", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("nope, out of luck")
				})

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreShoot, nil, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(MatchError(ContainSubstring("failed to reference DNS provider secret")))
			})

			It("should reject because the referenced DNS provider secret does not exist (update)", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().CloudProfiles().Informer().GetStore().Add(&cloudProfile)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().SecretBindings().Informer().GetStore().Add(&secretBinding)).To(Succeed())
				Expect(kubeInformerFactory.Core().V1().ConfigMaps().Informer().GetStore().Add(&configMap)).To(Succeed())

				oldShoot := coreShoot.DeepCopy()

				coreShoot.Spec.DNS = &core.DNS{
					Providers: []core.DNSProvider{
						{SecretName: ptr.To("foo")},
					},
				}

				kubeClient.AddReactor("get", "secrets", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("nope, out of luck")
				})

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreShoot, oldShoot, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(MatchError(ContainSubstring("failed to reference DNS provider secret")))
			})

			It("should pass because the referenced DNS provider secret does not exist but shoot has deletion timestamp", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().CloudProfiles().Informer().GetStore().Add(&cloudProfile)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().SecretBindings().Informer().GetStore().Add(&secretBinding)).To(Succeed())
				Expect(kubeInformerFactory.Core().V1().ConfigMaps().Informer().GetStore().Add(&configMap)).To(Succeed())

				oldShoot := coreShoot.DeepCopy()

				coreShoot.Spec.DNS = &core.DNS{
					Providers: []core.DNSProvider{
						{SecretName: ptr.To("foo")},
					},
				}

				now := metav1.Now()
				coreShoot.DeletionTimestamp = &now

				kubeClient.AddReactor("get", "secrets", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("nope, out of luck")
				})

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreShoot, oldShoot, core.Kind("Shoot").WithVersion("version"), coreShoot.Namespace, coreShoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(Not(HaveOccurred()))
			})

			It("should pass because the referenced DNS provider secret exists", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().CloudProfiles().Informer().GetStore().Add(&cloudProfile)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().SecretBindings().Informer().GetStore().Add(&secretBinding)).To(Succeed())
				Expect(gardenSecurityInformerFactory.Security().V1alpha1().CredentialsBindings().Informer().GetStore().Add(&credentialsBinding)).To(Succeed())
				Expect(kubeInformerFactory.Core().V1().ConfigMaps().Informer().GetStore().Add(&configMap)).To(Succeed())

				coreShoot.Spec.DNS = &core.DNS{
					Providers: []core.DNSProvider{
						{SecretName: ptr.To("foo")},
					},
				}

				kubeClient.AddReactor("get", "secrets", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, nil, nil
				})

				user := &user.DefaultInfo{Name: allowedUser}
				attrs := admission.NewAttributesRecord(&coreShoot, nil, core.Kind("Shoot").WithVersion("version"), shoot.Namespace, shoot.Name, core.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, user)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(Not(HaveOccurred()))
			})
		})

		Context("tests for BackupBucket objects", func() {
			It("should reject if the referred Seed is not found", func() {
				attrs := admission.NewAttributesRecord(&coreBackupBucket, nil, core.Kind("BackupBucket").WithVersion("version"), "", coreBackupBucket.Name, core.Resource("backupBuckets").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(BeForbiddenError())
				Expect(err).To(MatchError(ContainSubstring("backupBuckets.core.gardener.cloud %q is forbidden: seed.core.gardener.cloud %q not found", coreBackupBucket.Name, seed.Name)))
			})

			It("should reject if the referred Secret is not found", func() {
				kubeClient.AddReactor("get", "secrets", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, nil, fmt.Errorf("secret not found")
				})
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())

				attrs := admission.NewAttributesRecord(&coreBackupBucket, nil, core.Kind("BackupBucket").WithVersion("version"), "", coreBackupBucket.Name, core.Resource("backupBuckets").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(BeForbiddenError())
				Expect(err).To(MatchError(ContainSubstring("secret not found")))
			})

			It("should accept (direct secret lookup)", func() {
				kubeClient.AddReactor("get", "secrets", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: secret.Namespace,
							Name:      secret.Name,
						},
					}, nil
				})
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())

				attrs := admission.NewAttributesRecord(&coreBackupBucket, nil, core.Kind("BackupBucket").WithVersion("version"), "", coreBackupBucket.Name, core.Resource("backupBuckets").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should accept (secret found in cache)", func() {
				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())

				attrs := admission.NewAttributesRecord(&coreBackupBucket, nil, core.Kind("BackupBucket").WithVersion("version"), "", coreBackupBucket.Name, core.Resource("backupBuckets").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should accept deletion if no backupEntries are referencing it", func() {
				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())

				gardenCoreClient.AddReactor("list", "backupentries", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, &gardencorev1beta1.BackupEntryList{Items: []gardencorev1beta1.BackupEntry{}}, nil
				})

				attrs := admission.NewAttributesRecord(nil, nil, core.Kind("BackupBucket").WithVersion("version"), "", coreBackupBucket.Name, core.Resource("backupBuckets").WithVersion("version"), "", admission.Delete, &metav1.DeleteOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject deletion if there are backupEntries referencing it", func() {
				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())

				backupEntry2 := backupEntryBase.DeepCopy()
				backupEntry2.Name = "another-name"
				backupEntry2.Namespace = "another-namespace"

				gardenCoreClient.AddReactor("list", "backupentries", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, &gardencorev1beta1.BackupEntryList{Items: []gardencorev1beta1.BackupEntry{backupEntry, *backupEntry2}}, nil
				})

				attrs := admission.NewAttributesRecord(nil, nil, core.Kind("BackupBucket").WithVersion("version"), "", backupBucket.Name, core.Resource("backupBuckets").WithVersion("version"), "", admission.Delete, &metav1.DeleteOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(BeForbiddenError())
				Expect(err).To(MatchError(ContainSubstring("backupBuckets.core.gardener.cloud %q is forbidden: cannot delete BackupBucket because BackupEntries are still referencing it, backupEntryNames: %s/%s,%s/%s", backupBucket.Name, backupEntry.Namespace, backupEntry.Name, backupEntry2.Namespace, backupEntry2.Name)))
			})

			It("should reject deletion if the listing fails", func() {
				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())

				gardenCoreClient.AddReactor("list", "backupentries", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("error")
				})

				attrs := admission.NewAttributesRecord(nil, nil, core.Kind("BackupBucket").WithVersion("version"), "", coreBackupBucket.Name, core.Resource("backupBuckets").WithVersion("version"), "", admission.Delete, &metav1.DeleteOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
			})

			It("should forbid multiple BackupBuckets deletion if a BackupEntry referencing any of the BackupBuckets exists", func() {
				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())

				backupBucket2 := backupBucketBase.DeepCopy()
				backupBucket2.Name = "different-backupBucket"

				backupEntry.Spec.BucketName = backupBucket2.Name
				backupEntry2 := backupEntryBase.DeepCopy()
				backupEntry2.Name = "another-name"
				backupEntry2.Namespace = "another-namespace"
				backupEntry2.Spec.BucketName = backupBucket2.Name

				gardenCoreClient.AddReactor("list", "backupentries", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, &gardencorev1beta1.BackupEntryList{Items: []gardencorev1beta1.BackupEntry{backupEntry, *backupEntry2}}, nil
				})

				gardenCoreClient.AddReactor("list", "backupbuckets", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, &gardencorev1beta1.BackupBucketList{Items: []gardencorev1beta1.BackupBucket{*backupBucket2, backupBucket}}, nil
				})

				attrs := admission.NewAttributesRecord(nil, nil, core.Kind("BackupBucket").WithVersion("version"), "", "", core.Resource("backupBuckets").WithVersion("version"), "", admission.Delete, &metav1.DeleteOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(BeForbiddenError())
				Expect(err).To(MatchError(ContainSubstring("backupBuckets.core.gardener.cloud %q is forbidden: cannot delete BackupBucket because BackupEntries are still referencing it, backupEntryNames: %s/%s,%s/%s", backupBucket2.Name, backupEntry.Namespace, backupEntry.Name, backupEntry2.Namespace, backupEntry2.Name)))
			})

			It("should allow multiple BackupBuckets deletion if no BackupEntry exists referencing any of the BackupBuckets", func() {
				Expect(kubeInformerFactory.Core().V1().Secrets().Informer().GetStore().Add(&secret)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())

				backupBucket2 := backupBucketBase.DeepCopy()
				backupBucket2.Name = "different-backupBucket"

				backupEntry.Spec.BucketName = "some-other-bucket"
				backupEntry2 := backupEntryBase.DeepCopy()
				backupEntry2.Name = "another-name"
				backupEntry2.Namespace = "another-namespace"
				backupEntry2.Spec.BucketName = "some-other-bucket"

				gardenCoreClient.AddReactor("list", "backupentries", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, &gardencorev1beta1.BackupEntryList{Items: []gardencorev1beta1.BackupEntry{}}, nil
				})

				gardenCoreClient.AddReactor("list", "backupbuckets", func(_ testing.Action) (bool, runtime.Object, error) {
					return true, &gardencorev1beta1.BackupBucketList{Items: []gardencorev1beta1.BackupBucket{*backupBucket2, backupBucket}}, nil
				})

				attrs := admission.NewAttributesRecord(nil, nil, core.Kind("BackupBucket").WithVersion("version"), "", "", core.Resource("backupBuckets").WithVersion("version"), "", admission.Delete, &metav1.DeleteOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("tests for BackupEntry objects", func() {
			It("should reject if the referred Seed is not found", func() {
				attrs := admission.NewAttributesRecord(&coreBackupEntry, nil, core.Kind("BackupEntry").WithVersion("version"), coreBackupEntry.Namespace, coreBackupEntry.Name, core.Resource("backupEntries").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(BeForbiddenError())
				Expect(err).To(MatchError(ContainSubstring("backupEntries.core.gardener.cloud %q is forbidden: seed.core.gardener.cloud %q not found", coreBackupEntry.Name, seed.Name)))
			})

			It("should reject if the referred BackupBucket is not found", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
				attrs := admission.NewAttributesRecord(&coreBackupEntry, nil, core.Kind("BackupEntry").WithVersion("version"), coreBackupEntry.Namespace, coreBackupEntry.Name, core.Resource("backupEntries").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(BeForbiddenError())
				Expect(err).To(MatchError(ContainSubstring("backupEntries.core.gardener.cloud %q is forbidden: backupbucket.core.gardener.cloud %q not found", coreBackupEntry.Name, coreBackupBucket.Name)))
			})

			It("should accept if the referred resources exist", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Seeds().Informer().GetStore().Add(&seed)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().BackupBuckets().Informer().GetStore().Add(&backupBucket)).To(Succeed())
				attrs := admission.NewAttributesRecord(&coreBackupEntry, nil, core.Kind("BackupEntry").WithVersion("version"), coreBackupEntry.Namespace, coreBackupEntry.Name, core.Resource("backupEntries").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("tests for Project objects", func() {
			It("should set the created-by field", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Projects().Informer().GetStore().Add(&project)).To(Succeed())

				attrs := admission.NewAttributesRecord(&coreProject, nil, core.Kind("Project").WithVersion("version"), coreProject.Namespace, coreProject.Name, core.Resource("projects").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(coreProject.Spec.CreatedBy).To(Equal(&rbacv1.Subject{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     rbacv1.UserKind,
					Name:     defaultUserName,
				}))
			})

			It("should set the owner field (member with owner role found)", func() {
				projectCopy := project.DeepCopy()
				coreProjectCopy := coreProject.DeepCopy()
				ownerMember := &rbacv1.Subject{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     rbacv1.UserKind,
					Name:     "owner",
				}
				projectCopy.Name = "foo"
				projectCopy.Spec.Members = []gardencorev1beta1.ProjectMember{
					{
						Subject: *ownerMember,
						Roles:   []string{core.ProjectMemberOwner},
					},
				}
				coreProjectCopy.Name = "foo"
				coreProjectCopy.Spec.Members = []core.ProjectMember{
					{
						Subject: *ownerMember,
						Roles:   []string{core.ProjectMemberOwner},
					},
				}

				Expect(gardenCoreInformerFactory.Core().V1beta1().Projects().Informer().GetStore().Add(projectCopy)).To(Succeed())

				attrs := admission.NewAttributesRecord(coreProjectCopy, nil, core.Kind("Project").WithVersion("version"), coreProjectCopy.Namespace, coreProjectCopy.Name, core.Resource("projects").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(coreProjectCopy.Spec.Owner).To(Equal(ownerMember))
				Expect(coreProjectCopy.Spec.CreatedBy).To(Equal(&rbacv1.Subject{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     rbacv1.UserKind,
					Name:     defaultUserName,
				}))
			})

			It("should set the owner field", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Projects().Informer().GetStore().Add(&project)).To(Succeed())

				attrs := admission.NewAttributesRecord(&coreProject, nil, core.Kind("Project").WithVersion("version"), coreProject.Namespace, coreProject.Name, core.Resource("projects").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(coreProject.Spec.Owner).To(Equal(&rbacv1.Subject{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     rbacv1.UserKind,
					Name:     defaultUserName,
				}))
			})

			It("should add the owner to members", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Projects().Informer().GetStore().Add(&project)).To(Succeed())

				attrs := admission.NewAttributesRecord(&coreProject, nil, core.Kind("Project").WithVersion("version"), coreProject.Namespace, coreProject.Name, core.Resource("projects").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(coreProject.Spec.Members).To(ContainElement(Equal(core.ProjectMember{
					Subject: rbacv1.Subject{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     rbacv1.UserKind,
						Name:     defaultUserName,
					},
					Roles: []string{
						core.ProjectMemberAdmin,
						core.ProjectMemberOwner,
					},
				})))
			})

			It("should allow specifying a namespace which is not in use (create)", func() {
				project.Spec.Namespace = ptr.To("garden-foo")
				projectCopy := project.DeepCopy()
				projectCopy.Name = "project-2"
				projectCopy.Spec.Namespace = ptr.To("garden-bar")
				Expect(gardenCoreInformerFactory.Core().V1beta1().Projects().Informer().GetStore().Add(projectCopy)).To(Succeed())

				coreProject.Spec.Namespace = ptr.To("garden-foo")
				attrs := admission.NewAttributesRecord(&coreProject, nil, core.Kind("Project").WithVersion("version"), coreProject.Namespace, coreProject.Name, core.Resource("projects").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(Not(HaveOccurred()))
			})

			It("should allow specifying a namespace which is not in use (update)", func() {
				projectOld := project.DeepCopy()
				projectCopy := project.DeepCopy()
				project.Spec.Namespace = ptr.To("garden-foo")
				projectCopy.Name = "project-2"
				projectCopy.Spec.Namespace = ptr.To("garden-bar")
				Expect(gardenCoreInformerFactory.Core().V1beta1().Projects().Informer().GetStore().Add(projectOld)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Projects().Informer().GetStore().Add(projectCopy)).To(Succeed())

				coreProjectOld := coreProject.DeepCopy()
				coreProject.Spec.Namespace = ptr.To("garden-foo")
				attrs := admission.NewAttributesRecord(&coreProject, coreProjectOld, core.Kind("Project").WithVersion("version"), coreProject.Namespace, coreProject.Name, core.Resource("projects").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(Not(HaveOccurred()))
			})

			It("should allow specifying multiple projects w/o a namespace", func() {
				projectCopy := project.DeepCopy()
				projectCopy.Name = "project-2"
				Expect(gardenCoreInformerFactory.Core().V1beta1().Projects().Informer().GetStore().Add(projectCopy)).To(Succeed())

				attrs := admission.NewAttributesRecord(&coreProject, nil, core.Kind("Project").WithVersion("version"), coreProject.Namespace, coreProject.Name, core.Resource("projects").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(Not(HaveOccurred()))
			})

			It("should forbid specifying a namespace which is already used by another project (create)", func() {
				projectCopy := project.DeepCopy()
				projectCopy.Spec.Namespace = ptr.To("garden-foo")
				projectCopy.Name = "project-2"
				Expect(gardenCoreInformerFactory.Core().V1beta1().Projects().Informer().GetStore().Add(projectCopy)).To(Succeed())

				coreProject.Spec.Namespace = ptr.To("garden-foo")
				attrs := admission.NewAttributesRecord(&coreProject, nil, core.Kind("Project").WithVersion("version"), coreProject.Namespace, coreProject.Name, core.Resource("projects").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"ErrStatus": MatchFields(IgnoreExtras, Fields{
						"Code":    Equal(int32(http.StatusForbidden)),
						"Message": ContainSubstring("namespace \"garden-foo\" is already used by another project"),
					}),
				})))
			})

			It("should forbid specifying a namespace which is already used by another project (update)", func() {
				projectOld := project.DeepCopy()
				project.Spec.Namespace = ptr.To("garden-foo")
				projectCopy := project.DeepCopy()
				projectCopy.Name = "project-2"
				Expect(gardenCoreInformerFactory.Core().V1beta1().Projects().Informer().GetStore().Add(projectOld)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Projects().Informer().GetStore().Add(projectCopy)).To(Succeed())

				coreProjectOld := coreProject
				coreProject.Spec.Namespace = ptr.To("garden-foo")
				attrs := admission.NewAttributesRecord(&coreProject, &coreProjectOld, core.Kind("Project").WithVersion("version"), coreProject.Namespace, coreProject.Name, core.Resource("projects").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"ErrStatus": MatchFields(IgnoreExtras, Fields{
						"Code":    Equal(int32(http.StatusForbidden)),
						"Message": ContainSubstring("namespace \"garden-foo\" is already used by another project"),
					}),
				})))
			})
		})

		Context("CloudProfile - Update Kubernetes versions", func() {
			versions := []core.ExpirableVersion{
				{Version: "1.24.2"},
				{Version: "1.24.1"},
				{Version: "1.24.0"},
			}
			shootOne := shoot.DeepCopy()
			shootOne.Name = "shoot-One"
			shootOne.Spec.Provider.Type = "aws"
			shootOne.Spec.CloudProfileName = "aws-profile"
			shootOne.Spec.Kubernetes.Version = "1.24.2"

			shootTwo := shootOne.DeepCopy()
			shootTwo.Name = "shoot-Two"
			shootTwo.Spec.Kubernetes.Version = "1.24.1"
			var (
				cloudProfile = core.CloudProfile{
					ObjectMeta: metav1.ObjectMeta{Name: "aws-profile"},
					Spec: core.CloudProfileSpec{
						Kubernetes: core.KubernetesSettings{
							Versions: versions,
						},
					},
				}
			)
			It("should accept if no kubernetes version has been removed", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootOne)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootTwo)).To(Succeed())

				attrs := admission.NewAttributesRecord(&cloudProfile, &cloudProfile, core.Kind("CloudProfile").WithVersion("version"), "", cloudProfile.Name, core.Resource("CloudProfile").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should accept removal of kubernetes version that is not in use by any shoot", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootOne)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootTwo)).To(Succeed())

				cloudProfileNew := cloudProfile
				cloudProfileNew.Spec = core.CloudProfileSpec{
					Kubernetes: core.KubernetesSettings{
						Versions: []core.ExpirableVersion{
							{Version: "1.24.2"},
							{Version: "1.24.1"},
						},
					},
				}

				attrs := admission.NewAttributesRecord(&cloudProfileNew, &cloudProfile, core.Kind("CloudProfile").WithVersion("version"), "", cloudProfile.Name, core.Resource("CloudProfile").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject removal of kubernetes versions that are still in use by shoots", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootOne)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootTwo)).To(Succeed())

				cloudProfileNew := cloudProfile
				cloudProfileNew.Spec = core.CloudProfileSpec{
					Kubernetes: core.KubernetesSettings{
						Versions: []core.ExpirableVersion{
							{Version: "1.24.2"},
						},
					},
				}

				attrs := admission.NewAttributesRecord(&cloudProfileNew, &cloudProfile, core.Kind("CloudProfile").WithVersion("version"), "", cloudProfile.Name, core.Resource("CloudProfile").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("1.24.1"))
			})

			It("should accept removal of kubernetes version that is still in use by a shoot that is being deleted", func() {
				t := metav1.Now()
				shootTwoDeleted := shootTwo.DeepCopy()
				shootTwoDeleted.DeletionTimestamp = &t

				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootOne)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootTwoDeleted)).To(Succeed())

				cloudProfileNew := cloudProfile
				cloudProfileNew.Spec = core.CloudProfileSpec{
					Kubernetes: core.KubernetesSettings{
						Versions: []core.ExpirableVersion{
							{Version: "1.24.2"},
						},
					},
				}

				attrs := admission.NewAttributesRecord(&cloudProfileNew, &cloudProfile, core.Kind("CloudProfile").WithVersion("version"), "", cloudProfile.Name, core.Resource("CloudProfile").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("CloudProfile - Update Machine image versions", func() {
			versions := []core.MachineImageVersion{
				{
					ExpirableVersion: core.ExpirableVersion{
						Version: "1.17.3",
					},
				},
				{
					ExpirableVersion: core.ExpirableVersion{
						Version: "1.17.2",
					},
				},
				{
					ExpirableVersion: core.ExpirableVersion{
						Version: "1.17.1",
					},
				},
				{
					ExpirableVersion: core.ExpirableVersion{
						Version: "1.17.0",
					},
				},
				{
					ExpirableVersion: core.ExpirableVersion{
						Version: "1.16.0",
					},
				},
			}
			shootOne := shoot.DeepCopy()
			shootOne.Name = "shoot-One"
			shootOne.Spec.Provider.Type = "aws"
			shootOne.Spec.CloudProfileName = "aws-profile"
			shootOne.Spec.Provider.Workers = []gardencorev1beta1.Worker{
				{
					Name: "coreos-worker",
					Machine: gardencorev1beta1.Machine{
						Image: &gardencorev1beta1.ShootMachineImage{
							Name:    "coreos",
							Version: ptr.To("1.17.3"),
						},
					},
				},
			}

			shootTwo := shootOne.DeepCopy()
			shootTwo.Name = "shoot-Two"
			shootTwo.Spec.Provider.Workers = []gardencorev1beta1.Worker{
				{
					Name: "ubuntu-worker-1",
					Machine: gardencorev1beta1.Machine{
						Image: &gardencorev1beta1.ShootMachineImage{
							Name:    "ubuntu",
							Version: ptr.To("1.17.2"),
						},
					},
				},
				{
					Name: "ubuntu-worker-2",
					Machine: gardencorev1beta1.Machine{
						Image: &gardencorev1beta1.ShootMachineImage{
							Name:    "ubuntu",
							Version: ptr.To("1.17.1"),
						},
					},
				},
			}

			var (
				cloudProfile = core.CloudProfile{
					ObjectMeta: metav1.ObjectMeta{Name: "aws-profile"},
					Spec: core.CloudProfileSpec{
						MachineImages: []core.MachineImage{
							{
								Name:     "coreos",
								Versions: versions,
							},
							{
								Name:     "ubuntu",
								Versions: versions,
							},
						},
					},
				}
			)
			It("should accept if no machine image version has been removed", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootOne)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootTwo)).To(Succeed())

				attrs := admission.NewAttributesRecord(&cloudProfile, &cloudProfile, core.Kind("CloudProfile").WithVersion("version"), "", cloudProfile.Name, core.Resource("CloudProfile").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})
			It("should accept removal of a machine version that is not in use by any shoot", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootOne)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootTwo)).To(Succeed())

				newVersions := []core.MachineImageVersion{
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.17.3",
						},
					},
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.17.2",
						},
					},
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.17.1",
						},
					},
				}

				// new cloud profile has version 1.17.0 and 1.16.0 removed. These are not in use by any worker of any shoot
				cloudProfileNew := cloudProfile
				cloudProfileNew.Spec = core.CloudProfileSpec{
					MachineImages: []core.MachineImage{
						{
							Name:     "coreos",
							Versions: newVersions,
						},
						{
							Name:     "ubuntu",
							Versions: newVersions,
						},
					},
				}

				attrs := admission.NewAttributesRecord(&cloudProfileNew, &cloudProfile, core.Kind("CloudProfile").WithVersion("version"), "", cloudProfile.Name, core.Resource("CloudProfile").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject removal of a machine image version that is in use by a shoot", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootOne)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootTwo)).To(Succeed())

				newVersions := []core.MachineImageVersion{
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.17.3",
						},
					},
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.17.0",
						},
					},
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.16.0",
						},
					},
				}

				// new cloud profile has version 1.17.1 removed.
				cloudProfileNew := cloudProfile
				cloudProfileNew.Spec = core.CloudProfileSpec{
					MachineImages: []core.MachineImage{
						{
							Name:     "coreos",
							Versions: newVersions,
						},
						{
							Name:     "ubuntu",
							Versions: newVersions,
						},
					},
				}

				attrs := admission.NewAttributesRecord(&cloudProfileNew, &cloudProfile, core.Kind("CloudProfile").WithVersion("version"), "", cloudProfile.Name, core.Resource("CloudProfile").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("1.17.2"))
				Expect(err.Error()).To(ContainSubstring("1.17.1"))
				Expect(err.Error()).To(ContainSubstring(shootTwo.Spec.Provider.Workers[0].Machine.Image.Name))
				Expect(err.Error()).To(ContainSubstring(shootTwo.Spec.Provider.Workers[1].Machine.Image.Name))
			})

			It("should accept removal of a machine image version that is in use by a shoot that is being deleted", func() {
				t := metav1.Now()
				shootTwoDeleted := shootTwo.DeepCopy()
				shootTwoDeleted.DeletionTimestamp = &t

				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootOne)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootTwoDeleted)).To(Succeed())

				newVersions := []core.MachineImageVersion{
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.17.3",
						},
					},
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.17.0",
						},
					},
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.16.0",
						},
					},
				}

				// new cloud profile has version 1.17.1 removed.
				cloudProfileNew := cloudProfile
				cloudProfileNew.Spec = core.CloudProfileSpec{
					MachineImages: []core.MachineImage{
						{
							Name:     "coreos",
							Versions: newVersions,
						},
						{
							Name:     "ubuntu",
							Versions: newVersions,
						},
					},
				}

				attrs := admission.NewAttributesRecord(&cloudProfileNew, &cloudProfile, core.Kind("CloudProfile").WithVersion("version"), "", cloudProfile.Name, core.Resource("CloudProfile").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).NotTo(HaveOccurred())
			})

			// for existing Gardener installations
			// Shoot uses Machine Image that does not exist in the CloudProfile and uses machine image version that should be removed
			It("should reject deletion of image version", func() {
				s := shootTwo.DeepCopy()
				s.Spec.Provider.Workers = []gardencorev1beta1.Worker{
					{
						Name: "dummy-worker-1",
						Machine: gardencorev1beta1.Machine{
							Image: &gardencorev1beta1.ShootMachineImage{
								Name: "dummy",
								// version does not matter for this test, as image does not exist
								Version: ptr.To("1.1.1"),
							},
						},
					},
					{
						Name: "ubuntu-worker",
						Machine: gardencorev1beta1.Machine{
							Image: &gardencorev1beta1.ShootMachineImage{
								Name:    "ubuntu",
								Version: ptr.To("1.17.2"),
							},
						},
					},
				}
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootOne)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(s)).To(Succeed())

				newVersions := []core.MachineImageVersion{
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.17.3",
						},
					},
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.17.0",
						},
					},
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.16.0",
						},
					},
				}

				// new cloud profile has version 1.17.1 removed.
				cloudProfileNew := cloudProfile
				cloudProfileNew.Spec = core.CloudProfileSpec{
					MachineImages: []core.MachineImage{
						{
							Name:     "coreos",
							Versions: newVersions,
						},
						{
							Name:     "ubuntu",
							Versions: newVersions,
						},
					},
				}

				attrs := admission.NewAttributesRecord(&cloudProfileNew, &cloudProfile, core.Kind("CloudProfile").WithVersion("version"), "", cloudProfile.Name, core.Resource("CloudProfile").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("1.17.2"))
				Expect(err.Error()).To(ContainSubstring(s.Spec.Provider.Workers[1].Name))
			})

			It("should reject removal of a whole machine image which versions are in use by a shoot", func() {
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootOne)).To(Succeed())
				Expect(gardenCoreInformerFactory.Core().V1beta1().Shoots().Informer().GetStore().Add(shootTwo)).To(Succeed())

				newVersions := []core.MachineImageVersion{
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.17.3",
						},
					},
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.17.2",
						},
					},
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.17.1",
						},
					},
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.17.0",
						},
					},
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.16.0",
						},
					},
				}

				// new cloud profile has ubuntu image removed.
				cloudProfileNew := cloudProfile
				cloudProfileNew.Spec = core.CloudProfileSpec{
					MachineImages: []core.MachineImage{
						{
							Name:     "coreos",
							Versions: newVersions,
						},
					},
				}

				attrs := admission.NewAttributesRecord(&cloudProfileNew, &cloudProfile, core.Kind("CloudProfile").WithVersion("version"), "", cloudProfile.Name, core.Resource("CloudProfile").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, defaultUserInfo)

				err := admissionHandler.Admit(context.TODO(), attrs, nil)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("1.17.2"))
				Expect(err.Error()).To(ContainSubstring("1.17.1"))
				Expect(err.Error()).To(ContainSubstring(shootTwo.Spec.Provider.Workers[0].Name))
				Expect(err.Error()).To(ContainSubstring(shootTwo.Spec.Provider.Workers[1].Name))
			})
		})

		Context("tests for Gardenlet objects", func() {
			var (
				gardenlet   *seedmanagement.Gardenlet
				managedSeed *seedmanagementv1alpha1.ManagedSeed
			)

			BeforeEach(func() {
				gardenlet = &seedmanagement.Gardenlet{ObjectMeta: metav1.ObjectMeta{Name: "some-seed", Namespace: "some-namespace"}}
				managedSeed = &seedmanagementv1alpha1.ManagedSeed{ObjectMeta: gardenlet.ObjectMeta}
			})

			It("should accept because there is no managed seed with the same name", func() {
				attrs := admission.NewAttributesRecord(gardenlet, nil, seedmanagement.Kind("Gardenlet").WithVersion("version"), gardenlet.Namespace, gardenlet.Name, seedmanagement.Resource("gardenlets").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, &user.DefaultInfo{Name: allowedUser})

				Expect(admissionHandler.Admit(context.TODO(), attrs, nil)).To(Succeed())
			})

			It("should forbid because there is a managed seed with the same name", func() {
				Expect(seedManagementInformerFactory.Seedmanagement().V1alpha1().ManagedSeeds().Informer().GetStore().Add(managedSeed)).To(Succeed())

				attrs := admission.NewAttributesRecord(gardenlet, nil, seedmanagement.Kind("Gardenlet").WithVersion("version"), gardenlet.Namespace, gardenlet.Name, seedmanagement.Resource("gardenlets").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, &user.DefaultInfo{Name: allowedUser})

				Expect(admissionHandler.Admit(context.TODO(), attrs, nil)).To(MatchError(ContainSubstring("there is already a ManagedSeed object with the same name")))
			})
		})

		Context("tests for ManagedSeed objects", func() {
			var (
				managedSeed *seedmanagement.ManagedSeed
				gardenlet   *seedmanagementv1alpha1.Gardenlet
			)

			BeforeEach(func() {
				managedSeed = &seedmanagement.ManagedSeed{ObjectMeta: metav1.ObjectMeta{Name: "some-seed", Namespace: "some-namespace"}}
				gardenlet = &seedmanagementv1alpha1.Gardenlet{ObjectMeta: managedSeed.ObjectMeta}
			})

			It("should accept because there is no gardenlet with the same name", func() {
				attrs := admission.NewAttributesRecord(managedSeed, nil, seedmanagement.Kind("ManagedSeed").WithVersion("version"), gardenlet.Namespace, gardenlet.Name, seedmanagement.Resource("gardenlets").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, &user.DefaultInfo{Name: allowedUser})

				Expect(admissionHandler.Admit(context.TODO(), attrs, nil)).To(Succeed())
			})

			It("should forbid because there is a gardenlet with the same name", func() {
				Expect(seedManagementInformerFactory.Seedmanagement().V1alpha1().Gardenlets().Informer().GetStore().Add(gardenlet)).To(Succeed())

				attrs := admission.NewAttributesRecord(managedSeed, nil, seedmanagement.Kind("ManagedSeed").WithVersion("version"), managedSeed.Namespace, managedSeed.Name, seedmanagement.Resource("managedseeds").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, &user.DefaultInfo{Name: allowedUser})

				Expect(admissionHandler.Admit(context.TODO(), attrs, nil)).To(MatchError(ContainSubstring("there is already a Gardenlet object with the same name")))
			})
		})
	})

	Describe("#Register", func() {
		It("should register the plugin", func() {
			plugins := admission.NewPlugins()
			Register(plugins)

			registered := plugins.Registered()
			Expect(registered).To(HaveLen(1))
			Expect(registered).To(ContainElement("ResourceReferenceManager"))
		})
	})

	Describe("#New", func() {
		It("should only handle CREATE, UPDATE and DELETE operations", func() {
			rm, err := New()

			Expect(err).ToNot(HaveOccurred())
			Expect(rm.Handles(admission.Create)).To(BeTrue())
			Expect(rm.Handles(admission.Update)).To(BeTrue())
			Expect(rm.Handles(admission.Connect)).NotTo(BeTrue())
			Expect(rm.Handles(admission.Delete)).To(BeTrue())
		})
	})

	Describe("#ValidateInitialization", func() {
		It("should not return error if everything is set", func() {
			rm, _ := New()

			internalGardenClient := &internalclientset.Clientset{}
			rm.SetCoreClientSet(internalGardenClient)

			rm.SetCoreInformerFactory(gardencoreinformers.NewSharedInformerFactory(nil, 0))
			rm.SetSeedManagementInformerFactory(seedmanagementinformers.NewSharedInformerFactory(nil, 0))

			fakeAuthorizer := fakeAuthorizerType{}
			rm.SetAuthorizer(fakeAuthorizer)

			kubeInformerFactory := kubeinformers.NewSharedInformerFactory(nil, 0)
			rm.SetKubeInformerFactory(kubeInformerFactory)

			gardenSecurityInformerFactory := gardensecurityinformers.NewSharedInformerFactory(nil, 0)
			rm.SetSecurityInformerFactory(gardenSecurityInformerFactory)
			Expect(rm.ValidateInitialization()).To(Succeed())
		})
	})
})
