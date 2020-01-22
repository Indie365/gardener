// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package quotavalidator_test

import (
	"context"
	"time"

	"github.com/gardener/gardener/pkg/apis/garden"
	gardeninformers "github.com/gardener/gardener/pkg/client/garden/informers/internalversion"
	"github.com/gardener/gardener/pkg/logger"
	"github.com/gardener/gardener/pkg/operation/common"
	. "github.com/gardener/gardener/plugin/pkg/shoot/quotavalidator"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
)

var _ = Describe("quotavalidator", func() {
	Describe("#Admit", func() {
		var (
			admissionHandler      *QuotaValidator
			gardenInformerFactory gardeninformers.SharedInformerFactory
			shoot                 garden.Shoot
			oldShoot              garden.Shoot
			secretBinding         garden.SecretBinding
			quotaProject          garden.Quota
			quotaSecret           garden.Quota
			cloudProfile          garden.CloudProfile
			namespace             string = "test"
			trialNamespace        string = "trial"
			machineTypeName       string = "n1-standard-2"
			machineTypeName2      string = "machtype2"
			volumeTypeName        string = "pd-standard"

			cloudProfileBase = garden.CloudProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name: "profile",
				},
				Spec: garden.CloudProfileSpec{
					MachineTypes: []garden.MachineType{
						{
							Name:   machineTypeName,
							CPU:    resource.MustParse("2"),
							GPU:    resource.MustParse("0"),
							Memory: resource.MustParse("5Gi"),
						},
						{
							Name:   machineTypeName2,
							CPU:    resource.MustParse("2"),
							GPU:    resource.MustParse("0"),
							Memory: resource.MustParse("5Gi"),
							Storage: &garden.MachineTypeStorage{
								Class: garden.VolumeClassStandard,
							},
						},
					},
					VolumeTypes: []garden.VolumeType{
						{
							Name:  volumeTypeName,
							Class: "standard",
						},
					},
				},
			}

			quotaProjectLifetime = 1
			quotaProjectBase     = garden.Quota{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: trialNamespace,
					Name:      "project-quota",
				},
				Spec: garden.QuotaSpec{
					ClusterLifetimeDays: &quotaProjectLifetime,
					Scope: corev1.ObjectReference{
						APIVersion: "core.gardener.cloud/v1beta1",
						Kind:       "Project",
					},
					Metrics: corev1.ResourceList{
						garden.QuotaMetricCPU:             resource.MustParse("2"),
						garden.QuotaMetricGPU:             resource.MustParse("0"),
						garden.QuotaMetricMemory:          resource.MustParse("5Gi"),
						garden.QuotaMetricStorageStandard: resource.MustParse("30Gi"),
						garden.QuotaMetricStoragePremium:  resource.MustParse("0Gi"),
						garden.QuotaMetricLoadbalancer:    resource.MustParse("2"),
					},
				},
			}

			quotaSecretLifetime = 7
			quotaSecretBase     = garden.Quota{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: trialNamespace,
					Name:      "secret-quota",
				},
				Spec: garden.QuotaSpec{
					ClusterLifetimeDays: &quotaSecretLifetime,
					Scope: corev1.ObjectReference{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					Metrics: corev1.ResourceList{
						garden.QuotaMetricCPU:             resource.MustParse("4"),
						garden.QuotaMetricGPU:             resource.MustParse("0"),
						garden.QuotaMetricMemory:          resource.MustParse("10Gi"),
						garden.QuotaMetricStorageStandard: resource.MustParse("60Gi"),
						garden.QuotaMetricStoragePremium:  resource.MustParse("0Gi"),
						garden.QuotaMetricLoadbalancer:    resource.MustParse("4"),
					},
				},
			}

			secretBindingBase = garden.SecretBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-binding",
				},
				Quotas: []corev1.ObjectReference{
					{
						Namespace: trialNamespace,
						Name:      "project-quota",
					},
					{
						Namespace: trialNamespace,
						Name:      "secret-quota",
					},
				},
			}

			workersBase = []garden.Worker{
				{
					Name: "test-worker-1",
					Machine: garden.Machine{
						Type: machineTypeName,
					},
					Maximum: 1,
					Minimum: 1,
					Volume: &garden.Volume{
						Size: "30Gi",
						Type: &volumeTypeName,
					},
				},
			}

			workersBase2 = []garden.Worker{
				{
					Name: "test-worker-1",
					Machine: garden.Machine{
						Type: machineTypeName,
					},
					Maximum: 1,
					Minimum: 1,
					Volume: &garden.Volume{
						Size: "30Gi",
						Type: &volumeTypeName,
					},
				},
				{
					Name: "test-worker-2",
					Machine: garden.Machine{
						Type: machineTypeName,
					},
					Maximum: 1,
					Minimum: 1,
					Volume: &garden.Volume{
						Size: "30Gi",
						Type: &volumeTypeName,
					},
				},
			}

			shootBase = garden.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-shoot",
				},
				Spec: garden.ShootSpec{
					CloudProfileName:  "profile",
					SecretBindingName: "test-binding",
					Provider: garden.Provider{
						Workers: workersBase,
					},
					Kubernetes: garden.Kubernetes{
						Version: "1.0.1",
					},
					Addons: &garden.Addons{
						NginxIngress: &garden.NginxIngress{
							Addon: garden.Addon{
								Enabled: true,
							},
						},
					},
				},
			}
		)

		BeforeSuite(func() {
			logger.Logger = logger.NewLogger("")
		})

		BeforeEach(func() {
			shoot = *shootBase.DeepCopy()
			cloudProfile = *cloudProfileBase.DeepCopy()
			secretBinding = *secretBindingBase.DeepCopy()
			quotaProject = *quotaProjectBase.DeepCopy()
			quotaSecret = *quotaSecretBase.DeepCopy()

			admissionHandler, _ = New()
			admissionHandler.AssignReadyFunc(func() bool { return true })
			gardenInformerFactory = gardeninformers.NewSharedInformerFactory(nil, 0)
			admissionHandler.SetInternalGardenInformerFactory(gardenInformerFactory)
			gardenInformerFactory.Garden().InternalVersion().CloudProfiles().Informer().GetStore().Add(&cloudProfile)
			gardenInformerFactory.Garden().InternalVersion().Quotas().Informer().GetStore().Add(&quotaProject)
			gardenInformerFactory.Garden().InternalVersion().Quotas().Informer().GetStore().Add(&quotaSecret)
			gardenInformerFactory.Garden().InternalVersion().SecretBindings().Informer().GetStore().Add(&secretBindingBase)
		})

		Context("tests for Shoots, which have at least one Quota referenced", func() {
			It("should pass because all quotas limits are sufficient", func() {
				attrs := admission.NewAttributesRecord(&shoot, nil, garden.Kind("Shoot").WithVersion("version"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, nil)

				err := admissionHandler.Validate(context.TODO(), attrs, nil)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should fail because the limits of at least one quota are exceeded", func() {
				shoot.Spec.Provider.Workers[0].Maximum = 2
				attrs := admission.NewAttributesRecord(&shoot, nil, garden.Kind("Shoot").WithVersion("version"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, nil)

				err := admissionHandler.Validate(context.TODO(), attrs, nil)
				Expect(err).To(HaveOccurred())
			})

			It("should fail because other shoots exhaust quota limits", func() {
				shoot2 := *shoot.DeepCopy()
				shoot2.Name = "test-shoot-2"
				gardenInformerFactory.Garden().InternalVersion().Shoots().Informer().GetStore().Add(&shoot2)

				attrs := admission.NewAttributesRecord(&shoot, nil, garden.Kind("Shoot").WithVersion("version"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, nil)

				err := admissionHandler.Validate(context.TODO(), attrs, nil)
				Expect(err).To(HaveOccurred())
			})

			It("should fail because shoot with 2 workers exhaust quota limits", func() {
				shoot.Spec.Provider.Workers = workersBase2
				attrs := admission.NewAttributesRecord(&shoot, nil, garden.Kind("Shoot").WithVersion("version"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, nil)

				err := admissionHandler.Validate(context.TODO(), attrs, nil)
				Expect(err).To(HaveOccurred())
			})

			It("should pass because can update non worker property although quota is exceeded", func() {
				oldShoot = *shoot.DeepCopy()
				quotaProject.Spec.Metrics[garden.QuotaMetricCPU] = resource.MustParse("1")
				gardenInformerFactory.Garden().InternalVersion().Quotas().Informer().GetStore().Add(&quotaProject)

				shoot.Spec.Kubernetes.Version = "1.1.1"
				attrs := admission.NewAttributesRecord(&shoot, &oldShoot, garden.Kind("Shoot").WithVersion("version"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, nil)

				err := admissionHandler.Validate(context.TODO(), attrs, nil)
				Expect(err).NotTo(HaveOccurred())
			})

			Context("machine type in cloud profile defines storage", func() {
				It("should pass because quota is large enough", func() {
					shoot2 := *shoot.DeepCopy()
					shoot2.Spec.Provider.Workers[0].Machine.Type = machineTypeName2
					shoot2.Spec.Provider.Workers[0].Volume.Size = "19Gi"

					quotaProject.Spec.Metrics[garden.QuotaMetricStorageStandard] = resource.MustParse("20Gi")

					attrs := admission.NewAttributesRecord(&shoot2, nil, garden.Kind("Shoot").WithVersion("version"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, nil)

					err := admissionHandler.Validate(context.TODO(), attrs, nil)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should fail because quota is not large enough", func() {
					shoot2 := *shoot.DeepCopy()
					shoot2.Spec.Provider.Workers[0].Machine.Type = machineTypeName2
					shoot2.Spec.Provider.Workers[0].Volume.Size = "21Gi"

					quotaProject.Spec.Metrics[garden.QuotaMetricStorageStandard] = resource.MustParse("20Gi")

					attrs := admission.NewAttributesRecord(&shoot2, nil, garden.Kind("Shoot").WithVersion("version"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, nil)

					err := admissionHandler.Validate(context.TODO(), attrs, nil)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Quota limits exceeded"))
				})
			})
		})

		Context("tests for Quota validation corner cases", func() {
			It("should pass because shoot is intended to get deleted", func() {
				var now metav1.Time
				now.Time = time.Now()
				shoot.DeletionTimestamp = &now

				attrs := admission.NewAttributesRecord(&shoot, nil, garden.Kind("Shoot").WithVersion("version"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, nil)

				err := admissionHandler.Validate(context.TODO(), attrs, nil)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should pass because shoots secret binding has no quotas referenced", func() {
				secretBinding.Quotas = make([]corev1.ObjectReference, 0)
				gardenInformerFactory.Garden().InternalVersion().SecretBindings().Informer().GetStore().Add(&secretBinding)
				attrs := admission.NewAttributesRecord(&shoot, nil, garden.Kind("Shoot").WithVersion("version"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, nil)

				err := admissionHandler.Validate(context.TODO(), attrs, nil)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should pass shoots secret binding having quota with no metrics", func() {
				emptyQuotaName := "empty-quota"
				emptyQuota := garden.Quota{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: trialNamespace,
						Name:      emptyQuotaName,
					},
					Spec: garden.QuotaSpec{
						ClusterLifetimeDays: &quotaProjectLifetime,
						Scope: corev1.ObjectReference{
							APIVersion: "core.gardener.cloud/v1beta1",
							Kind:       "Project",
						},
						Metrics: corev1.ResourceList{},
					},
				}
				secretBinding.Quotas = []corev1.ObjectReference{
					{
						Namespace: trialNamespace,
						Name:      emptyQuotaName,
					},
				}

				gardenInformerFactory.Garden().InternalVersion().Quotas().Informer().GetStore().Add(&emptyQuota)
				gardenInformerFactory.Garden().InternalVersion().SecretBindings().Informer().GetStore().Add(&secretBinding)
				attrs := admission.NewAttributesRecord(&shoot, nil, garden.Kind("Shoot").WithVersion("version"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("version"), "", admission.Create, &metav1.CreateOptions{}, false, nil)

				err := admissionHandler.Validate(context.TODO(), attrs, nil)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("tests for extending the lifetime of a Shoot", func() {
			BeforeEach(func() {
				annotations := map[string]string{
					common.ShootExpirationTimestamp: "2018-01-01T00:00:00+00:00",
				}
				shoot.Annotations = annotations
				oldShoot = *shoot.DeepCopy()
			})

			It("should pass because no quota prescribe a clusterLifetime", func() {
				quotaProject.Spec.ClusterLifetimeDays = nil
				quotaSecret.Spec.ClusterLifetimeDays = nil
				gardenInformerFactory.Garden().InternalVersion().Quotas().Informer().GetStore().Add(&quotaProject)
				gardenInformerFactory.Garden().InternalVersion().Quotas().Informer().GetStore().Add(&quotaSecret)

				attrs := admission.NewAttributesRecord(&shoot, &oldShoot, garden.Kind("Shoot").WithVersion("version"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, nil)

				err := admissionHandler.Validate(context.TODO(), attrs, nil)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should pass as shoot expiration time can be extended", func() {
				shoot.Annotations[common.ShootExpirationTimestamp] = "2018-01-02T00:00:00+00:00" // plus 1 day
				attrs := admission.NewAttributesRecord(&shoot, &oldShoot, garden.Kind("Shoot").WithVersion("version"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, nil)

				err := admissionHandler.Validate(context.TODO(), attrs, nil)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should fail as shoots expiration time can’t be extended, because requested time higher then quota allows", func() {
				shoot.Annotations[common.ShootExpirationTimestamp] = "2018-01-09T00:00:00+00:00" // plus 8 days
				attrs := admission.NewAttributesRecord(&shoot, &oldShoot, garden.Kind("Shoot").WithVersion("version"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("version"), "", admission.Update, &metav1.UpdateOptions{}, false, nil)

				err := admissionHandler.Validate(context.TODO(), attrs, nil)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
