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

package etcd_test

import (
	"context"
	"fmt"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	mockkubernetes "github.com/gardener/gardener/pkg/client/kubernetes/mock"
	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"
	. "github.com/gardener/gardener/pkg/operation/botanist/component/etcd"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/gardener"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	secretutils "github.com/gardener/gardener/pkg/utils/secrets"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	fakesecretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager/fake"
	"github.com/gardener/gardener/pkg/utils/test"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"

	druidv1alpha1 "github.com/gardener/etcd-druid/api/v1alpha1"
	hvpav1alpha1 "github.com/gardener/hvpa-controller/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta1 "k8s.io/api/autoscaling/v2beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	kubernetesscheme "k8s.io/client-go/kubernetes/scheme"
	testclock "k8s.io/utils/clock/testing"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("Etcd", func() {
	Describe("#ServiceName", func() {
		It("should return the expected service name", func() {
			Expect(ServiceName(testRole)).To(Equal("etcd-" + testRole + "-client"))
		})
	})

	var (
		ctrl                 *gomock.Controller
		c                    *mockclient.MockClient
		fakeClient           client.Client
		sm                   secretsmanager.Interface
		etcd                 Interface
		log                  logr.Logger
		failureToleranceType *gardencorev1beta1.FailureToleranceType

		ctx                     = context.TODO()
		fakeErr                 = fmt.Errorf("fake err")
		class                   = ClassNormal
		annotations             = map[string]string{}
		replicas                = pointer.Int32(1)
		storageCapacity         = "12Gi"
		storageCapacityQuantity = resource.MustParse(storageCapacity)
		defragmentationSchedule = "abcd"

		secretNameCA         = "ca-etcd"
		secretNamePeerCA     = "ca-etcd-peer"
		secretNameServer     = "etcd-server-" + testRole
		secretNameServerPeer = "etcd-peer-server-" + testRole
		secretNameClient     = "etcd-client"

		maintenanceTimeWindow = gardencorev1beta1.MaintenanceTimeWindow{
			Begin: "1234",
			End:   "5678",
		}
		now                     = time.Time{}
		quota                   = resource.MustParse("8Gi")
		garbageCollectionPolicy = druidv1alpha1.GarbageCollectionPolicy(druidv1alpha1.GarbageCollectionPolicyExponential)
		garbageCollectionPeriod = metav1.Duration{Duration: 12 * time.Hour}
		compressionPolicy       = druidv1alpha1.GzipCompression
		compressionSpec         = druidv1alpha1.CompressionSpec{
			Enabled: pointer.Bool(true),
			Policy:  &compressionPolicy,
		}
		backupLeaderElectionEtcdConnectionTimeout = &metav1.Duration{Duration: 10 * time.Second}
		backupLeaderElectionReelectionPeriod      = &metav1.Duration{Duration: 11 * time.Second}

		updateModeAuto     = hvpav1alpha1.UpdateModeAuto
		containerPolicyOff = vpaautoscalingv1.ContainerScalingModeOff
		controlledValues   = vpaautoscalingv1.ContainerControlledValuesRequestsOnly
		metricsBasic       = druidv1alpha1.Basic
		metricsExtensive   = druidv1alpha1.Extensive

		networkPolicyClientName = "allow-etcd"
		networkPolicyPeerName   = "allow-etcd-peer"
		etcdName                = "etcd-" + testRole
		hvpaName                = "etcd-" + testRole

		protocolTCP       = corev1.ProtocolTCP
		portEtcdClient    = intstr.FromInt(2379)
		portEtcdPeer      = intstr.FromInt(2380)
		portBackupRestore = intstr.FromInt(8080)

		clientNetworkPolicy = &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      networkPolicyClientName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					"gardener.cloud/description": "Allows Ingress to etcd pods from the Shoot's Kubernetes API Server.",
				},
				Labels: map[string]string{
					"gardener.cloud/role": "controlplane",
				},
			},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"gardener.cloud/role": "controlplane",
						"app":                 "etcd-statefulset",
					},
				},
				Ingress: []networkingv1.NetworkPolicyIngressRule{
					{
						From: []networkingv1.NetworkPolicyPeer{
							{
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"gardener.cloud/role": "controlplane",
										"app":                 "kubernetes",
										"role":                "apiserver",
									},
								},
							},
							{
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"gardener.cloud/role": "monitoring",
										"app":                 "prometheus",
										"role":                "monitoring",
									},
								},
							},
						},
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Protocol: &protocolTCP,
								Port:     &portEtcdClient,
							},
							{
								Protocol: &protocolTCP,
								Port:     &portBackupRestore,
							},
						},
					},
				},
				PolicyTypes: []networkingv1.PolicyType{
					networkingv1.PolicyTypeIngress,
				},
			},
		}

		peerNetworkPolicy = &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      networkPolicyPeerName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					"gardener.cloud/description": "Allows Ingress to etcd pods from etcd pods for peer communication.",
				},
				Labels: map[string]string{
					"gardener.cloud/role": "controlplane",
				},
			},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"gardener.cloud/role": "controlplane",
						"app":                 "etcd-statefulset",
					},
				},
				Egress: []networkingv1.NetworkPolicyEgressRule{
					{
						To: []networkingv1.NetworkPolicyPeer{
							{
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"gardener.cloud/role": "controlplane",
										"app":                 "etcd-statefulset",
									},
								},
							},
						},
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Protocol: &protocolTCP,
								Port:     &portEtcdClient,
							},
							{
								Protocol: &protocolTCP,
								Port:     &portBackupRestore,
							},
							{
								Protocol: &protocolTCP,
								Port:     &portEtcdPeer,
							},
						},
					},
				},
				Ingress: []networkingv1.NetworkPolicyIngressRule{
					{
						From: []networkingv1.NetworkPolicyPeer{
							{
								PodSelector: &metav1.LabelSelector{
									MatchLabels: GetLabels(),
								},
							},
						},
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Protocol: &protocolTCP,
								Port:     &portEtcdClient,
							},
							{
								Protocol: &protocolTCP,
								Port:     &portBackupRestore,
							},
							{
								Protocol: &protocolTCP,
								Port:     &portEtcdPeer,
							},
						},
					},
				},
				PolicyTypes: []networkingv1.PolicyType{
					networkingv1.PolicyTypeIngress,
					networkingv1.PolicyTypeEgress,
				},
			},
		}

		etcdObjFor = func(
			class Class,
			replicas int32,
			backupConfig *BackupConfig,
			existingDefragmentationSchedule,
			existingBackupSchedule string,
			existingResourcesContainerEtcd *corev1.ResourceRequirements,
			existingResourcesContainerBackupRestore *corev1.ResourceRequirements,
			failureToleranceType *gardencorev1beta1.FailureToleranceType,
			caSecretName string,
			clientSecretName string,
			serverSecretName string,
			peerCASecretName *string,
			peerServerSecretName *string,
		) *druidv1alpha1.Etcd {

			defragSchedule := defragmentationSchedule
			if existingDefragmentationSchedule != "" {
				defragSchedule = existingDefragmentationSchedule
			}

			resourcesContainerEtcd := &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("1G"),
				},
			}
			if existingResourcesContainerEtcd != nil {
				resourcesContainerEtcd = existingResourcesContainerEtcd
			}

			resourcesContainerBackupRestore := &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("23m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
			}
			if existingResourcesContainerBackupRestore != nil {
				resourcesContainerBackupRestore = existingResourcesContainerBackupRestore
			}

			obj := &druidv1alpha1.Etcd{
				ObjectMeta: metav1.ObjectMeta{
					Name:      etcdName,
					Namespace: testNamespace,
					Annotations: map[string]string{
						"gardener.cloud/operation": "reconcile",
						"gardener.cloud/timestamp": now.String(),
					},
					Labels: map[string]string{
						"gardener.cloud/role": "controlplane",
						"role":                testRole,
					},
				},
				Spec: druidv1alpha1.EtcdSpec{
					Replicas:          replicas,
					PriorityClassName: pointer.String("gardener-system-500"),
					Labels: map[string]string{
						"gardener.cloud/role":              "controlplane",
						"garden.sapcloud.io/role":          "controlplane",
						"role":                             testRole,
						"app":                              "etcd-statefulset",
						"networking.gardener.cloud/to-dns": "allowed",
						"networking.gardener.cloud/to-public-networks":  "allowed",
						"networking.gardener.cloud/to-private-networks": "allowed",
						"networking.gardener.cloud/to-seed-apiserver":   "allowed",
					},
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"garden.sapcloud.io/role": "controlplane",
							"role":                    testRole,
							"app":                     "etcd-statefulset",
						},
					},
					Etcd: druidv1alpha1.EtcdConfig{
						Resources: resourcesContainerEtcd,
						ClientUrlTLS: &druidv1alpha1.TLSConfig{
							TLSCASecretRef: druidv1alpha1.SecretReference{
								SecretReference: corev1.SecretReference{
									Name:      caSecretName,
									Namespace: testNamespace,
								},
								DataKey: pointer.String("bundle.crt"),
							},
							ServerTLSSecretRef: corev1.SecretReference{
								Name:      serverSecretName,
								Namespace: testNamespace,
							},
							ClientTLSSecretRef: corev1.SecretReference{
								Name:      clientSecretName,
								Namespace: testNamespace,
							},
						},
						ServerPort:              &PortEtcdPeer,
						ClientPort:              &PortEtcdClient,
						Metrics:                 &metricsBasic,
						DefragmentationSchedule: &defragSchedule,
						Quota:                   &quota,
					},
					Backup: druidv1alpha1.BackupSpec{
						TLS: &druidv1alpha1.TLSConfig{
							TLSCASecretRef: druidv1alpha1.SecretReference{
								SecretReference: corev1.SecretReference{
									Name:      caSecretName,
									Namespace: testNamespace,
								},
								DataKey: pointer.String("bundle.crt"),
							},
							ServerTLSSecretRef: corev1.SecretReference{
								Name:      serverSecretName,
								Namespace: testNamespace,
							},
							ClientTLSSecretRef: corev1.SecretReference{
								Name:      clientSecretName,
								Namespace: testNamespace,
							},
						},
						Port:                    &PortBackupRestore,
						Resources:               resourcesContainerBackupRestore,
						GarbageCollectionPolicy: &garbageCollectionPolicy,
						GarbageCollectionPeriod: &garbageCollectionPeriod,
						SnapshotCompression:     &compressionSpec,
					},
					StorageCapacity:     &storageCapacityQuantity,
					VolumeClaimTemplate: pointer.String(etcdName),
				},
			}

			if class == ClassImportant {
				obj.Spec.Annotations = map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "false"}
				obj.Spec.Etcd.Metrics = &metricsExtensive
				obj.Spec.VolumeClaimTemplate = pointer.String(testRole + "-etcd")
			}

			if failureToleranceType != nil {
				obj.Spec.Etcd.PeerUrlTLS = &druidv1alpha1.TLSConfig{
					ServerTLSSecretRef: corev1.SecretReference{
						Name:      secretNameServerPeer,
						Namespace: testNamespace,
					},
					TLSCASecretRef: druidv1alpha1.SecretReference{
						SecretReference: corev1.SecretReference{
							Name:      secretNamePeerCA,
							Namespace: testNamespace,
						},
						DataKey: pointer.String(secretutils.DataKeyCertificateBundle),
					},
				}

				switch *failureToleranceType {
				case gardencorev1beta1.FailureToleranceTypeNode:
					obj.Spec.SchedulingConstraints = druidv1alpha1.SchedulingConstraints{
						Affinity: &corev1.Affinity{
							PodAntiAffinity: &corev1.PodAntiAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
									{
										TopologyKey: corev1.LabelHostname,
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: map[string]string{
												"gardener.cloud/role": "controlplane",
												"role":                testRole,
											},
										},
									},
								},
							},
						},
					}
				case gardencorev1beta1.FailureToleranceTypeZone:
					obj.Spec.SchedulingConstraints = druidv1alpha1.SchedulingConstraints{
						Affinity: &corev1.Affinity{
							PodAntiAffinity: &corev1.PodAntiAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
									{
										TopologyKey: corev1.LabelTopologyZone,
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: map[string]string{
												"gardener.cloud/role": "controlplane",
												"role":                testRole,
											},
										},
									},
								},
							},
						},
					}
				}
			}

			if pointer.StringDeref(peerServerSecretName, "") != "" {
				obj.Spec.Etcd.PeerUrlTLS.ServerTLSSecretRef = corev1.SecretReference{
					Name:      *peerServerSecretName,
					Namespace: testNamespace,
				}
			}

			if pointer.StringDeref(peerCASecretName, "") != "" {
				obj.Spec.Etcd.PeerUrlTLS.TLSCASecretRef = druidv1alpha1.SecretReference{
					SecretReference: corev1.SecretReference{
						Name:      *peerCASecretName,
						Namespace: testNamespace,
					},
					DataKey: pointer.String(secretutils.DataKeyCertificateBundle),
				}
			}

			if backupConfig != nil {
				fullSnapshotSchedule := backupConfig.FullSnapshotSchedule
				if existingBackupSchedule != "" {
					fullSnapshotSchedule = existingBackupSchedule
				}

				provider := druidv1alpha1.StorageProvider(backupConfig.Provider)
				deltaSnapshotPeriod := metav1.Duration{Duration: 5 * time.Minute}
				deltaSnapshotMemoryLimit := resource.MustParse("100Mi")

				obj.Spec.Backup.Store = &druidv1alpha1.StoreSpec{
					SecretRef: &corev1.SecretReference{Name: backupConfig.SecretRefName},
					Container: &backupConfig.Container,
					Provider:  &provider,
					Prefix:    backupConfig.Prefix + "/etcd-" + testRole,
				}
				obj.Spec.Backup.FullSnapshotSchedule = &fullSnapshotSchedule
				obj.Spec.Backup.DeltaSnapshotPeriod = &deltaSnapshotPeriod
				obj.Spec.Backup.DeltaSnapshotMemoryLimit = &deltaSnapshotMemoryLimit

				if backupConfig.LeaderElection != nil {
					obj.Spec.Backup.LeaderElection = &druidv1alpha1.LeaderElectionSpec{
						EtcdConnectionTimeout: backupLeaderElectionEtcdConnectionTimeout,
						ReelectionPeriod:      backupLeaderElectionReelectionPeriod,
					}
				}
			}

			return obj
		}
		hvpaFor = func(class Class, replicas int32, scaleDownUpdateMode string) *hvpav1alpha1.Hvpa {
			obj := &hvpav1alpha1.Hvpa{
				ObjectMeta: metav1.ObjectMeta{
					Name:      hvpaName,
					Namespace: testNamespace,
					Labels: map[string]string{
						"gardener.cloud/role": "controlplane",
						"role":                testRole,
						"app":                 "etcd-statefulset",
					},
				},
				Spec: hvpav1alpha1.HvpaSpec{
					Replicas: pointer.Int32(1),
					MaintenanceTimeWindow: &hvpav1alpha1.MaintenanceTimeWindow{
						Begin: maintenanceTimeWindow.Begin,
						End:   maintenanceTimeWindow.End,
					},
					Hpa: hvpav1alpha1.HpaSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"role": "etcd-hpa-" + testRole,
							},
						},
						Deploy: false,
						Template: hvpav1alpha1.HpaTemplate{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"role": "etcd-hpa-" + testRole,
								},
							},
							Spec: hvpav1alpha1.HpaTemplateSpec{
								MinReplicas: &replicas,
								MaxReplicas: replicas,
								Metrics: []autoscalingv2beta1.MetricSpec{
									{
										Type: autoscalingv2beta1.ResourceMetricSourceType,
										Resource: &autoscalingv2beta1.ResourceMetricSource{
											Name:                     corev1.ResourceCPU,
											TargetAverageUtilization: pointer.Int32(80),
										},
									},
									{
										Type: autoscalingv2beta1.ResourceMetricSourceType,
										Resource: &autoscalingv2beta1.ResourceMetricSource{
											Name:                     corev1.ResourceMemory,
											TargetAverageUtilization: pointer.Int32(80),
										},
									},
								},
							},
						},
					},
					Vpa: hvpav1alpha1.VpaSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"role": "etcd-vpa-" + testRole,
							},
						},
						Deploy: true,
						ScaleUp: hvpav1alpha1.ScaleType{
							UpdatePolicy: hvpav1alpha1.UpdatePolicy{
								UpdateMode: &updateModeAuto,
							},
							StabilizationDuration: pointer.String("5m"),
							MinChange: hvpav1alpha1.ScaleParams{
								CPU: hvpav1alpha1.ChangeParams{
									Value:      pointer.String("1"),
									Percentage: pointer.Int32(80),
								},
								Memory: hvpav1alpha1.ChangeParams{
									Value:      pointer.String("2G"),
									Percentage: pointer.Int32(80),
								},
							},
						},
						ScaleDown: hvpav1alpha1.ScaleType{
							UpdatePolicy: hvpav1alpha1.UpdatePolicy{
								UpdateMode: &scaleDownUpdateMode,
							},
							StabilizationDuration: pointer.String("15m"),
							MinChange: hvpav1alpha1.ScaleParams{
								CPU: hvpav1alpha1.ChangeParams{
									Value:      pointer.String("1"),
									Percentage: pointer.Int32(80),
								},
								Memory: hvpav1alpha1.ChangeParams{
									Value:      pointer.String("2G"),
									Percentage: pointer.Int32(80),
								},
							},
						},
						LimitsRequestsGapScaleParams: hvpav1alpha1.ScaleParams{
							CPU: hvpav1alpha1.ChangeParams{
								Value:      pointer.String("2"),
								Percentage: pointer.Int32(40),
							},
							Memory: hvpav1alpha1.ChangeParams{
								Value:      pointer.String("5G"),
								Percentage: pointer.Int32(40),
							},
						},
						Template: hvpav1alpha1.VpaTemplate{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"role": "etcd-vpa-" + testRole,
								},
							},
							Spec: hvpav1alpha1.VpaTemplateSpec{
								ResourcePolicy: &vpaautoscalingv1.PodResourcePolicy{
									ContainerPolicies: []vpaautoscalingv1.ContainerResourcePolicy{
										{
											ContainerName: "etcd",
											MinAllowed: corev1.ResourceList{
												corev1.ResourceCPU:    resource.MustParse("50m"),
												corev1.ResourceMemory: resource.MustParse("200M"),
											},
											MaxAllowed: corev1.ResourceList{
												corev1.ResourceCPU:    resource.MustParse("4"),
												corev1.ResourceMemory: resource.MustParse("30G"),
											},
											ControlledValues: &controlledValues,
										},
										{
											ContainerName:    "backup-restore",
											Mode:             &containerPolicyOff,
											ControlledValues: &controlledValues,
										},
									},
								},
							},
						},
					},
					WeightBasedScalingIntervals: []hvpav1alpha1.WeightBasedScalingInterval{
						{
							VpaWeight:         hvpav1alpha1.VpaOnly,
							StartReplicaCount: replicas,
							LastReplicaCount:  replicas,
						},
					},
					TargetRef: &autoscalingv2beta1.CrossVersionObjectReference{
						APIVersion: "apps/v1",
						Kind:       "StatefulSet",
						Name:       etcdName,
					},
				},
			}

			if class == ClassImportant {
				obj.Spec.Vpa.Template.Spec.ResourcePolicy.ContainerPolicies[0].MinAllowed = corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("700M"),
				}
			}

			return obj
		}
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		c = mockclient.NewMockClient(ctrl)
		fakeClient = fakeclient.NewClientBuilder().WithScheme(kubernetesscheme.Scheme).Build()
		sm = fakesecretsmanager.New(fakeClient, testNamespace)
		log = logr.Discard()

		By("creating secrets managed outside of this package for whose secretsmanager.Get() will be called")
		Expect(fakeClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ca-etcd", Namespace: testNamespace}})).To(Succeed())
		etcd = New(c, log, testNamespace, sm, testRole, class, annotations, failureToleranceType, replicas, storageCapacity, &defragmentationSchedule, "", "1.20.1")
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#Deploy", func() {
		var scaleDownUpdateMode = hvpav1alpha1.UpdateModeMaintenanceWindow
		newSetHVPAConfigFunc := func(updateMode string) func() {
			return func() {
				etcd.SetHVPAConfig(&HVPAConfig{
					Enabled:               true,
					MaintenanceTimeWindow: maintenanceTimeWindow,
					ScaleDownUpdateMode:   pointer.String(updateMode),
				})
			}
		}
		setHVPAConfig := newSetHVPAConfigFunc(scaleDownUpdateMode)

		BeforeEach(setHVPAConfig)

		It("should fail because the etcd object retrieval fails", func() {
			c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).Return(fakeErr)

			Expect(etcd.Deploy(ctx)).To(MatchError(fakeErr))
		})

		It("should fail because the statefulset object retrieval fails (using the default sts name)", func() {
			c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
			c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&appsv1.StatefulSet{})).Return(fakeErr)

			Expect(etcd.Deploy(ctx)).To(MatchError(fakeErr))
		})

		It("should fail because the statefulset object retrieval fails (using the sts name from etcd object)", func() {
			statefulSetName := "sts-name"

			c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).DoAndReturn(
				func(ctx context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
					(&druidv1alpha1.Etcd{
						Status: druidv1alpha1.EtcdStatus{
							Etcd: &druidv1alpha1.CrossVersionObjectReference{
								Name: statefulSetName,
							},
						},
					}).DeepCopyInto(obj.(*druidv1alpha1.Etcd))
					return nil
				},
			)
			c.EXPECT().Get(ctx, kutil.Key(testNamespace, statefulSetName), gomock.AssignableToTypeOf(&appsv1.StatefulSet{})).Return(fakeErr)

			Expect(etcd.Deploy(ctx)).To(MatchError(fakeErr))
		})

		It("should fail because the network policy cannot be created", func() {
			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&appsv1.StatefulSet{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),

				c.EXPECT().Get(ctx, kutil.Key(testNamespace, networkPolicyClientName), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()).Return(fakeErr),
			)

			Expect(etcd.Deploy(ctx)).To(MatchError(fakeErr))
		})

		It("should fail because the etcd cannot be created", func() {
			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&appsv1.StatefulSet{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),

				c.EXPECT().Get(ctx, kutil.Key(testNamespace, networkPolicyClientName), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()).Return(fakeErr),
			)

			Expect(etcd.Deploy(ctx)).To(MatchError(fakeErr))
		})

		It("should fail because the hvpa cannot be created", func() {
			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&appsv1.StatefulSet{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),

				c.EXPECT().Get(ctx, kutil.Key(testNamespace, networkPolicyClientName), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, hvpaName), gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{}), gomock.Any()).Return(fakeErr),
			)

			Expect(etcd.Deploy(ctx)).To(MatchError(fakeErr))
		})

		It("should fail because the hvpa cannot be deleted", func() {
			etcd.SetHVPAConfig(&HVPAConfig{})

			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&appsv1.StatefulSet{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),

				c.EXPECT().Get(ctx, kutil.Key(testNamespace, networkPolicyClientName), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()),
				c.EXPECT().Delete(ctx, gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{})).Return(fakeErr),
			)

			Expect(etcd.Deploy(ctx)).To(MatchError(fakeErr))
		})

		It("should successfully deploy (normal etcd)", func() {
			oldTimeNow := TimeNow
			defer func() { TimeNow = oldTimeNow }()
			TimeNow = func() time.Time { return now }

			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&appsv1.StatefulSet{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),

				c.EXPECT().Get(ctx, kutil.Key(testNamespace, networkPolicyClientName), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
					Expect(obj).To(DeepEqual(clientNetworkPolicy))
				}),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
					Expect(obj).To(DeepEqual(etcdObjFor(
						class,
						1,
						nil,
						"",
						"",
						nil,
						nil,
						nil,
						secretNameCA,
						secretNameClient,
						secretNameServer,
						nil,
						nil)))
				}),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, hvpaName), gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
					Expect(obj).To(DeepEqual(hvpaFor(class, 1, scaleDownUpdateMode)))
				}),
			)

			Expect(etcd.Deploy(ctx)).To(Succeed())
		})

		It("should not panic during deploy when etcd resource exists, but its status is not yet populated", func() {
			oldTimeNow := TimeNow
			defer func() { TimeNow = oldTimeNow }()
			TimeNow = func() time.Time { return now }

			var (
				existingReplicas int32 = 245
			)

			etcd = New(c, log, testNamespace, sm, testRole, class, annotations, failureToleranceType, nil, storageCapacity, &defragmentationSchedule, "", "1.20.1")
			setHVPAConfig()

			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")).DoAndReturn(func(ctx context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
					(&druidv1alpha1.Etcd{
						ObjectMeta: metav1.ObjectMeta{
							Name:      etcdName,
							Namespace: testNamespace,
						},
						Spec: druidv1alpha1.EtcdSpec{
							Replicas: existingReplicas,
						},
					}).DeepCopyInto(obj.(*druidv1alpha1.Etcd))
					return nil
				}),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&appsv1.StatefulSet{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),

				c.EXPECT().Get(ctx, kutil.Key(testNamespace, networkPolicyClientName), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
					Expect(obj).To(DeepEqual(clientNetworkPolicy))
				}),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()).Do(func(ctx context.Context, obj *druidv1alpha1.Etcd, _ client.Patch, _ ...client.PatchOption) {
					// ignore status when comparing
					obj.Status = druidv1alpha1.EtcdStatus{}

					Expect(obj).To(DeepEqual(etcdObjFor(
						class,
						existingReplicas,
						nil,
						"",
						"",
						nil,
						nil,
						nil,
						secretNameCA,
						secretNameClient,
						secretNameServer,
						nil,
						nil)))
				}),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, hvpaName), gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
					Expect(obj).To(DeepEqual(hvpaFor(class, existingReplicas, scaleDownUpdateMode)))
				}),
			)

			Expect(etcd.Deploy(ctx)).To(Succeed())
		})

		It("should successfully deploy (normal etcd) and retain replicas (etcd found)", func() {
			oldTimeNow := TimeNow
			defer func() { TimeNow = oldTimeNow }()
			TimeNow = func() time.Time { return now }

			var (
				existingReplicas int32 = 245
			)

			etcd = New(c, log, testNamespace, sm, testRole, class, annotations, failureToleranceType, nil, storageCapacity, &defragmentationSchedule, "", "1.20.1")
			setHVPAConfig()

			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")).DoAndReturn(func(ctx context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
					(&druidv1alpha1.Etcd{
						ObjectMeta: metav1.ObjectMeta{
							Name:      etcdName,
							Namespace: testNamespace,
						},
						Spec: druidv1alpha1.EtcdSpec{
							Replicas: existingReplicas,
						},
						Status: druidv1alpha1.EtcdStatus{
							Etcd: &druidv1alpha1.CrossVersionObjectReference{
								Name: etcdName,
							},
						},
					}).DeepCopyInto(obj.(*druidv1alpha1.Etcd))
					return nil
				}),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&appsv1.StatefulSet{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),

				c.EXPECT().Get(ctx, kutil.Key(testNamespace, networkPolicyClientName), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
					Expect(obj).To(DeepEqual(clientNetworkPolicy))
				}),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()).Do(func(ctx context.Context, obj *druidv1alpha1.Etcd, _ client.Patch, _ ...client.PatchOption) {
					// ignore status when comparing
					obj.Status = druidv1alpha1.EtcdStatus{}

					Expect(obj).To(DeepEqual(etcdObjFor(
						class,
						existingReplicas,
						nil,
						"",
						"",
						nil,
						nil,
						nil,
						secretNameCA,
						secretNameClient,
						secretNameServer,
						nil,
						nil)))
				}),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, hvpaName), gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
					Expect(obj).To(DeepEqual(hvpaFor(class, existingReplicas, scaleDownUpdateMode)))
				}),
			)

			Expect(etcd.Deploy(ctx)).To(Succeed())
		})

		It("should successfully deploy (normal etcd) and retain annotations (etcd found)", func() {
			oldTimeNow := TimeNow
			defer func() { TimeNow = oldTimeNow }()
			TimeNow = func() time.Time { return now }

			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")).DoAndReturn(func(ctx context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
					(&druidv1alpha1.Etcd{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"foo": "bar",
							},
							Name:      etcdName,
							Namespace: testNamespace,
						},
						Status: druidv1alpha1.EtcdStatus{
							Etcd: &druidv1alpha1.CrossVersionObjectReference{
								Name: etcdName,
							},
						},
					}).DeepCopyInto(obj.(*druidv1alpha1.Etcd))
					return nil
				}),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&appsv1.StatefulSet{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),

				c.EXPECT().Get(ctx, kutil.Key(testNamespace, networkPolicyClientName), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
					Expect(obj).To(DeepEqual(clientNetworkPolicy))
				}),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()).Do(func(ctx context.Context, obj *druidv1alpha1.Etcd, _ client.Patch, _ ...client.PatchOption) {
					// ignore status when comparing
					obj.Status = druidv1alpha1.EtcdStatus{}

					expectedObj := etcdObjFor(
						class,
						1,
						nil,
						"",
						"",
						nil,
						nil,
						nil,
						secretNameCA,
						secretNameClient,
						secretNameServer,
						nil,
						nil)
					expectedObj.Annotations = utils.MergeStringMaps(expectedObj.Annotations, map[string]string{
						"foo": "bar",
					})

					Expect(obj).To(DeepEqual(expectedObj))
				}),

				c.EXPECT().Get(ctx, kutil.Key(testNamespace, hvpaName), gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
					Expect(obj).To(DeepEqual(hvpaFor(class, 1, scaleDownUpdateMode)))
				}),
			)

			Expect(etcd.Deploy(ctx)).To(Succeed())
		})

		It("should successfully deploy (normal etcd) and keep the existing defragmentation schedule", func() {
			oldTimeNow := TimeNow
			defer func() { TimeNow = oldTimeNow }()
			TimeNow = func() time.Time { return now }

			existingDefragmentationSchedule := "foobardefragexisting"

			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).DoAndReturn(func(ctx context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
					(&druidv1alpha1.Etcd{
						ObjectMeta: metav1.ObjectMeta{
							Name:      etcdName,
							Namespace: testNamespace,
						},
						Spec: druidv1alpha1.EtcdSpec{
							Etcd: druidv1alpha1.EtcdConfig{
								DefragmentationSchedule: &existingDefragmentationSchedule,
							},
						},
						Status: druidv1alpha1.EtcdStatus{
							Etcd: &druidv1alpha1.CrossVersionObjectReference{
								Name: etcdName,
							},
						},
					}).DeepCopyInto(obj.(*druidv1alpha1.Etcd))
					return nil
				}),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&appsv1.StatefulSet{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),

				c.EXPECT().Get(ctx, kutil.Key(testNamespace, networkPolicyClientName), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
					Expect(obj).To(DeepEqual(clientNetworkPolicy))
				}),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()).Do(func(ctx context.Context, obj *druidv1alpha1.Etcd, _ client.Patch, _ ...client.PatchOption) {
					// ignore status when comparing
					obj.Status = druidv1alpha1.EtcdStatus{}

					Expect(obj).To(DeepEqual(etcdObjFor(
						class,
						1,
						nil,
						existingDefragmentationSchedule,
						"",
						nil,
						nil,
						nil,
						secretNameCA,
						secretNameClient,
						secretNameServer,
						nil,
						nil)))
				}),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, hvpaName), gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
					Expect(obj).To(DeepEqual(hvpaFor(class, 1, scaleDownUpdateMode)))
				}),
			)

			Expect(etcd.Deploy(ctx)).To(Succeed())
		})

		It("should successfully deploy (normal etcd) and keep the existing resource request settings (but not limits) to not interfer with HVPA controller", func() {
			oldTimeNow := TimeNow
			defer func() { TimeNow = oldTimeNow }()
			TimeNow = func() time.Time { return now }

			var (
				existingResourcesContainerEtcd = corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("2G"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("3"),
						corev1.ResourceMemory: resource.MustParse("4G"),
					},
				}
				existingResourcesContainerBackupRestore = corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("5"),
						corev1.ResourceMemory: resource.MustParse("6G"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("7"),
						corev1.ResourceMemory: resource.MustParse("8G"),
					},
				}

				expectedResourcesContainerEtcd = corev1.ResourceRequirements{
					Requests: existingResourcesContainerEtcd.Requests,
				}
				expectedResourcesContainerBackupRestore = corev1.ResourceRequirements{
					Requests: existingResourcesContainerBackupRestore.Requests,
				}
			)

			gomock.InOrder(
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&appsv1.StatefulSet{})).DoAndReturn(func(ctx context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
					(&appsv1.StatefulSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      etcdName,
							Namespace: testNamespace,
						},
						Spec: appsv1.StatefulSetSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:      "etcd",
											Resources: existingResourcesContainerEtcd,
										},
										{
											Name:      "backup-restore",
											Resources: existingResourcesContainerBackupRestore,
										},
									},
								},
							},
						},
					}).DeepCopyInto(obj.(*appsv1.StatefulSet))
					return nil
				}),

				c.EXPECT().Get(ctx, kutil.Key(testNamespace, networkPolicyClientName), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
					Expect(obj).To(DeepEqual(clientNetworkPolicy))
				}),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
					Expect(obj).To(DeepEqual(etcdObjFor(
						class,
						1,
						nil,
						"",
						"",
						&expectedResourcesContainerEtcd,
						&expectedResourcesContainerBackupRestore,
						nil,
						secretNameCA,
						secretNameClient,
						secretNameServer,
						nil,
						nil)))
				}),
				c.EXPECT().Get(ctx, kutil.Key(testNamespace, hvpaName), gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{})),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
					Expect(obj).To(DeepEqual(hvpaFor(class, 1, scaleDownUpdateMode)))
				}),
			)

			Expect(etcd.Deploy(ctx)).To(Succeed())
		})

		for _, shootPurpose := range []gardencorev1beta1.ShootPurpose{gardencorev1beta1.ShootPurposeEvaluation, gardencorev1beta1.ShootPurposeProduction} {
			var purpose = shootPurpose
			It(fmt.Sprintf("should successfully deploy (important etcd): purpose = %q", purpose), func() {
				oldTimeNow := TimeNow
				defer func() { TimeNow = oldTimeNow }()
				TimeNow = func() time.Time { return now }

				class := ClassImportant

				updateMode := hvpav1alpha1.UpdateModeMaintenanceWindow
				if purpose == gardencorev1beta1.ShootPurposeProduction {
					updateMode = hvpav1alpha1.UpdateModeOff
				}

				replicas = pointer.Int32(1)

				etcd = New(c, log, testNamespace, sm, testRole, class, annotations, failureToleranceType, replicas, storageCapacity, &defragmentationSchedule, "", "1.20.1")
				newSetHVPAConfigFunc(updateMode)()

				gomock.InOrder(
					c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
					c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&appsv1.StatefulSet{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),

					c.EXPECT().Get(ctx, kutil.Key(testNamespace, networkPolicyClientName), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
						Expect(obj).To(DeepEqual(clientNetworkPolicy))
					}),
					c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
						Expect(obj).To(DeepEqual(etcdObjFor(
							class,
							1,
							nil,
							"",
							"",
							nil,
							nil,
							nil,
							secretNameCA,
							secretNameClient,
							secretNameServer,
							nil,
							nil)))
					}),
					c.EXPECT().Get(ctx, kutil.Key(testNamespace, hvpaName), gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
						Expect(obj).To(DeepEqual(hvpaFor(class, 1, updateMode)))
					}),
				)

				Expect(etcd.Deploy(ctx)).To(Succeed())
			})
		}

		Context("with backup", func() {
			var backupConfig = &BackupConfig{
				Provider:             "prov",
				SecretRefName:        "secret-key",
				Prefix:               "prefix",
				Container:            "bucket",
				FullSnapshotSchedule: "1234",
			}

			BeforeEach(func() {
				etcd.SetBackupConfig(backupConfig)
			})

			It("should successfully deploy (with backup)", func() {
				oldTimeNow := TimeNow
				defer func() { TimeNow = oldTimeNow }()
				TimeNow = func() time.Time { return now }

				gomock.InOrder(
					c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
					c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&appsv1.StatefulSet{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),

					c.EXPECT().Get(ctx, kutil.Key(testNamespace, networkPolicyClientName), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
						Expect(obj).To(DeepEqual(clientNetworkPolicy))
					}),
					c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
						Expect(obj).To(DeepEqual(etcdObjFor(
							class,
							1,
							backupConfig,
							"",
							"",
							nil,
							nil,
							nil,
							secretNameCA,
							secretNameClient,
							secretNameServer,
							nil,
							nil)))
					}),
					c.EXPECT().Get(ctx, kutil.Key(testNamespace, hvpaName), gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
						Expect(obj).To(DeepEqual(hvpaFor(class, 1, scaleDownUpdateMode)))
					}),
				)

				Expect(etcd.Deploy(ctx)).To(Succeed())
			})

			It("should successfully deploy (with backup) and keep the existing backup schedule", func() {
				oldTimeNow := TimeNow
				defer func() { TimeNow = oldTimeNow }()
				TimeNow = func() time.Time { return now }

				existingBackupSchedule := "foobarbackupexisting"

				gomock.InOrder(
					c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).DoAndReturn(func(ctx context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
						(&druidv1alpha1.Etcd{
							ObjectMeta: metav1.ObjectMeta{
								Name:      etcdName,
								Namespace: testNamespace,
							},
							Spec: druidv1alpha1.EtcdSpec{
								Backup: druidv1alpha1.BackupSpec{
									FullSnapshotSchedule: &existingBackupSchedule,
								},
							},
							Status: druidv1alpha1.EtcdStatus{
								Etcd: &druidv1alpha1.CrossVersionObjectReference{
									Name: "",
								},
							},
						}).DeepCopyInto(obj.(*druidv1alpha1.Etcd))
						return nil
					}),
					c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&appsv1.StatefulSet{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),

					c.EXPECT().Get(ctx, kutil.Key(testNamespace, networkPolicyClientName), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
						Expect(obj).To(DeepEqual(clientNetworkPolicy))
					}),
					c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
						expobj := etcdObjFor(
							class,
							1,
							backupConfig,
							"",
							existingBackupSchedule,
							nil,
							nil,
							nil,
							secretNameCA,
							secretNameClient,
							secretNameServer,
							nil,
							nil)
						expobj.Status.Etcd = &druidv1alpha1.CrossVersionObjectReference{}

						Expect(obj).To(DeepEqual(expobj))
					}),
					c.EXPECT().Get(ctx, kutil.Key(testNamespace, hvpaName), gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&hvpav1alpha1.Hvpa{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
						Expect(obj).To(DeepEqual(hvpaFor(class, 1, scaleDownUpdateMode)))
					}),
				)

				Expect(etcd.Deploy(ctx)).To(Succeed())
			})
		})

		Context("when HA setup is configured", func() {
			var (
				rotationPhase gardencorev1beta1.ShootCredentialsRotationPhase
			)

			createExpectations := func(failureToleranceType *gardencorev1beta1.FailureToleranceType, caSecretName, clientSecretName, serverSecretName, peerCASecretName, peerServerSecretName string) {
				gomock.InOrder(
					c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
					c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&appsv1.StatefulSet{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),

					c.EXPECT().Get(ctx, kutil.Key(testNamespace, networkPolicyClientName), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
						Expect(obj).To(DeepEqual(clientNetworkPolicy))
					}),
					c.EXPECT().Get(ctx, kutil.Key(testNamespace, networkPolicyPeerName), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
						Expect(obj).To(DeepEqual(peerNetworkPolicy))
					}),
					c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).DoAndReturn(
						func(_ context.Context, _ client.ObjectKey, etcd *druidv1alpha1.Etcd, _ ...client.GetOption) error {
							if peerServerSecretName != "" {
								etcd.Spec.Etcd.PeerUrlTLS = &druidv1alpha1.TLSConfig{
									ServerTLSSecretRef: corev1.SecretReference{
										Name:      peerServerSecretName,
										Namespace: testNamespace,
									},
								}
							}
							return nil
						}),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()).Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
						Expect(obj).To(DeepEqual(etcdObjFor(
							class,
							1,
							nil,
							"",
							"",
							nil,
							nil,
							failureToleranceType,
							caSecretName,
							clientSecretName,
							serverSecretName,
							&peerCASecretName,
							&peerServerSecretName,
						)))
					}),
					c.EXPECT().Delete(ctx, &hvpav1alpha1.Hvpa{ObjectMeta: metav1.ObjectMeta{Name: "etcd-" + testRole, Namespace: testNamespace}}),
				)
			}

			BeforeEach(func() {
				Expect(fakeClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ca-etcd-peer", Namespace: testNamespace}})).To(Succeed())
			})

			JustBeforeEach(func() {
				etcd = New(c, log, testNamespace, sm, testRole, class, annotations, failureToleranceType, replicas, storageCapacity, &defragmentationSchedule, rotationPhase, "1.20.1")
			})

			Context("when CA rotation phase is in `Preparing` state", func() {
				var (
					clientCASecret *corev1.Secret
					peerCASecret   *corev1.Secret
				)

				BeforeEach(func() {
					annotations = map[string]string{}
					failureToleranceType = getFailureToleranceTypeRef(gardencorev1beta1.FailureToleranceTypeNode)
					rotationPhase = gardencorev1beta1.RotationPreparing

					secretNamesToTimes := map[string]time.Time{}

					// A "real" SecretsManager is needed here because in further tests we want to differentiate
					// between what was issued by the old and new CAs.
					var err error
					sm, err = secretsmanager.New(
						ctx,
						logr.New(logf.NullLogSink{}),
						testclock.NewFakeClock(time.Now()),
						fakeClient,
						testNamespace,
						"",
						secretsmanager.Config{
							SecretNamesToTimes: secretNamesToTimes,
						})
					Expect(err).ToNot(HaveOccurred())

					// Create new etcd CA
					_, err = sm.Generate(ctx,
						&secretutils.CertificateSecretConfig{Name: v1beta1constants.SecretNameCAETCD, CommonName: "etcd", CertType: secretutils.CACert})
					Expect(err).ToNot(HaveOccurred())

					// Create new peer CA
					_, err = sm.Generate(ctx,
						&secretutils.CertificateSecretConfig{Name: v1beta1constants.SecretNameCAETCDPeer, CommonName: "etcd-peer", CertType: secretutils.CACert})
					Expect(err).ToNot(HaveOccurred())

					// Set time to trigger CA rotation
					secretNamesToTimes[v1beta1constants.SecretNameCAETCDPeer] = now

					// Rotate CA
					_, err = sm.Generate(ctx,
						&secretutils.CertificateSecretConfig{Name: v1beta1constants.SecretNameCAETCDPeer, CommonName: "etcd-peer", CertType: secretutils.CACert},
						secretsmanager.Rotate(secretsmanager.KeepOld))
					Expect(err).ToNot(HaveOccurred())

					var ok bool
					clientCASecret, ok = sm.Get(v1beta1constants.SecretNameCAETCD)
					Expect(ok).To(BeTrue())

					peerCASecret, ok = sm.Get(v1beta1constants.SecretNameCAETCDPeer)
					Expect(ok).To(BeTrue())

					DeferCleanup(func() {
						rotationPhase = ""
					})
				})

				It("should successfully deploy", func() {
					oldTimeNow := TimeNow
					defer func() { TimeNow = oldTimeNow }()
					TimeNow = func() time.Time { return now }

					peerServerSecret, err := sm.Generate(ctx, &secretutils.CertificateSecretConfig{
						Name:       "etcd-peer-server-" + testRole,
						CommonName: "etcd-server",
						DNSNames: []string{
							"etcd-test-peer",
							"etcd-test-peer.shoot--test--test",
							"etcd-test-peer.shoot--test--test.svc",
							"etcd-test-peer.shoot--test--test.svc.cluster.local",
							"*.etcd-test-peer",
							"*.etcd-test-peer.shoot--test--test",
							"*.etcd-test-peer.shoot--test--test.svc",
							"*.etcd-test-peer.shoot--test--test.svc.cluster.local",
						},
						CertType:                    secretutils.ServerClientCert,
						SkipPublishingCACertificate: true,
					}, secretsmanager.SignedByCA(v1beta1constants.SecretNameCAETCDPeer, secretsmanager.UseCurrentCA), secretsmanager.Rotate(secretsmanager.InPlace))
					Expect(err).ToNot(HaveOccurred())

					clientSecret, err := sm.Generate(ctx, &secretutils.CertificateSecretConfig{
						Name:                        SecretNameClient,
						CommonName:                  "etcd-client",
						CertType:                    secretutils.ClientCert,
						SkipPublishingCACertificate: true,
					}, secretsmanager.SignedByCA(v1beta1constants.SecretNameCAETCD), secretsmanager.Rotate(secretsmanager.InPlace))
					Expect(err).ToNot(HaveOccurred())

					serverSecret, err := sm.Generate(ctx, &secretutils.CertificateSecretConfig{
						Name:       "etcd-server-" + testRole,
						CommonName: "etcd-server",
						DNSNames: []string{
							"etcd-test-local",
							"etcd-test-client",
							"etcd-test-client.shoot--test--test",
							"etcd-test-client.shoot--test--test.svc",
							"etcd-test-client.shoot--test--test.svc.cluster.local",
							"*.etcd-test-peer",
							"*.etcd-test-peer.shoot--test--test",
							"*.etcd-test-peer.shoot--test--test.svc",
							"*.etcd-test-peer.shoot--test--test.svc.cluster.local",
						},
						CertType:                    secretutils.ServerClientCert,
						SkipPublishingCACertificate: true,
					}, secretsmanager.SignedByCA(v1beta1constants.SecretNameCAETCD), secretsmanager.Rotate(secretsmanager.InPlace))
					Expect(err).ToNot(HaveOccurred())

					createExpectations(failureToleranceType, clientCASecret.Name, clientSecret.Name, serverSecret.Name, peerCASecret.Name, peerServerSecret.Name)

					Expect(etcd.Deploy(ctx)).To(Succeed())
				})
			})

			Context("when configured for single-zone", func() {
				BeforeEach(func() {
					annotations = map[string]string{}
					failureToleranceType = getFailureToleranceTypeRef(gardencorev1beta1.FailureToleranceTypeNode)
				})

				It("should successfully deploy", func() {
					oldTimeNow := TimeNow
					defer func() { TimeNow = oldTimeNow }()
					TimeNow = func() time.Time { return now }

					createExpectations(failureToleranceType, secretNameCA, secretNameClient, secretNameServer, secretNamePeerCA, secretNameServerPeer)

					Expect(etcd.Deploy(ctx)).To(Succeed())
				})
			})

			Context("when configured for multi-zone", func() {
				BeforeEach(func() {
					annotations = map[string]string{}
					failureToleranceType = getFailureToleranceTypeRef(gardencorev1beta1.FailureToleranceTypeZone)
				})

				It("should successfully deploy", func() {
					oldTimeNow := TimeNow
					defer func() { TimeNow = oldTimeNow }()
					TimeNow = func() time.Time { return now }

					createExpectations(failureToleranceType, secretNameCA, secretNameClient, secretNameServer, secretNamePeerCA, secretNameServerPeer)

					Expect(etcd.Deploy(ctx)).To(Succeed())
				})
			})
		})
	})

	Describe("#Destroy", func() {
		var (
			etcdRes                   *druidv1alpha1.Etcd
			nowFunc                   func() time.Time
			zoneAnnotations           map[string]string
			shootFailureToleranceType *gardencorev1beta1.FailureToleranceType
		)

		JustBeforeEach(func() {
			etcd = New(c, log, testNamespace, sm, testRole, class, zoneAnnotations, shootFailureToleranceType, replicas, storageCapacity, &defragmentationSchedule, "", "1.20.1")
		})

		BeforeEach(func() {
			zoneAnnotations = make(map[string]string)
			nowFunc = func() time.Time {
				return time.Date(1, 1, 1, 1, 1, 1, 1, time.UTC)
			}
			etcdRes = &druidv1alpha1.Etcd{ObjectMeta: metav1.ObjectMeta{
				Name:      "etcd-" + testRole,
				Namespace: testNamespace,
				Annotations: map[string]string{
					"confirmation.gardener.cloud/deletion": "true",
					"gardener.cloud/timestamp":             nowFunc().String(),
				},
			}}
		})

		It("should properly delete all expected objects", func() {
			defer test.WithVar(&gardener.TimeNow, nowFunc)()
			gomock.InOrder(
				c.EXPECT().Patch(ctx, etcdRes, gomock.Any()),
				c.EXPECT().Delete(ctx, &hvpav1alpha1.Hvpa{ObjectMeta: metav1.ObjectMeta{Name: "etcd-" + testRole, Namespace: testNamespace}}),
				c.EXPECT().Delete(ctx, etcdRes),
				c.EXPECT().Delete(ctx, &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: networkPolicyClientName, Namespace: testNamespace}}),
			)
			Expect(etcd.Destroy(ctx)).To(Succeed())
		})

		It("should fail when the hvpa deletion fails", func() {
			defer test.WithVar(&gardener.TimeNow, nowFunc)()

			gomock.InOrder(
				c.EXPECT().Patch(ctx, etcdRes, gomock.Any()),
				c.EXPECT().Delete(ctx, &hvpav1alpha1.Hvpa{ObjectMeta: metav1.ObjectMeta{Name: "etcd-" + testRole, Namespace: testNamespace}}).Return(fakeErr),
			)

			Expect(etcd.Destroy(ctx)).To(MatchError(fakeErr))
		})

		It("should fail when the etcd deletion fails", func() {
			defer test.WithVar(&gardener.TimeNow, nowFunc)()

			gomock.InOrder(
				c.EXPECT().Patch(ctx, etcdRes, gomock.Any()),
				c.EXPECT().Delete(ctx, &hvpav1alpha1.Hvpa{ObjectMeta: metav1.ObjectMeta{Name: "etcd-" + testRole, Namespace: testNamespace}}),
				c.EXPECT().Delete(ctx, etcdRes).Return(fakeErr),
			)

			Expect(etcd.Destroy(ctx)).To(MatchError(fakeErr))
		})

		It("should fail when the network policy deletion fails", func() {
			defer test.WithVar(&gardener.TimeNow, nowFunc)()

			gomock.InOrder(
				c.EXPECT().Patch(ctx, etcdRes, gomock.Any()),
				c.EXPECT().Delete(ctx, &hvpav1alpha1.Hvpa{ObjectMeta: metav1.ObjectMeta{Name: "etcd-" + testRole, Namespace: testNamespace}}),
				c.EXPECT().Delete(ctx, etcdRes),
				c.EXPECT().Delete(ctx, &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: networkPolicyClientName, Namespace: testNamespace}}).Return(fakeErr),
			)

			Expect(etcd.Destroy(ctx)).To(MatchError(fakeErr))
		})
	})

	Describe("#Snapshot", func() {
		It("should return an error when the backup config is nil", func() {
			Expect(etcd.Snapshot(ctx, nil)).To(MatchError(ContainSubstring("no backup is configured")))
		})

		Context("w/ backup configuration", func() {
			var (
				podExecutor *mockkubernetes.MockPodExecutor
				podName     = "some-etcd-pod"
				selector    = labels.SelectorFromSet(map[string]string{
					"app":  "etcd-statefulset",
					"role": testRole,
				})
			)

			BeforeEach(func() {
				etcd.SetBackupConfig(&BackupConfig{})
				podExecutor = mockkubernetes.NewMockPodExecutor(ctrl)
			})

			It("should successfully execute the full snapshot command", func() {
				podList := &corev1.PodList{
					Items: []corev1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: podName,
							},
						},
					},
				}

				c.EXPECT().List(
					ctx,
					gomock.AssignableToTypeOf(&corev1.PodList{}),
					client.InNamespace(testNamespace),
					client.MatchingLabelsSelector{Selector: selector},
				).DoAndReturn(
					func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
						podList.DeepCopyInto(list.(*corev1.PodList))
						return nil
					},
				)

				podExecutor.EXPECT().Execute(
					testNamespace,
					podName,
					"backup-restore",
					"/bin/sh",
					"curl -k https://etcd-"+testRole+"-local:8080/snapshot/full?final=true",
				)

				Expect(etcd.Snapshot(ctx, podExecutor)).To(Succeed())
			})

			It("should return an error when the pod listing fails", func() {
				c.EXPECT().List(
					ctx,
					gomock.AssignableToTypeOf(&corev1.PodList{}),
					client.InNamespace(testNamespace),
					client.MatchingLabelsSelector{Selector: selector},
				).Return(fakeErr)

				Expect(etcd.Snapshot(ctx, podExecutor)).To(MatchError(fakeErr))
			})

			It("should return an error when the pod list is empty", func() {
				podList := &corev1.PodList{}

				c.EXPECT().List(
					ctx,
					gomock.AssignableToTypeOf(&corev1.PodList{}),
					client.InNamespace(testNamespace),
					client.MatchingLabelsSelector{Selector: selector},
				).DoAndReturn(
					func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
						podList.DeepCopyInto(list.(*corev1.PodList))
						return nil
					},
				)

				Expect(etcd.Snapshot(ctx, podExecutor)).To(MatchError(ContainSubstring("didn't find any pods")))
			})

			It("should return an error when the execution command fails", func() {
				podList := &corev1.PodList{
					Items: []corev1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: podName,
							},
						},
					},
				}

				c.EXPECT().List(
					ctx,
					gomock.AssignableToTypeOf(&corev1.PodList{}),
					client.InNamespace(testNamespace),
					client.MatchingLabelsSelector{Selector: selector},
				).DoAndReturn(
					func(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
						podList.DeepCopyInto(obj.(*corev1.PodList))
						return nil
					},
				)

				podExecutor.EXPECT().Execute(
					testNamespace,
					podName,
					"backup-restore",
					"/bin/sh",
					"curl -k https://etcd-"+testRole+"-local:8080/snapshot/full?final=true",
				).Return(nil, fakeErr)

				Expect(etcd.Snapshot(ctx, podExecutor)).To(MatchError(fakeErr))
			})
		})
	})

	Describe("#Scale", func() {
		var etcdObj *druidv1alpha1.Etcd

		BeforeEach(func() {
			etcdObj = &druidv1alpha1.Etcd{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd-test",
					Namespace: testNamespace,
				},
			}
		})

		It("should scale ETCD from 0 to 1", func() {
			etcdObj.Spec.Replicas = 0

			nowFunc := func() time.Time {
				return now
			}
			defer test.WithVar(&TimeNow, nowFunc)()

			c.EXPECT().Get(ctx, client.ObjectKeyFromObject(etcdObj), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).DoAndReturn(
				func(_ context.Context, _ client.ObjectKey, etcd *druidv1alpha1.Etcd, _ ...client.GetOption) error {
					*etcd = *etcdObj
					return nil
				},
			)

			c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()).DoAndReturn(
				func(_ context.Context, etcd *druidv1alpha1.Etcd, patch client.Patch, _ ...client.PatchOption) error {
					data, err := patch.Data(etcd)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(data)).To(Equal(fmt.Sprintf(`{"metadata":{"annotations":{"gardener.cloud/operation":"reconcile","gardener.cloud/timestamp":"%s"}},"spec":{"replicas":1}}`, now.String())))
					return nil
				})

			Expect(etcd.Scale(ctx, 1)).To(Succeed())
		})

		It("should set operation annotation when replica count is unchanged", func() {
			etcdObj.Spec.Replicas = 1

			nowFunc := func() time.Time {
				return now
			}
			defer test.WithVar(&TimeNow, nowFunc)()

			c.EXPECT().Get(ctx, client.ObjectKeyFromObject(etcdObj), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).DoAndReturn(
				func(_ context.Context, _ client.ObjectKey, etcd *druidv1alpha1.Etcd, _ ...client.GetOption) error {
					*etcd = *etcdObj
					return nil
				},
			)

			c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()).DoAndReturn(
				func(_ context.Context, etcd *druidv1alpha1.Etcd, patch client.Patch, _ ...client.PatchOption) error {
					data, err := patch.Data(etcd)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(data)).To(Equal(fmt.Sprintf(`{"metadata":{"annotations":{"gardener.cloud/operation":"reconcile","gardener.cloud/timestamp":"%s"}}}`, now.String())))
					return nil
				})

			Expect(etcd.Scale(ctx, 1)).To(Succeed())
		})

		It("should fail if GardenerTimestamp is unexpected", func() {
			nowFunc := func() time.Time {
				return now
			}
			defer test.WithVar(&TimeNow, nowFunc)()

			gomock.InOrder(
				c.EXPECT().Get(ctx, client.ObjectKeyFromObject(etcdObj), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, etcd *druidv1alpha1.Etcd, _ ...client.GetOption) error {
						*etcd = *etcdObj
						return nil
					},
				),
				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()),
				c.EXPECT().Get(ctx, client.ObjectKeyFromObject(etcdObj), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).DoAndReturn(
					func(_ context.Context, _ client.ObjectKey, etcd *druidv1alpha1.Etcd, _ ...client.GetOption) error {
						etcdObj.Annotations = map[string]string{
							v1beta1constants.GardenerTimestamp: "foo",
						}
						*etcd = *etcdObj
						return nil
					},
				),
			)

			Expect(etcd.Scale(ctx, 1)).To(Succeed())
			Expect(etcd.Scale(ctx, 1)).Should(MatchError(`object's "gardener.cloud/timestamp" annotation is not "0001-01-01 00:00:00 +0000 UTC" but "foo"`))
		})

		It("should fail because operation annotation is set", func() {
			c.EXPECT().Get(ctx, client.ObjectKeyFromObject(etcdObj), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).DoAndReturn(
				func(_ context.Context, _ client.ObjectKey, etcd *druidv1alpha1.Etcd, _ ...client.GetOption) error {
					etcdObj.Annotations = map[string]string{
						v1beta1constants.GardenerOperation: v1beta1constants.GardenerOperationReconcile,
					}
					*etcd = *etcdObj
					return nil
				},
			)

			Expect(etcd.Scale(ctx, 1)).Should(MatchError(`etcd object still has operation annotation set`))
		})
	})

	Describe("#RolloutPeerCA", func() {
		var failureToleranceTypeZone *gardencorev1beta1.FailureToleranceType

		JustBeforeEach(func() {
			etcd = New(c, log, testNamespace, sm, testRole, class, annotations, failureToleranceTypeZone, replicas, storageCapacity, &defragmentationSchedule, "", "1.20.1")
		})

		Context("when HA control-plane is not requested", func() {
			BeforeEach(func() {
				failureToleranceTypeZone = nil
			})

			It("should do nothing and succeed without expectations", func() {
				Expect(etcd.RolloutPeerCA(ctx)).To(Succeed())
			})
		})

		Context("when HA control-plane is requested", func() {
			createEtcdObj := func(caName string) *druidv1alpha1.Etcd {
				return &druidv1alpha1.Etcd{
					ObjectMeta: metav1.ObjectMeta{
						Name:      etcdName,
						Namespace: testNamespace,
					},
					Spec: druidv1alpha1.EtcdSpec{
						Etcd: druidv1alpha1.EtcdConfig{
							PeerUrlTLS: &druidv1alpha1.TLSConfig{
								TLSCASecretRef: druidv1alpha1.SecretReference{
									SecretReference: corev1.SecretReference{
										Name:      caName,
										Namespace: testNamespace,
									},
									DataKey: pointer.String(secretutils.DataKeyCertificateBundle),
								},
							},
						},
					},
				}
			}

			BeforeEach(func() {
				failureToleranceTypeZone = getFailureToleranceTypeRef(gardencorev1beta1.FailureToleranceTypeZone)
				DeferCleanup(test.WithVar(&TimeNow, func() time.Time { return now }))
			})

			It("should patch the etcd resource with the new peer CA secret name", func() {
				Expect(fakeClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ca-etcd-peer", Namespace: testNamespace}})).To(Succeed())

				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")).DoAndReturn(func(ctx context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
					createEtcdObj("old-ca").DeepCopyInto(obj.(*druidv1alpha1.Etcd))
					return nil
				})

				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()).DoAndReturn(
					func(_ context.Context, obj *druidv1alpha1.Etcd, patch client.Patch, _ ...client.PatchOption) error {
						data, err := patch.Data(obj)
						Expect(err).ToNot(HaveOccurred())
						Expect(data).To(MatchJSON("{\"metadata\":{\"annotations\":{\"gardener.cloud/operation\":\"reconcile\",\"gardener.cloud/timestamp\":\"0001-01-01 00:00:00 +0000 UTC\"}},\"spec\":{\"etcd\":{\"peerUrlTls\":{\"tlsCASecretRef\":{\"name\":\"ca-etcd-peer\"}}}}}"))
						return nil
					})

				Expect(etcd.RolloutPeerCA(ctx)).To(Succeed())
			})

			It("should not patch anything because the expected CA ref is already configured", func() {
				peerCAName := "ca-etcd-peer"

				Expect(fakeClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: peerCAName, Namespace: testNamespace}})).To(Succeed())

				c.EXPECT().Get(ctx, kutil.Key(testNamespace, etcdName), gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")).DoAndReturn(func(ctx context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
					createEtcdObj(peerCAName).DeepCopyInto(obj.(*druidv1alpha1.Etcd))
					return nil
				})

				c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&druidv1alpha1.Etcd{}), gomock.Any()).DoAndReturn(
					func(_ context.Context, obj *druidv1alpha1.Etcd, patch client.Patch, _ ...client.PatchOption) error {
						data, err := patch.Data(obj)
						Expect(err).ToNot(HaveOccurred())
						Expect(data).To(MatchJSON("{}"))
						return nil
					})

				Expect(etcd.RolloutPeerCA(ctx)).To(Succeed())
			})

			It("should fail because CA cannot be found", func() {
				Expect(etcd.RolloutPeerCA(ctx)).To(MatchError("secret \"ca-etcd-peer\" not found"))
			})
		})
	})
})

func getFailureToleranceTypeRef(f gardencorev1beta1.FailureToleranceType) *gardencorev1beta1.FailureToleranceType {
	return &f
}
