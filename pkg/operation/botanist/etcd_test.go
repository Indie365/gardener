// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package botanist_test

import (
	"context"
	"fmt"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	seedmanagementv1alpha1 "github.com/gardener/gardener/pkg/apis/seedmanagement/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	fakeclientset "github.com/gardener/gardener/pkg/client/kubernetes/fake"
	"github.com/gardener/gardener/pkg/features"
	gardenletconfig "github.com/gardener/gardener/pkg/gardenlet/apis/config"
	gardenletfeatures "github.com/gardener/gardener/pkg/gardenlet/features"
	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"
	"github.com/gardener/gardener/pkg/operation"
	. "github.com/gardener/gardener/pkg/operation/botanist"
	"github.com/gardener/gardener/pkg/operation/botanist/component/etcd"
	mocketcd "github.com/gardener/gardener/pkg/operation/botanist/component/etcd/mock"
	seedpkg "github.com/gardener/gardener/pkg/operation/seed"
	shootpkg "github.com/gardener/gardener/pkg/operation/shoot"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	fakesecretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager/fake"
	"github.com/gardener/gardener/pkg/utils/test"

	hvpav1alpha1 "github.com/gardener/hvpa-controller/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-multierror"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubernetesscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Etcd", func() {
	var (
		ctrl             *gomock.Controller
		kubernetesClient kubernetes.Interface
		c                *mockclient.MockClient
		reader           *mockclient.MockReader
		fakeClient       client.Client
		sm               secretsmanager.Interface
		botanist         *Botanist

		ctx                   = context.TODO()
		fakeErr               = fmt.Errorf("fake err")
		namespace             = "shoot--foo--bar"
		role                  = "test"
		class                 = etcd.ClassImportant
		maintenanceTimeWindow = gardencorev1beta1.MaintenanceTimeWindow{
			Begin: "123456+0000",
			End:   "162543+0000",
		}
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		c = mockclient.NewMockClient(ctrl)
		reader = mockclient.NewMockReader(ctrl)
		kubernetesClient = fakeclientset.NewClientSetBuilder().
			WithClient(c).
			WithAPIReader(reader).
			Build()
		fakeClient = fakeclient.NewClientBuilder().WithScheme(kubernetesscheme.Scheme).Build()
		sm = fakesecretsmanager.New(fakeClient, namespace)
		botanist = &Botanist{Operation: &operation.Operation{}}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#DefaultEtcd", func() {
		var hvpaEnabled = true

		BeforeEach(func() {
			botanist.SecretsManager = sm
			botanist.K8sSeedClient = kubernetesClient
			botanist.Seed = &seedpkg.Seed{}
			botanist.Shoot = &shootpkg.Shoot{
				SeedNamespace: namespace,
			}
			botanist.Seed.SetInfo(&gardencorev1beta1.Seed{})
			botanist.Shoot.SetInfo(&gardencorev1beta1.Shoot{
				Spec: gardencorev1beta1.ShootSpec{
					Kubernetes: gardencorev1beta1.Kubernetes{
						Version: "1.20.2",
					},
					Maintenance: &gardencorev1beta1.Maintenance{
						TimeWindow: &maintenanceTimeWindow,
					},
				},
			})
		})

		Context("no ManagedSeeds", func() {
			BeforeEach(func() {
				botanist.ManagedSeed = nil
			})

			computeUpdateMode := func(class etcd.Class, purpose gardencorev1beta1.ShootPurpose) string {
				if class == etcd.ClassImportant && (purpose == gardencorev1beta1.ShootPurposeProduction || purpose == gardencorev1beta1.ShootPurposeInfrastructure) {
					return hvpav1alpha1.UpdateModeOff
				}
				return hvpav1alpha1.UpdateModeMaintenanceWindow
			}

			for _, etcdClass := range []etcd.Class{etcd.ClassNormal, etcd.ClassImportant} {
				for _, shootPurpose := range []gardencorev1beta1.ShootPurpose{gardencorev1beta1.ShootPurposeEvaluation, gardencorev1beta1.ShootPurposeProduction, gardencorev1beta1.ShootPurposeInfrastructure} {
					var (
						class   = etcdClass
						purpose = shootPurpose
					)
					It(fmt.Sprintf("should successfully create an etcd interface: class = %q, purpose = %q", class, purpose), func() {
						defer test.WithFeatureGate(gardenletfeatures.FeatureGate, features.HVPA, hvpaEnabled)()

						botanist.Shoot.Purpose = purpose

						validator := &newEtcdValidator{
							expectedClient:                  Equal(c),
							expectedLogger:                  BeAssignableToTypeOf(logr.Logger{}),
							expectedNamespace:               Equal(namespace),
							expectedSecretsManager:          Equal(sm),
							expectedRole:                    Equal(role),
							expectedClass:                   Equal(class),
							expectedReplicas:                PointTo(Equal(int32(1))),
							expectedStorageCapacity:         Equal("10Gi"),
							expectedDefragmentationSchedule: Equal(pointer.String("34 12 */3 * *")),
							expectedHVPAConfig: Equal(&etcd.HVPAConfig{
								Enabled:               hvpaEnabled,
								MaintenanceTimeWindow: maintenanceTimeWindow,
								ScaleDownUpdateMode:   pointer.String(computeUpdateMode(class, purpose)),
							}),
						}

						oldNewEtcd := NewEtcd
						defer func() { NewEtcd = oldNewEtcd }()
						NewEtcd = validator.NewEtcd

						etcd, err := botanist.DefaultEtcd(role, class)
						Expect(etcd).NotTo(BeNil())
						Expect(err).NotTo(HaveOccurred())
					})
				}
			}
		})

		Context("no HVPAShootedSeed feature gate", func() {
			hvpaForShootedSeedEnabled := false

			BeforeEach(func() {
				botanist.ManagedSeed = &seedmanagementv1alpha1.ManagedSeed{}
			})

			It("should successfully create an etcd interface (normal class)", func() {
				defer test.WithFeatureGate(gardenletfeatures.FeatureGate, features.HVPAForShootedSeed, hvpaForShootedSeedEnabled)()

				validator := &newEtcdValidator{
					expectedClient:                  Equal(c),
					expectedLogger:                  BeAssignableToTypeOf(logr.Logger{}),
					expectedNamespace:               Equal(namespace),
					expectedSecretsManager:          Equal(sm),
					expectedRole:                    Equal(role),
					expectedClass:                   Equal(class),
					expectedReplicas:                PointTo(Equal(int32(1))),
					expectedStorageCapacity:         Equal("10Gi"),
					expectedDefragmentationSchedule: Equal(pointer.String("34 12 * * *")),
					expectedHVPAConfig: Equal(&etcd.HVPAConfig{
						Enabled:               hvpaForShootedSeedEnabled,
						MaintenanceTimeWindow: maintenanceTimeWindow,
						ScaleDownUpdateMode:   pointer.String(hvpav1alpha1.UpdateModeMaintenanceWindow),
					}),
				}

				oldNewEtcd := NewEtcd
				defer func() { NewEtcd = oldNewEtcd }()
				NewEtcd = validator.NewEtcd

				etcd, err := botanist.DefaultEtcd(role, class)
				Expect(etcd).NotTo(BeNil())
				Expect(err).NotTo(HaveOccurred())
			})

			It("should successfully create an etcd interface (important class)", func() {
				class := etcd.ClassImportant

				defer test.WithFeatureGate(gardenletfeatures.FeatureGate, features.HVPAForShootedSeed, hvpaForShootedSeedEnabled)()

				validator := &newEtcdValidator{
					expectedClient:                  Equal(c),
					expectedLogger:                  BeAssignableToTypeOf(logr.Logger{}),
					expectedNamespace:               Equal(namespace),
					expectedSecretsManager:          Equal(sm),
					expectedRole:                    Equal(role),
					expectedClass:                   Equal(class),
					expectedReplicas:                PointTo(Equal(int32(1))),
					expectedStorageCapacity:         Equal("10Gi"),
					expectedDefragmentationSchedule: Equal(pointer.String("34 12 * * *")),
					expectedHVPAConfig: Equal(&etcd.HVPAConfig{
						Enabled:               hvpaForShootedSeedEnabled,
						MaintenanceTimeWindow: maintenanceTimeWindow,
						ScaleDownUpdateMode:   pointer.String(hvpav1alpha1.UpdateModeMaintenanceWindow),
					}),
				}

				oldNewEtcd := NewEtcd
				defer func() { NewEtcd = oldNewEtcd }()
				NewEtcd = validator.NewEtcd

				etcd, err := botanist.DefaultEtcd(role, class)
				Expect(etcd).NotTo(BeNil())
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("should return an error because the maintenance time window cannot be parsed", func() {
			defer test.WithFeatureGate(gardenletfeatures.FeatureGate, features.HVPA, true)()
			botanist.Shoot.GetInfo().Spec.Maintenance.TimeWindow = &gardencorev1beta1.MaintenanceTimeWindow{
				Begin: "foobar",
				End:   "barfoo",
			}

			etcd, err := botanist.DefaultEtcd(role, class)
			Expect(etcd).To(BeNil())
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("#DeployEtcd", func() {
		var (
			etcdMain, etcdEvents *mocketcd.MockInterface
			shootUID             = types.UID("uuid")
		)

		BeforeEach(func() {
			etcdMain, etcdEvents = mocketcd.NewMockInterface(ctrl), mocketcd.NewMockInterface(ctrl)

			botanist.K8sSeedClient = kubernetesClient
			botanist.Seed = &seedpkg.Seed{}
			botanist.Shoot = &shootpkg.Shoot{
				Components: &shootpkg.Components{
					ControlPlane: &shootpkg.ControlPlane{
						EtcdMain:   etcdMain,
						EtcdEvents: etcdEvents,
					},
				},
				SeedNamespace:         namespace,
				BackupEntryName:       namespace + "--" + string(shootUID),
				InternalClusterDomain: "internal.example.com",
			}
			botanist.Seed.SetInfo(&gardencorev1beta1.Seed{
				Status: gardencorev1beta1.SeedStatus{
					ClusterIdentity: pointer.String("seed-identity"),
				},
			})
			botanist.Shoot.SetInfo(&gardencorev1beta1.Shoot{
				Spec: gardencorev1beta1.ShootSpec{
					Maintenance: &gardencorev1beta1.Maintenance{
						TimeWindow: &maintenanceTimeWindow,
					},
				},
				Status: gardencorev1beta1.ShootStatus{
					TechnicalID: namespace,
					UID:         shootUID,
				},
			})
		})

		It("should fail when the deploy function fails for etcd-main", func() {
			etcdMain.EXPECT().Deploy(ctx).Return(fakeErr)
			etcdEvents.EXPECT().Deploy(ctx)

			err := botanist.DeployEtcd(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(&multierror.Error{}))
			Expect(err.(*multierror.Error).Errors).To(ConsistOf(Equal(fakeErr)))
		})

		It("should fail when the deploy function fails for etcd-events", func() {
			etcdMain.EXPECT().Deploy(ctx)
			etcdEvents.EXPECT().Deploy(ctx).Return(fakeErr)

			err := botanist.DeployEtcd(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(&multierror.Error{}))
			Expect(err.(*multierror.Error).Errors).To(ConsistOf(Equal(fakeErr)))
		})

		Context("w/o backup", func() {
			BeforeEach(func() {
				botanist.Seed.GetInfo().Spec.Backup = nil
			})

			It("should set the secrets and deploy", func() {
				etcdMain.EXPECT().Deploy(ctx)
				etcdEvents.EXPECT().Deploy(ctx)

				c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "etcd-server-cert", Namespace: namespace}})

				Expect(botanist.DeployEtcd(ctx)).To(Succeed())
			})
		})

		Context("w/ backup", func() {
			var (
				backupProvider = "prov"
				bucketName     = "container"
				backupSecret   = &corev1.Secret{
					Data: map[string][]byte{
						"bucketName": []byte(bucketName),
					},
				}
				backupLeaderElectionConfig = &gardenletconfig.ETCDBackupLeaderElection{
					ReelectionPeriod: &metav1.Duration{Duration: 2 * time.Second},
				}

				expectGetBackupSecret = func() {
					c.EXPECT().Get(ctx, kutil.Key(namespace, "etcd-backup"), gomock.AssignableToTypeOf(&corev1.Secret{})).DoAndReturn(
						func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
							backupSecret.DeepCopyInto(obj.(*corev1.Secret))
							return nil
						},
					)
				}
				expectSetBackupConfig = func() {
					etcdMain.EXPECT().SetBackupConfig(&etcd.BackupConfig{
						Provider:             backupProvider,
						SecretRefName:        "etcd-backup",
						Prefix:               namespace + "--" + string(shootUID),
						Container:            bucketName,
						FullSnapshotSchedule: "1 12 * * *",
						LeaderElection:       backupLeaderElectionConfig,
					})
				}
				expectSetOwnerCheckConfig = func() {
					etcdMain.EXPECT().SetOwnerCheckConfig(&etcd.OwnerCheckConfig{
						Name: "owner.internal.example.com",
						ID:   "seed-identity",
					})
				}
			)

			BeforeEach(func() {
				botanist.Seed.GetInfo().Spec.Backup = &gardencorev1beta1.SeedBackup{
					Provider: backupProvider,
				}
				botanist.Config = &gardenletconfig.GardenletConfiguration{
					ETCDConfig: &gardenletconfig.ETCDConfig{
						BackupLeaderElection: backupLeaderElectionConfig,
					},
				}
			})

			It("should set the secrets and deploy with owner checks", func() {
				expectGetBackupSecret()
				expectSetBackupConfig()
				expectSetOwnerCheckConfig()
				etcdMain.EXPECT().Deploy(ctx)
				etcdEvents.EXPECT().Deploy(ctx)
				c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "etcd-server-cert", Namespace: namespace}})

				Expect(botanist.DeployEtcd(ctx)).To(Succeed())
			})

			It("should set secrets and deploy without owner checks if HAControlPlanes is enabled and the high-availability annotation is set on the shoot", func() {
				defer test.WithFeatureGate(gardenletfeatures.FeatureGate, features.HAControlPlanes, true)()
				botanist.Shoot.GetInfo().ObjectMeta.Annotations = map[string]string{
					v1beta1constants.ShootAlphaControlPlaneHighAvailability: "true",
				}

				expectGetBackupSecret()
				expectSetBackupConfig()
				etcdMain.EXPECT().Deploy(ctx)
				etcdEvents.EXPECT().Deploy(ctx)
				c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "etcd-server-cert", Namespace: namespace}})

				Expect(botanist.DeployEtcd(ctx)).To(Succeed())
			})

			It("should set the secrets and deploy without owner checks if they are disabled", func() {
				botanist.Seed.GetInfo().Spec.Settings = &gardencorev1beta1.SeedSettings{
					OwnerChecks: &gardencorev1beta1.SeedSettingOwnerChecks{
						Enabled: false,
					},
				}

				expectGetBackupSecret()
				expectSetBackupConfig()
				etcdMain.EXPECT().Deploy(ctx)
				etcdEvents.EXPECT().Deploy(ctx)
				c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "etcd-server-cert", Namespace: namespace}})

				Expect(botanist.DeployEtcd(ctx)).To(Succeed())
			})

			It("should fail when reading the backup secret fails", func() {
				c.EXPECT().Get(ctx, kutil.Key(namespace, "etcd-backup"), gomock.AssignableToTypeOf(&corev1.Secret{})).Return(fakeErr)

				Expect(botanist.DeployEtcd(ctx)).To(MatchError(fakeErr))
			})

			It("should fail when the backup schedule cannot be determined", func() {
				botanist.Shoot.GetInfo().Spec.Maintenance.TimeWindow = &gardencorev1beta1.MaintenanceTimeWindow{
					Begin: "foobar",
					End:   "barfoo",
				}
				expectGetBackupSecret()

				Expect(botanist.DeployEtcd(ctx)).To(HaveOccurred())
			})
		})
	})

	Describe("#DestroyEtcd", func() {
		var (
			etcdMain, etcdEvents *mocketcd.MockInterface
		)

		BeforeEach(func() {
			etcdMain, etcdEvents = mocketcd.NewMockInterface(ctrl), mocketcd.NewMockInterface(ctrl)

			botanist.Shoot = &shootpkg.Shoot{
				Components: &shootpkg.Components{
					ControlPlane: &shootpkg.ControlPlane{
						EtcdMain:   etcdMain,
						EtcdEvents: etcdEvents,
					},
				},
			}
		})

		It("should fail when the destroy function fails for etcd-main", func() {
			etcdMain.EXPECT().Destroy(ctx).Return(fakeErr)
			etcdEvents.EXPECT().Destroy(ctx)

			err := botanist.DestroyEtcd(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(&multierror.Error{}))
			Expect(err.(*multierror.Error).Errors).To(ConsistOf(Equal(fakeErr)))
		})

		It("should fail when the destroy function fails for etcd-events", func() {
			etcdMain.EXPECT().Destroy(ctx)
			etcdEvents.EXPECT().Destroy(ctx).Return(fakeErr)

			err := botanist.DestroyEtcd(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(&multierror.Error{}))
			Expect(err.(*multierror.Error).Errors).To(ConsistOf(Equal(fakeErr)))
		})

		It("should succeed when both etcd-main and etcd-events destroy is successful", func() {
			etcdMain.EXPECT().Destroy(ctx)
			etcdEvents.EXPECT().Destroy(ctx)

			Expect(botanist.DestroyEtcd(ctx)).To(Succeed())
		})
	})
})

type newEtcdValidator struct {
	etcd.Interface

	expectedClient                  gomegatypes.GomegaMatcher
	expectedLogger                  gomegatypes.GomegaMatcher
	expectedNamespace               gomegatypes.GomegaMatcher
	expectedSecretsManager          gomegatypes.GomegaMatcher
	expectedRole                    gomegatypes.GomegaMatcher
	expectedClass                   gomegatypes.GomegaMatcher
	expectedReplicas                gomegatypes.GomegaMatcher
	expectedStorageCapacity         gomegatypes.GomegaMatcher
	expectedDefragmentationSchedule gomegatypes.GomegaMatcher
	expectedHVPAConfig              gomegatypes.GomegaMatcher
}

func (v *newEtcdValidator) NewEtcd(
	client client.Client,
	log logr.Logger,
	namespace string,
	secretsManager secretsmanager.Interface,
	role string,
	class etcd.Class,
	_ map[string]string,
	replicas *int32,
	storageCapacity string,
	defragmentationSchedule *string,
	_ gardencorev1beta1.ShootCredentialsRotationPhase,
	_ string,
) etcd.Interface {
	Expect(client).To(v.expectedClient)
	Expect(log).To(v.expectedLogger)
	Expect(namespace).To(v.expectedNamespace)
	Expect(secretsManager).To(v.expectedSecretsManager)
	Expect(role).To(v.expectedRole)
	Expect(class).To(v.expectedClass)
	Expect(replicas).To(v.expectedReplicas)
	Expect(storageCapacity).To(v.expectedStorageCapacity)
	Expect(defragmentationSchedule).To(v.expectedDefragmentationSchedule)

	return v
}

func (v *newEtcdValidator) SetHVPAConfig(config *etcd.HVPAConfig) {
	Expect(config).To(v.expectedHVPAConfig)
}
