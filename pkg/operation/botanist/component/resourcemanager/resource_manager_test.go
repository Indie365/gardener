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

package resourcemanager_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"
	"github.com/gardener/gardener/pkg/operation/botanist/component"
	. "github.com/gardener/gardener/pkg/operation/botanist/component/resourcemanager"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	fakesecretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager/fake"
	"github.com/gardener/gardener/pkg/utils/test"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"

	"github.com/Masterminds/semver"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	kubernetesscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("ResourceManager", func() {
	var (
		ctrl            *gomock.Controller
		c               *mockclient.MockClient
		fakeClient      client.Client
		sm              secretsmanager.Interface
		resourceManager Interface

		ctx                               = context.TODO()
		deployNamespace                   = "fake-ns"
		fakeErr                           = fmt.Errorf("fake error")
		image                             = "fake-image"
		replicas                    int32 = 1
		healthPort                  int32 = 8081
		metricsPort                 int32 = 8080
		serverPort                        = 10250
		version                           = semver.MustParse("1.22.1")
		binPackingSchedulingProfile       = gardencorev1beta1.SchedulingProfileBinPacking

		// optional configuration
		clusterIdentity                      = "foo"
		secretNameServer                     = "gardener-resource-manager-server"
		secretMountPathServer                = "/etc/gardener-resource-manager-tls"
		secretMountPathRootCA                = "/etc/gardener-resource-manager-root-ca"
		secretMountPathAPIAccess             = "/var/run/secrets/kubernetes.io/serviceaccount"
		secrets                              Secrets
		alwaysUpdate                               = true
		concurrentSyncs                      int32 = 20
		genericTokenKubeconfigSecretName           = "generic-token-kubeconfig"
		clusterRoleName                            = "gardener-resource-manager-seed"
		healthSyncPeriod                           = time.Minute
		leaseDuration                              = time.Second * 40
		maxConcurrentHealthWorkers           int32 = 20
		maxConcurrentTokenInvalidatorWorkers int32 = 23
		maxConcurrentTokenRequestorWorkers   int32 = 21
		maxConcurrentRootCAPublisherWorkers  int32 = 22
		renewDeadline                              = time.Second * 10
		resourceClass                              = "fake-ResourceClass"
		retryPeriod                                = time.Second * 20
		syncPeriod                                 = time.Second * 80
		watchedNamespace                           = "fake-ns"
		targetDisableCache                         = true
		maxUnavailable                             = intstr.FromInt(1)
		failurePolicy                              = admissionregistrationv1.Fail
		matchPolicy                                = admissionregistrationv1.Exact
		sideEffect                                 = admissionregistrationv1.SideEffectClassNone
		networkPolicyProtocol                      = corev1.ProtocolTCP
		networkPolicyPort                          = intstr.FromInt(serverPort)

		allowAll                     []rbacv1.PolicyRule
		allowManagedResources        []rbacv1.PolicyRule
		cfg                          Values
		clusterRole                  *rbacv1.ClusterRole
		clusterRoleBinding           *rbacv1.ClusterRoleBinding
		deployment                   *appsv1.Deployment
		computeArgs                  func(watchedNamespace *string, targetKubeconfig *string) []string
		deploymentFor                func(kubernetesVersion *semver.Version, watchedNamespace *string, targetKubeconfig *string, targetClusterDiffersFromSourceCluster bool) *appsv1.Deployment
		defaultLabels                map[string]string
		roleBinding                  *rbacv1.RoleBinding
		role                         *rbacv1.Role
		secret                       *corev1.Secret
		service                      *corev1.Service
		serviceAccount               *corev1.ServiceAccount
		updateMode                   = vpaautoscalingv1.UpdateModeAuto
		controlledValues             = vpaautoscalingv1.ContainerControlledValuesRequestsOnly
		pdbV1beta1                   *policyv1beta1.PodDisruptionBudget
		pdbV1                        *policyv1.PodDisruptionBudget
		vpa                          *vpaautoscalingv1.VerticalPodAutoscaler
		mutatingWebhookConfiguration *admissionregistrationv1.MutatingWebhookConfiguration
		managedResourceSecret        *corev1.Secret
		managedResource              *resourcesv1alpha1.ManagedResource
		networkPolicy                *networkingv1.NetworkPolicy
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		c = mockclient.NewMockClient(ctrl)
		fakeClient = fakeclient.NewClientBuilder().WithScheme(kubernetesscheme.Scheme).Build()
		sm = fakesecretsmanager.New(fakeClient, deployNamespace)

		By("creating secrets managed outside of this package for whose secretsmanager.Get() will be called")
		Expect(fakeClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ca", Namespace: deployNamespace}})).To(Succeed())
		Expect(fakeClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "generic-token-kubeconfig", Namespace: deployNamespace}})).To(Succeed())

		secrets = Secrets{}
		allowAll = []rbacv1.PolicyRule{{
			APIGroups: []string{"*"},
			Resources: []string{"*"},
			Verbs:     []string{"*"},
		}}
		allowManagedResources = []rbacv1.PolicyRule{
			{
				APIGroups: []string{"resources.gardener.cloud"},
				Resources: []string{"managedresources", "managedresources/status"},
				Verbs:     []string{"get", "list", "watch", "update", "patch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "watch", "update", "patch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps", "events"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{"gardener-resource-manager"},
				Verbs:         []string{"get", "watch", "update", "patch"},
			},
			{
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{"leases"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups:     []string{"coordination.k8s.io"},
				Resources:     []string{"leases"},
				ResourceNames: []string{"gardener-resource-manager"},
				Verbs:         []string{"get", "watch", "update"},
			},
		}
		defaultLabels = map[string]string{
			v1beta1constants.GardenRole: v1beta1constants.GardenRoleControlPlane,
			v1beta1constants.LabelApp:   "gardener-resource-manager",
		}

		clusterRole = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:   clusterRoleName,
				Labels: defaultLabels,
			},
			Rules: allowAll}
		clusterRoleBinding = &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:   clusterRoleName,
				Labels: defaultLabels,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     clusterRoleName,
			},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      "gardener-resource-manager",
				Namespace: deployNamespace,
			}}}
		role = &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: deployNamespace,
				Name:      "gardener-resource-manager",
				Labels:    defaultLabels,
			},
			Rules: allowManagedResources}
		roleBinding = &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: deployNamespace,
				Name:      "gardener-resource-manager",
				Labels:    defaultLabels,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     "gardener-resource-manager",
			},
			Subjects: []rbacv1.Subject{{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      "gardener-resource-manager",
				Namespace: deployNamespace,
			}}}

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shoot-access-gardener-resource-manager",
				Namespace: deployNamespace,
				Annotations: map[string]string{
					"serviceaccount.resources.gardener.cloud/name":                      "gardener-resource-manager",
					"serviceaccount.resources.gardener.cloud/namespace":                 "kube-system",
					"serviceaccount.resources.gardener.cloud/token-expiration-duration": "24h",
				},
				Labels: map[string]string{
					"resources.gardener.cloud/purpose": "token-requestor",
				},
			},
			Type: corev1.SecretTypeOpaque,
		}

		service = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gardener-resource-manager",
				Namespace: deployNamespace,
				Labels:    defaultLabels},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app": "gardener-resource-manager"},
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{
						Name:     "metrics",
						Port:     metricsPort,
						Protocol: corev1.ProtocolTCP,
					},
					{
						Name:     "health",
						Port:     healthPort,
						Protocol: corev1.ProtocolTCP,
					},
					{
						Name:       "server",
						Port:       443,
						TargetPort: intstr.FromInt(serverPort),
						Protocol:   corev1.ProtocolTCP,
					},
				},
			},
		}
		cfg = Values{
			AlwaysUpdate:                         &alwaysUpdate,
			ClusterIdentity:                      &clusterIdentity,
			ConcurrentSyncs:                      &concurrentSyncs,
			HealthSyncPeriod:                     &healthSyncPeriod,
			LeaseDuration:                        &leaseDuration,
			MaxConcurrentHealthWorkers:           &maxConcurrentHealthWorkers,
			MaxConcurrentTokenInvalidatorWorkers: &maxConcurrentTokenInvalidatorWorkers,
			MaxConcurrentTokenRequestorWorkers:   &maxConcurrentTokenRequestorWorkers,
			MaxConcurrentRootCAPublisherWorkers:  &maxConcurrentRootCAPublisherWorkers,
			RenewDeadline:                        &renewDeadline,
			Replicas:                             &replicas,
			ResourceClass:                        &resourceClass,
			RetryPeriod:                          &retryPeriod,
			SecretNameServerCA:                   "ca",
			SyncPeriod:                           &syncPeriod,
			TargetDiffersFromSourceCluster:       true,
			TargetDisableCache:                   &targetDisableCache,
			Version:                              version,
			WatchedNamespace:                     &watchedNamespace,
			VPA: &VPAConfig{
				MinAllowed: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("20m"),
					corev1.ResourceMemory: resource.MustParse("30Mi"),
				},
			},
			SchedulingProfile:                   &binPackingSchedulingProfile,
			DefaultSeccompProfileEnabled:        true,
			PodTopologySpreadConstraintsEnabled: true,
			PodZoneAffinityEnabled:              true,
		}
		resourceManager = New(c, deployNamespace, sm, image, cfg)
		resourceManager.SetSecrets(secrets)

		serviceAccount = &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "gardener-resource-manager",
				Namespace: deployNamespace,
				Labels:    defaultLabels,
			},
			AutomountServiceAccountToken: pointer.Bool(false),
		}

		computeArgs = func(watchedNamespace *string, targetKubeconfig *string) []string {
			cmd := []string{
				"--always-update=true",
				"--cluster-id=" + clusterIdentity,
				"--garbage-collector-sync-period=12h",
				fmt.Sprintf("--health-bind-address=:%v", healthPort),
				fmt.Sprintf("--health-max-concurrent-workers=%v", maxConcurrentHealthWorkers),
				fmt.Sprintf("--token-requestor-max-concurrent-workers=%v", maxConcurrentTokenRequestorWorkers),
				fmt.Sprintf("--token-invalidator-max-concurrent-workers=%v", maxConcurrentTokenInvalidatorWorkers),
				fmt.Sprintf("--root-ca-publisher-max-concurrent-workers=%v", maxConcurrentRootCAPublisherWorkers),
				fmt.Sprintf("--root-ca-file=%s/bundle.crt", secretMountPathRootCA),
				fmt.Sprintf("--health-sync-period=%v", healthSyncPeriod),
				"--leader-election=true",
				fmt.Sprintf("--leader-election-lease-duration=%v", leaseDuration),
				fmt.Sprintf("--leader-election-namespace=%v", deployNamespace),
				fmt.Sprintf("--leader-election-renew-deadline=%v", renewDeadline),
				fmt.Sprintf("--leader-election-retry-period=%v", retryPeriod),
				fmt.Sprintf("--max-concurrent-workers=%v", concurrentSyncs),
				fmt.Sprintf("--metrics-bind-address=:%v", metricsPort),
			}
			if watchedNamespace != nil {
				cmd = append(cmd, fmt.Sprintf("--namespace=%v", *watchedNamespace))
			}
			cmd = append(cmd,
				fmt.Sprintf("--resource-class=%v", resourceClass),
				fmt.Sprintf("--sync-period=%v", syncPeriod),
				"--target-disable-cache",
				fmt.Sprintf("--port=%d", serverPort),
				fmt.Sprintf("--tls-cert-dir=%v", secretMountPathServer),
			)
			if targetKubeconfig != nil {
				cmd = append(cmd, fmt.Sprintf("--target-kubeconfig=%v", *targetKubeconfig))
			}
			cmd = append(cmd,
				"--pod-scheduler-name-webhook-enabled=true",
				fmt.Sprintf("--pod-scheduler-name-webhook-scheduler=%v", "bin-packing-scheduler"),
			)
			cmd = append(cmd, "--pod-topology-spread-constraints-webhook-enabled=true")
			cmd = append(cmd, "--pod-zone-affinity-webhook-enabled=true")
			cmd = append(cmd, "--seccomp-profile-webhook-enabled=true")

			return cmd
		}
		deploymentFor = func(kubernetesVersion *semver.Version, watchedNamespace *string, targetKubeconfig *string, targetClusterDiffersFromSourceCluster bool) *appsv1.Deployment {
			priorityClassName := v1beta1constants.PriorityClassNameSeedSystemCritical
			if targetClusterDiffersFromSourceCluster {
				priorityClassName = v1beta1constants.PriorityClassNameShootControlPlane400
			}

			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      v1beta1constants.DeploymentNameGardenerResourceManager,
					Namespace: deployNamespace,
					Labels:    defaultLabels,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas:             pointer.Int32(1),
					RevisionHistoryLimit: pointer.Int32(1),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "gardener-resource-manager",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"projected-token-mount.resources.gardener.cloud/skip": "true",
								"networking.gardener.cloud/to-dns":                    "allowed",
								"networking.gardener.cloud/to-seed-apiserver":         "allowed",
								"networking.gardener.cloud/from-prometheus":           "allowed",
								"networking.gardener.cloud/to-shoot-apiserver":        "allowed",
								"networking.gardener.cloud/from-shoot-apiserver":      "allowed",
								v1beta1constants.GardenRole:                           v1beta1constants.GardenRoleControlPlane,
								v1beta1constants.LabelApp:                             "gardener-resource-manager",
							},
						},
						Spec: corev1.PodSpec{
							Affinity: &corev1.Affinity{
								PodAntiAffinity: &corev1.PodAntiAffinity{
									PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
										{
											Weight: 100,
											PodAffinityTerm: corev1.PodAffinityTerm{
												TopologyKey: corev1.LabelHostname,
												LabelSelector: &metav1.LabelSelector{
													MatchLabels: map[string]string{
														v1beta1constants.GardenRole: v1beta1constants.GardenRoleControlPlane,
														v1beta1constants.LabelApp:   "gardener-resource-manager",
													},
												},
											},
										},
									},
								},
							},
							PriorityClassName: priorityClassName,
							SecurityContext: &corev1.PodSecurityContext{
								FSGroup: pointer.Int64(65532),
								SeccompProfile: &corev1.SeccompProfile{
									Type: corev1.SeccompProfileTypeRuntimeDefault,
								},
							},
							ServiceAccountName: "gardener-resource-manager",
							Containers: []corev1.Container{
								{
									Args:            computeArgs(watchedNamespace, targetKubeconfig),
									Image:           image,
									ImagePullPolicy: corev1.PullIfNotPresent,
									LivenessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Path:   "/healthz",
												Scheme: "HTTP",
												Port:   intstr.FromInt(int(healthPort)),
											},
										},
										InitialDelaySeconds: 30,
										FailureThreshold:    5,
										PeriodSeconds:       10,
										SuccessThreshold:    1,
										TimeoutSeconds:      5,
									},
									Name: "gardener-resource-manager",
									Ports: []corev1.ContainerPort{
										{
											Name:          "metrics",
											ContainerPort: metricsPort,
											Protocol:      corev1.ProtocolTCP,
										},
										{
											Name:          "health",
											ContainerPort: healthPort,
											Protocol:      corev1.ProtocolTCP,
										},
									},
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Path:   "/readyz",
												Scheme: "HTTP",
												Port:   intstr.FromInt(int(healthPort)),
											},
										},
										InitialDelaySeconds: 10,
									},
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("23m"),
											corev1.ResourceMemory: resource.MustParse("47Mi"),
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											MountPath: secretMountPathAPIAccess,
											Name:      "kube-api-access-gardener",
											ReadOnly:  true,
										},
										{
											MountPath: secretMountPathServer,
											Name:      "tls",
											ReadOnly:  true,
										},

										{
											MountPath: secretMountPathRootCA,
											Name:      "root-ca",
											ReadOnly:  true,
										},
										{
											Name:      "kubeconfig",
											MountPath: "/var/run/secrets/gardener.cloud/shoot/generic-kubeconfig",
											ReadOnly:  true,
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "kube-api-access-gardener",
									VolumeSource: corev1.VolumeSource{
										Projected: &corev1.ProjectedVolumeSource{
											DefaultMode: pointer.Int32(420),
											Sources: []corev1.VolumeProjection{
												{
													ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
														ExpirationSeconds: pointer.Int64(43200),
														Path:              "token",
													},
												},
												{
													ConfigMap: &corev1.ConfigMapProjection{
														LocalObjectReference: corev1.LocalObjectReference{
															Name: "kube-root-ca.crt",
														},
														Items: []corev1.KeyToPath{{
															Key:  "ca.crt",
															Path: "ca.crt",
														}},
													},
												},
												{
													DownwardAPI: &corev1.DownwardAPIProjection{
														Items: []corev1.DownwardAPIVolumeFile{{
															FieldRef: &corev1.ObjectFieldSelector{
																APIVersion: "v1",
																FieldPath:  "metadata.namespace",
															},
															Path: "namespace",
														}},
													},
												},
											},
										},
									},
								},
								{
									Name: "tls",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName:  secretNameServer,
											DefaultMode: pointer.Int32(420),
										},
									},
								},
								{
									Name: "root-ca",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName:  "ca",
											DefaultMode: pointer.Int32(420),
										},
									},
								},
								{
									Name: "kubeconfig",
									VolumeSource: corev1.VolumeSource{
										Projected: &corev1.ProjectedVolumeSource{
											DefaultMode: pointer.Int32(420),
											Sources: []corev1.VolumeProjection{
												{
													Secret: &corev1.SecretProjection{
														LocalObjectReference: corev1.LocalObjectReference{
															Name: genericTokenKubeconfigSecretName,
														},
														Items: []corev1.KeyToPath{{
															Key:  "kubeconfig",
															Path: "kubeconfig",
														}},
														Optional: pointer.Bool(false),
													},
												},
												{
													Secret: &corev1.SecretProjection{
														LocalObjectReference: corev1.LocalObjectReference{
															Name: "shoot-access-gardener-resource-manager",
														},
														Items: []corev1.KeyToPath{{
															Key:  resourcesv1alpha1.DataKeyToken,
															Path: resourcesv1alpha1.DataKeyToken,
														}},
														Optional: pointer.Bool(false),
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			return deployment
		}
		vpa = &vpaautoscalingv1.VerticalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gardener-resource-manager-vpa",
				Namespace: deployNamespace,
				Labels:    defaultLabels,
			},
			Spec: vpaautoscalingv1.VerticalPodAutoscalerSpec{
				TargetRef: &autoscalingv1.CrossVersionObjectReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "gardener-resource-manager",
				},
				UpdatePolicy: &vpaautoscalingv1.PodUpdatePolicy{
					UpdateMode: &updateMode,
				},
				ResourcePolicy: &vpaautoscalingv1.PodResourcePolicy{
					ContainerPolicies: []vpaautoscalingv1.ContainerResourcePolicy{
						{
							ContainerName: vpaautoscalingv1.DefaultContainerResourcePolicy,
							MinAllowed: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("20m"),
								corev1.ResourceMemory: resource.MustParse("30Mi"),
							},
							MaxAllowed: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("4"),
								corev1.ResourceMemory: resource.MustParse("10G"),
							},
							ControlledValues: &controlledValues,
						},
					},
				},
			},
		}
		pdbV1beta1 = &policyv1beta1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gardener-resource-manager",
				Namespace: deployNamespace,
				Labels:    defaultLabels,
			},
			Spec: policyv1beta1.PodDisruptionBudgetSpec{
				MaxUnavailable: &maxUnavailable,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						v1beta1constants.GardenRole: v1beta1constants.GardenRoleControlPlane,
						v1beta1constants.LabelApp:   "gardener-resource-manager",
					},
				},
			},
		}
		pdbV1 = &policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gardener-resource-manager",
				Namespace: deployNamespace,
				Labels:    defaultLabels,
			},
			Spec: policyv1.PodDisruptionBudgetSpec{
				MaxUnavailable: &maxUnavailable,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						v1beta1constants.GardenRole: v1beta1constants.GardenRoleControlPlane,
						v1beta1constants.LabelApp:   "gardener-resource-manager",
					},
				},
			},
		}
		mutatingWebhookConfiguration = &admissionregistrationv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gardener-resource-manager",
				Namespace: deployNamespace,
				Labels: map[string]string{
					"app": "gardener-resource-manager",
					"remediation.webhook.shoot.gardener.cloud/exclude": "true",
				},
			},
			Webhooks: []admissionregistrationv1.MutatingWebhook{
				{
					Name: "token-invalidator.resources.gardener.cloud",
					Rules: []admissionregistrationv1.RuleWithOperations{{
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"secrets"},
						},
						Operations: []admissionregistrationv1.OperationType{"CREATE", "UPDATE"},
					}},
					NamespaceSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{{
							Key:      "gardener.cloud/purpose",
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   []string{"kube-system", "kubernetes-dashboard"},
						}},
					},
					ObjectSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"resources.gardener.cloud/purpose": "token-invalidator"},
					},
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Name:      "gardener-resource-manager",
							Namespace: deployNamespace,
							Path:      pointer.String("/webhooks/invalidate-service-account-token-secret"),
						},
					},
					AdmissionReviewVersions: []string{"v1beta1", "v1"},
					FailurePolicy:           &failurePolicy,
					MatchPolicy:             &matchPolicy,
					SideEffects:             &sideEffect,
					TimeoutSeconds:          pointer.Int32(10),
				},
				{
					Name: "projected-token-mount.resources.gardener.cloud",
					Rules: []admissionregistrationv1.RuleWithOperations{{
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
						Operations: []admissionregistrationv1.OperationType{"CREATE"},
					}},
					NamespaceSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{{
							Key:      "gardener.cloud/purpose",
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   []string{"kube-system", "kubernetes-dashboard"},
						}},
					},
					ObjectSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "projected-token-mount.resources.gardener.cloud/skip",
								Operator: metav1.LabelSelectorOpDoesNotExist,
							},
							{
								Key:      "app",
								Operator: metav1.LabelSelectorOpNotIn,
								Values:   []string{"gardener-resource-manager"},
							},
						},
					},
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Name:      "gardener-resource-manager",
							Namespace: deployNamespace,
							Path:      pointer.String("/webhooks/mount-projected-service-account-token"),
						},
					},
					AdmissionReviewVersions: []string{"v1beta1", "v1"},
					FailurePolicy:           &failurePolicy,
					MatchPolicy:             &matchPolicy,
					SideEffects:             &sideEffect,
					TimeoutSeconds:          pointer.Int32(10),
				},
				{
					Name: "seccomp-profile.resources.gardener.cloud",
					Rules: []admissionregistrationv1.RuleWithOperations{{
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
						Operations: []admissionregistrationv1.OperationType{"CREATE"},
					}},
					NamespaceSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{{
							Key:      "gardener.cloud/purpose",
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   []string{"kube-system", "kubernetes-dashboard"},
						}},
					},
					ObjectSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "seccompprofile.resources.gardener.cloud/skip",
								Operator: metav1.LabelSelectorOpDoesNotExist,
							},
							{
								Key:      "app",
								Operator: metav1.LabelSelectorOpNotIn,
								Values:   []string{"gardener-resource-manager"},
							},
						},
					},
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Name:      "gardener-resource-manager",
							Namespace: deployNamespace,
							Path:      pointer.String("/webhooks/seccomp-profile"),
						},
					},
					AdmissionReviewVersions: []string{"v1beta1", "v1"},
					FailurePolicy:           &failurePolicy,
					MatchPolicy:             &matchPolicy,
					SideEffects:             &sideEffect,
					TimeoutSeconds:          pointer.Int32(10),
				},
				{
					Name: "pod-topology-spread-constraints.resources.gardener.cloud",
					Rules: []admissionregistrationv1.RuleWithOperations{{
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
						Operations: []admissionregistrationv1.OperationType{"CREATE"},
					}},
					NamespaceSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{{
							Key:      "gardener.cloud/purpose",
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   []string{"kube-system", "kubernetes-dashboard"},
						}},
					},
					ObjectSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "app",
								Operator: metav1.LabelSelectorOpNotIn,
								Values:   []string{"gardener-resource-manager"},
							},
							{
								Key:      "topologyspreadconstraints.resources.gardener.cloud/skip",
								Operator: metav1.LabelSelectorOpDoesNotExist,
							},
						},
					},
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Name:      "gardener-resource-manager",
							Namespace: deployNamespace,
							Path:      pointer.String("/webhooks/pod-topology-spread-constraints"),
						},
					},
					AdmissionReviewVersions: []string{"v1beta1", "v1"},
					FailurePolicy:           &failurePolicy,
					MatchPolicy:             &matchPolicy,
					SideEffects:             &sideEffect,
					TimeoutSeconds:          pointer.Int32(10),
				},
				{
					Name: "pod-zone-affinity.resources.gardener.cloud",
					Rules: []admissionregistrationv1.RuleWithOperations{{
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
						Operations: []admissionregistrationv1.OperationType{"CREATE"},
					}},
					NamespaceSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{{
							Key:      "control-plane.shoot.gardener.cloud/enforce-zone",
							Operator: metav1.LabelSelectorOpExists,
						}},
					},
					ObjectSelector: &metav1.LabelSelector{},
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Name:      "gardener-resource-manager",
							Namespace: deployNamespace,
							Path:      pointer.String("/webhooks/pod-zone-affinity"),
						},
					},
					AdmissionReviewVersions: []string{"v1beta1", "v1"},
					FailurePolicy:           &failurePolicy,
					MatchPolicy:             &matchPolicy,
					SideEffects:             &sideEffect,
					TimeoutSeconds:          pointer.Int32(10),
				},
			},
		}
		mutatingWebhookConfigurationYAML := `apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  creationTimestamp: null
  labels:
    app: gardener-resource-manager
  name: gardener-resource-manager-shoot
  namespace: fake-ns
webhooks:
- admissionReviewVersions:
  - v1beta1
  - v1
  clientConfig:
    url: https://gardener-resource-manager.` + deployNamespace + `:443/webhooks/invalidate-service-account-token-secret
  failurePolicy: Fail
  matchPolicy: Exact
  name: token-invalidator.resources.gardener.cloud
  namespaceSelector:
    matchExpressions:
    - key: gardener.cloud/purpose
      operator: In
      values:
      - kube-system
      - kubernetes-dashboard
  objectSelector:
    matchLabels:
      resources.gardener.cloud/purpose: token-invalidator
  rules:
  - apiGroups:
    - ""
    apiVersions:
    - v1
    operations:
    - CREATE
    - UPDATE
    resources:
    - secrets
  sideEffects: None
  timeoutSeconds: 10
- admissionReviewVersions:
  - v1beta1
  - v1
  clientConfig:
    url: https://gardener-resource-manager.` + deployNamespace + `:443/webhooks/mount-projected-service-account-token
  failurePolicy: Fail
  matchPolicy: Exact
  name: projected-token-mount.resources.gardener.cloud
  namespaceSelector:
    matchExpressions:
    - key: gardener.cloud/purpose
      operator: In
      values:
      - kube-system
      - kubernetes-dashboard
  objectSelector:
    matchExpressions:
    - key: projected-token-mount.resources.gardener.cloud/skip
      operator: DoesNotExist
    - key: app
      operator: NotIn
      values:
      - gardener-resource-manager
  rules:
  - apiGroups:
    - ""
    apiVersions:
    - v1
    operations:
    - CREATE
    resources:
    - pods
  sideEffects: None
  timeoutSeconds: 10
- admissionReviewVersions:
  - v1beta1
  - v1
  clientConfig:
    url: https://gardener-resource-manager.` + deployNamespace + `:443/webhooks/default-pod-scheduler-name
  failurePolicy: Ignore
  matchPolicy: Exact
  name: pod-scheduler-name.resources.gardener.cloud
  namespaceSelector: {}
  objectSelector: {}
  rules:
  - apiGroups:
    - ""
    apiVersions:
    - v1
    operations:
    - CREATE
    resources:
    - pods
  sideEffects: None
  timeoutSeconds: 10
`
		clusterRoleBindingTargetYAML := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  annotations:
    resources.gardener.cloud/keep-object: "true"
  creationTimestamp: null
  name: gardener.cloud:target:resource-manager
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: gardener-resource-manager
  namespace: kube-system
`

		managedResourceSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "managedresource-shoot-core-gardener-resource-manager",
				Namespace: deployNamespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"mutatingwebhookconfiguration__" + deployNamespace + "__gardener-resource-manager-shoot.yaml": []byte(mutatingWebhookConfigurationYAML),
				"clusterrolebinding____gardener.cloud_target_resource-manager.yaml":                           []byte(clusterRoleBindingTargetYAML),
			},
		}
		managedResource = &resourcesv1alpha1.ManagedResource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shoot-core-gardener-resource-manager",
				Namespace: deployNamespace,
				Labels:    map[string]string{"origin": "gardener"},
			},
			Spec: resourcesv1alpha1.ManagedResourceSpec{
				SecretRefs: []corev1.LocalObjectReference{
					{Name: "managedresource-shoot-core-gardener-resource-manager"},
				},
				InjectLabels: map[string]string{"shoot.gardener.cloud/no-cleanup": "true"},
				KeepObjects:  pointer.Bool(false),
			},
		}
		networkPolicy = &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "allow-kube-apiserver-to-gardener-resource-manager",
				Namespace: deployNamespace,
				Labels:    defaultLabels,
				Annotations: map[string]string{
					v1beta1constants.GardenerDescription: "Allows Egress from shoot's kube-apiserver pods to gardener-resource-manager pods.",
				},
			},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						v1beta1constants.LabelApp:   v1beta1constants.LabelKubernetes,
						v1beta1constants.LabelRole:  v1beta1constants.LabelAPIServer,
						v1beta1constants.GardenRole: v1beta1constants.GardenRoleControlPlane,
					},
				},
				Egress: []networkingv1.NetworkPolicyEgressRule{{
					To: []networkingv1.NetworkPolicyPeer{{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								v1beta1constants.LabelApp: "gardener-resource-manager",
							},
						},
					}},
					Ports: []networkingv1.NetworkPolicyPort{{
						Protocol: &networkPolicyProtocol,
						Port:     &networkPolicyPort,
					}},
				}},
				PolicyTypes: []networkingv1.PolicyType{
					networkingv1.PolicyTypeEgress,
				},
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#Deploy", func() {
		Context("target cluster != source cluster; watched namespace is set", func() {
			JustBeforeEach(func() {
				role.Namespace = watchedNamespace
				deployment = deploymentFor(cfg.Version, &watchedNamespace, pointer.String(gutil.PathGenericKubeconfig), true)
				resourceManager = New(c, deployNamespace, sm, image, cfg)
				resourceManager.SetSecrets(secrets)
			})

			Context("should successfully deploy all resources (w/ shoot access secret", func() {
				JustBeforeEach(func() {
					gomock.InOrder(
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, secret.Name), gomock.AssignableToTypeOf(&corev1.Secret{})),
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&corev1.Secret{}), gomock.Any()).
							Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(secret))
							}),
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})),
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&corev1.ServiceAccount{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(serviceAccount))
							}),
						c.EXPECT().Get(ctx, kutil.Key(watchedNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&rbacv1.Role{})),
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&rbacv1.Role{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(role))
							}),
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&rbacv1.RoleBinding{})),
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&rbacv1.RoleBinding{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(roleBinding))
							}),
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&corev1.Service{})),
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&corev1.Service{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(service))
							}),
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&appsv1.Deployment{})),
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&appsv1.Deployment{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(deployment))
							}),
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager-vpa"), gomock.AssignableToTypeOf(&vpaautoscalingv1.VerticalPodAutoscaler{})),
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&vpaautoscalingv1.VerticalPodAutoscaler{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(vpa))
							}),
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "managedresource-shoot-core-gardener-resource-manager"), gomock.AssignableToTypeOf(&corev1.Secret{})),
						c.EXPECT().Update(ctx, gomock.AssignableToTypeOf(&corev1.Secret{})).Do(func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) {
							Expect(obj).To(DeepEqual(managedResourceSecret))
						}),
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "shoot-core-gardener-resource-manager"), gomock.AssignableToTypeOf(&resourcesv1alpha1.ManagedResource{})),
						c.EXPECT().Update(ctx, gomock.AssignableToTypeOf(&resourcesv1alpha1.ManagedResource{})).Do(func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) {
							Expect(obj).To(DeepEqual(managedResource))
						}),
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "allow-kube-apiserver-to-gardener-resource-manager"), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(networkPolicy))
							}),
						c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager-server"}}),
					)
				})
				Context("Kubernetes version >= 1.21", func() {
					BeforeEach(func() {
						cfg.Version = semver.MustParse("1.24.0")

					})
					It("should successfully deploy all resources (w/ shoot access secret)", func() {
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, pdbV1.Name), gomock.AssignableToTypeOf(&policyv1.PodDisruptionBudget{}))
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&policyv1.PodDisruptionBudget{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(pdbV1))
							})

						Expect(resourceManager.Deploy(ctx)).To(Succeed())

					})
				})
				Context("Kubernetes version < 1.21", func() {
					BeforeEach(func() {
						cfg.Version = semver.MustParse("1.19.0")
					})
					It("should successfully deploy all resources (w/ shoot access secret)", func() {
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, pdbV1beta1.Name), gomock.AssignableToTypeOf(&policyv1beta1.PodDisruptionBudget{}))
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&policyv1beta1.PodDisruptionBudget{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(pdbV1beta1))
							})

						Expect(resourceManager.Deploy(ctx)).To(Succeed())

					})
				})
			})

			Context("should successfully deploy all resources (w/ bootstrap kubeconfig)", func() {
				JustBeforeEach(func() {
					secretNameBootstrapKubeconfig := "bootstrap-kubeconfig"

					secrets.BootstrapKubeconfig = &component.Secret{Name: secretNameBootstrapKubeconfig}
					resourceManager = New(c, deployNamespace, sm, image, cfg)
					resourceManager.SetSecrets(secrets)

					gomock.InOrder(
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, secret.Name), gomock.AssignableToTypeOf(&corev1.Secret{})),
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&corev1.Secret{}), gomock.Any()).
							Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(secret))
							}),
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})),
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&corev1.ServiceAccount{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(serviceAccount))
							}),
						c.EXPECT().Get(ctx, kutil.Key(watchedNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&rbacv1.Role{})),
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&rbacv1.Role{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(role))
							}),
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&rbacv1.RoleBinding{})),
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&rbacv1.RoleBinding{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(roleBinding))
							}),
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&corev1.Service{})),
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&corev1.Service{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(service))
							}),
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&appsv1.Deployment{})),
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&appsv1.Deployment{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								deployment.Spec.Template.Spec.Containers[0].VolumeMounts[len(deployment.Spec.Template.Spec.Containers[0].VolumeMounts)-1].Name = "kubeconfig-bootstrap"
								deployment.Spec.Template.Spec.Volumes[len(deployment.Spec.Template.Spec.Volumes)-1] = corev1.Volume{
									Name: "kubeconfig-bootstrap",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName:  secretNameBootstrapKubeconfig,
											DefaultMode: pointer.Int32(420),
										},
									},
								}

								Expect(obj).To(DeepEqual(deployment))
							}),
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager-vpa"), gomock.AssignableToTypeOf(&vpaautoscalingv1.VerticalPodAutoscaler{})),
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&vpaautoscalingv1.VerticalPodAutoscaler{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(vpa))
							}),
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "managedresource-shoot-core-gardener-resource-manager"), gomock.AssignableToTypeOf(&corev1.Secret{})),
						c.EXPECT().Update(ctx, gomock.AssignableToTypeOf(&corev1.Secret{})).Do(func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) {
							Expect(obj).To(DeepEqual(managedResourceSecret))
						}),
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "shoot-core-gardener-resource-manager"), gomock.AssignableToTypeOf(&resourcesv1alpha1.ManagedResource{})),
						c.EXPECT().Update(ctx, gomock.AssignableToTypeOf(&resourcesv1alpha1.ManagedResource{})).Do(func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) {
							Expect(obj).To(DeepEqual(managedResource))
						}),
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "allow-kube-apiserver-to-gardener-resource-manager"), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(networkPolicy))
							}),
						c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager-server"}}),
					)
				})
				Context("Kubernetes version >= 1.21", func() {
					BeforeEach(func() {
						cfg.Version = semver.MustParse("1.24.0")
					})

					It("should successfully deploy all resources (w/ bootstrap kubeconfig)", func() {
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, pdbV1.Name), gomock.AssignableToTypeOf(&policyv1.PodDisruptionBudget{}))
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&policyv1.PodDisruptionBudget{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(pdbV1))
							})

						Expect(resourceManager.Deploy(ctx)).To(Succeed())
					})
				})
				Context("Kubernetes version < 1.21", func() {
					BeforeEach(func() {
						cfg.Version = semver.MustParse("1.19.0")
					})

					It("should successfully deploy all resources (w/ bootstrap kubeconfig)", func() {
						c.EXPECT().Get(ctx, kutil.Key(deployNamespace, pdbV1beta1.Name), gomock.AssignableToTypeOf(&policyv1beta1.PodDisruptionBudget{}))
						c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&policyv1beta1.PodDisruptionBudget{}), gomock.Any()).
							Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
								Expect(obj).To(DeepEqual(pdbV1beta1))
							})

						Expect(resourceManager.Deploy(ctx)).To(Succeed())
					})
				})
			})
		})

		Context("target cluster != source cluster, watched namespace is nil", func() {
			BeforeEach(func() {
				clusterRole.Rules = allowManagedResources
				cfg.TargetDiffersFromSourceCluster = true
				cfg.WatchedNamespace = nil
				deployment = deploymentFor(cfg.Version, nil, pointer.String(gutil.PathGenericKubeconfig), true)

				resourceManager = New(c, deployNamespace, sm, image, cfg)
				resourceManager.SetSecrets(secrets)
			})

			It("should deploy a ClusterRole allowing access to mr related resources", func() {
				gomock.InOrder(
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, secret.Name), gomock.AssignableToTypeOf(&corev1.Secret{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&corev1.Secret{}), gomock.Any()).
						Do(func(ctx context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) {
							Expect(obj).To(DeepEqual(secret))
						}),
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&corev1.ServiceAccount{}), gomock.Any()).
						Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
							Expect(obj).To(DeepEqual(serviceAccount))
						}),
					c.EXPECT().Get(ctx, kutil.Key(clusterRoleName), gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&rbacv1.ClusterRole{}), gomock.Any()).
						Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
							Expect(obj).To(DeepEqual(clusterRole))
						}),
					c.EXPECT().Get(ctx, kutil.Key(clusterRoleName), gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{}), gomock.Any()).
						Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
							Expect(obj).To(DeepEqual(clusterRoleBinding))
						}),
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&corev1.Service{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&corev1.Service{}), gomock.Any()).
						Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
							Expect(obj).To(DeepEqual(service))
						}),
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&appsv1.Deployment{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&appsv1.Deployment{}), gomock.Any()).
						Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
							Expect(obj).To(DeepEqual(deployment))
						}),
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, pdbV1.Name), gomock.AssignableToTypeOf(&policyv1.PodDisruptionBudget{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&policyv1.PodDisruptionBudget{}), gomock.Any()).
						Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
							Expect(obj).To(DeepEqual(pdbV1))
						}),
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager-vpa"), gomock.AssignableToTypeOf(&vpaautoscalingv1.VerticalPodAutoscaler{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&vpaautoscalingv1.VerticalPodAutoscaler{}), gomock.Any()).
						Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
							Expect(obj).To(DeepEqual(vpa))
						}),
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "managedresource-shoot-core-gardener-resource-manager"), gomock.AssignableToTypeOf(&corev1.Secret{})),
					c.EXPECT().Update(ctx, gomock.AssignableToTypeOf(&corev1.Secret{})).Do(func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) {
						Expect(obj).To(DeepEqual(managedResourceSecret))
					}),
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "shoot-core-gardener-resource-manager"), gomock.AssignableToTypeOf(&resourcesv1alpha1.ManagedResource{})),
					c.EXPECT().Update(ctx, gomock.AssignableToTypeOf(&resourcesv1alpha1.ManagedResource{})).Do(func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) {
						Expect(obj).To(DeepEqual(managedResource))
					}),
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "allow-kube-apiserver-to-gardener-resource-manager"), gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&networkingv1.NetworkPolicy{}), gomock.Any()).
						Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
							Expect(obj).To(DeepEqual(networkPolicy))
						}),
					c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager-server"}}),
				)
				Expect(resourceManager.Deploy(ctx)).To(Succeed())
			})

			It("should fail because the ClusterRole can not be created", func() {
				gomock.InOrder(
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, secret.Name), gomock.AssignableToTypeOf(&corev1.Secret{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&corev1.Secret{}), gomock.Any()),
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&corev1.ServiceAccount{}), gomock.Any()),
					c.EXPECT().Get(ctx, kutil.Key(clusterRoleName), gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&rbacv1.ClusterRole{}), gomock.Any()).Return(fakeErr),
				)

				Expect(resourceManager.Deploy(ctx)).To(MatchError(fakeErr))
			})

			It("should fail because the ClusterRoleBinding can not be created", func() {
				gomock.InOrder(
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, secret.Name), gomock.AssignableToTypeOf(&corev1.Secret{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&corev1.Secret{}), gomock.Any()),
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&corev1.ServiceAccount{}), gomock.Any()),
					c.EXPECT().Get(ctx, kutil.Key(clusterRoleName), gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&rbacv1.ClusterRole{}), gomock.Any()),
					c.EXPECT().Get(ctx, kutil.Key(clusterRoleName), gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{}), gomock.Any()).Return(fakeErr),
				)

				Expect(resourceManager.Deploy(ctx)).To(MatchError(fakeErr))
			})
		})

		Context("target cluster = source cluster", func() {
			BeforeEach(func() {
				clusterRole.Rules = allowAll
				deployment = deploymentFor(cfg.Version, &watchedNamespace, nil, false)

				for i, arg := range deployment.Spec.Template.Spec.Containers[0].Args {
					if strings.HasPrefix(arg, "--root-ca-file=") {
						deployment.Spec.Template.Spec.Containers[0].Args[i] = "--root-ca-file=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
					}
				}
				deployment.Spec.Template.Spec.Volumes = deployment.Spec.Template.Spec.Volumes[:len(deployment.Spec.Template.Spec.Volumes)-2]
				deployment.Spec.Template.Spec.Containers[0].VolumeMounts = deployment.Spec.Template.Spec.Containers[0].VolumeMounts[:len(deployment.Spec.Template.Spec.Containers[0].VolumeMounts)-2]
				deployment.Spec.Template.Labels["gardener.cloud/role"] = "seed"
				deployment.Spec.Template.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.LabelSelector.MatchLabels["gardener.cloud/role"] = "seed"
				pdbV1.Spec.Selector.MatchLabels["gardener.cloud/role"] = "seed"

				// Remove controlplane label from resources
				delete(serviceAccount.ObjectMeta.Labels, v1beta1constants.GardenRole)
				delete(clusterRole.ObjectMeta.Labels, v1beta1constants.GardenRole)
				delete(clusterRoleBinding.ObjectMeta.Labels, v1beta1constants.GardenRole)
				delete(service.ObjectMeta.Labels, v1beta1constants.GardenRole)
				delete(deployment.ObjectMeta.Labels, v1beta1constants.GardenRole)
				delete(vpa.ObjectMeta.Labels, v1beta1constants.GardenRole)
				delete(pdbV1.ObjectMeta.Labels, v1beta1constants.GardenRole)
				// Remove networking label from deployment template
				delete(deployment.Spec.Template.Labels, "networking.gardener.cloud/to-dns")
				delete(deployment.Spec.Template.Labels, "networking.gardener.cloud/to-seed-apiserver")
				delete(deployment.Spec.Template.Labels, "networking.gardener.cloud/to-shoot-apiserver")
				delete(deployment.Spec.Template.Labels, "networking.gardener.cloud/from-shoot-apiserver")

				cfg.TargetDiffersFromSourceCluster = false
				resourceManager = New(c, deployNamespace, sm, image, cfg)
				resourceManager.SetSecrets(secrets)
			})

			It("should deploy a cluster role allowing all access", func() {
				gomock.InOrder(
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&corev1.ServiceAccount{}), gomock.Any()).
						Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
							Expect(obj).To(DeepEqual(serviceAccount))
						}),
					c.EXPECT().Get(ctx, kutil.Key(clusterRoleName), gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&rbacv1.ClusterRole{}), gomock.Any()).
						Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
							Expect(obj).To(DeepEqual(clusterRole))
						}),
					c.EXPECT().Get(ctx, kutil.Key(clusterRoleName), gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{}), gomock.Any()).
						Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
							Expect(obj).To(DeepEqual(clusterRoleBinding))
						}),
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&corev1.Service{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&corev1.Service{}), gomock.Any()).
						Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
							Expect(obj).To(DeepEqual(service))
						}),
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&appsv1.Deployment{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&appsv1.Deployment{}), gomock.Any()).
						Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
							Expect(obj).To(DeepEqual(deployment))
						}),
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, pdbV1.Name), gomock.AssignableToTypeOf(&policyv1.PodDisruptionBudget{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&policyv1.PodDisruptionBudget{}), gomock.Any()).
						Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
							Expect(obj).To(DeepEqual(pdbV1))
						}),
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager-vpa"), gomock.AssignableToTypeOf(&vpaautoscalingv1.VerticalPodAutoscaler{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&vpaautoscalingv1.VerticalPodAutoscaler{}), gomock.Any()).
						Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
							Expect(obj).To(DeepEqual(vpa))
						}),
					c.EXPECT().Get(ctx, kutil.Key(deployNamespace, "gardener-resource-manager"), gomock.AssignableToTypeOf(&admissionregistrationv1.MutatingWebhookConfiguration{})),
					c.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&admissionregistrationv1.MutatingWebhookConfiguration{}), gomock.Any()).
						Do(func(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) {
							Expect(obj).To(DeepEqual(mutatingWebhookConfiguration))
						}),
					c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager-server"}}),
				)
				Expect(resourceManager.Deploy(ctx)).To(Succeed())
			})
		})
	})

	Describe("#Destroy", func() {
		Context("target differs from source cluster", func() {
			JustBeforeEach(func() {
				deployment = deploymentFor(cfg.Version, &watchedNamespace, pointer.String(gutil.PathGenericKubeconfig), true)
				resourceManager = New(c, deployNamespace, sm, image, cfg)
			})

			Context("should delete all created resources", func() {
				JustBeforeEach(func() {
					gomock.InOrder(
						c.EXPECT().Delete(ctx, &resourcesv1alpha1.ManagedResource{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}}),
						c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "managedresource-shoot-core-gardener-resource-manager"}}),
						c.EXPECT().Get(gomock.Any(), client.ObjectKey{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}, gomock.AssignableToTypeOf(&resourcesv1alpha1.ManagedResource{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
						c.EXPECT().Delete(ctx, &vpaautoscalingv1.VerticalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager-vpa"}}),
						c.EXPECT().Delete(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
						c.EXPECT().Delete(ctx, &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
						c.EXPECT().Delete(ctx, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
						c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: secret.Name}}),
						c.EXPECT().Delete(ctx, &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
						c.EXPECT().Delete(ctx, &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					)
				})

				Context("Kubernetes version >= v1.21", func() {
					BeforeEach(func() {
						cfg.Version = semver.MustParse("1.22")
					})
					It("should delete all created resources", func() {
						c.EXPECT().Delete(ctx, &policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}})

						Expect(resourceManager.Destroy(ctx)).To(Succeed())
					})
				})

				Context("Kubernetes version < v1.21", func() {
					BeforeEach(func() {
						cfg.Version = semver.MustParse("1.20")
					})
					It("should delete all created resources", func() {
						c.EXPECT().Delete(ctx, &policyv1beta1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}})

						Expect(resourceManager.Destroy(ctx)).To(Succeed())
					})
				})
			})

			It("should fail because the managed resource cannot be deleted", func() {
				gomock.InOrder(
					c.EXPECT().Delete(ctx, &resourcesv1alpha1.ManagedResource{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}}).Return(fakeErr),
				)

				Expect(resourceManager.Destroy(ctx)).To(MatchError(fakeErr))
			})

			It("should fail because the managed resource secret cannot be deleted", func() {
				gomock.InOrder(
					c.EXPECT().Delete(ctx, &resourcesv1alpha1.ManagedResource{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "managedresource-shoot-core-gardener-resource-manager"}}).Return(fakeErr),
				)

				Expect(resourceManager.Destroy(ctx)).To(MatchError(fakeErr))
			})

			It("should fail because the pdb cannot be deleted", func() {
				gomock.InOrder(
					c.EXPECT().Delete(ctx, &resourcesv1alpha1.ManagedResource{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "managedresource-shoot-core-gardener-resource-manager"}}),
					c.EXPECT().Get(gomock.Any(), client.ObjectKey{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}, gomock.AssignableToTypeOf(&resourcesv1alpha1.ManagedResource{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
					c.EXPECT().Delete(ctx, &policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}).Return(fakeErr),
				)

				Expect(resourceManager.Destroy(ctx)).To(MatchError(fakeErr))
			})

			It("should fail because the vpa cannot be deleted", func() {
				gomock.InOrder(
					c.EXPECT().Delete(ctx, &resourcesv1alpha1.ManagedResource{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "managedresource-shoot-core-gardener-resource-manager"}}),
					c.EXPECT().Get(gomock.Any(), client.ObjectKey{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}, gomock.AssignableToTypeOf(&resourcesv1alpha1.ManagedResource{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
					c.EXPECT().Delete(ctx, &policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &vpaautoscalingv1.VerticalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager-vpa"}}).Return(fakeErr),
				)

				Expect(resourceManager.Destroy(ctx)).To(MatchError(fakeErr))
			})

			It("should fail because the deployment cannot be deleted", func() {
				gomock.InOrder(
					c.EXPECT().Delete(ctx, &resourcesv1alpha1.ManagedResource{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "managedresource-shoot-core-gardener-resource-manager"}}),
					c.EXPECT().Get(gomock.Any(), client.ObjectKey{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}, gomock.AssignableToTypeOf(&resourcesv1alpha1.ManagedResource{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
					c.EXPECT().Delete(ctx, &policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &vpaautoscalingv1.VerticalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager-vpa"}}),
					c.EXPECT().Delete(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}).Return(fakeErr),
				)

				Expect(resourceManager.Destroy(ctx)).To(MatchError(fakeErr))
			})

			It("should fail because the service cannot be deleted", func() {
				gomock.InOrder(
					c.EXPECT().Delete(ctx, &resourcesv1alpha1.ManagedResource{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "managedresource-shoot-core-gardener-resource-manager"}}),
					c.EXPECT().Get(gomock.Any(), client.ObjectKey{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}, gomock.AssignableToTypeOf(&resourcesv1alpha1.ManagedResource{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
					c.EXPECT().Delete(ctx, &policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &vpaautoscalingv1.VerticalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager-vpa"}}),
					c.EXPECT().Delete(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}).Return(fakeErr),
				)

				Expect(resourceManager.Destroy(ctx)).To(MatchError(fakeErr))
			})

			It("should fail because the service account cannot be deleted", func() {
				gomock.InOrder(
					c.EXPECT().Delete(ctx, &resourcesv1alpha1.ManagedResource{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "managedresource-shoot-core-gardener-resource-manager"}}),
					c.EXPECT().Get(gomock.Any(), client.ObjectKey{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}, gomock.AssignableToTypeOf(&resourcesv1alpha1.ManagedResource{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
					c.EXPECT().Delete(ctx, &policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &vpaautoscalingv1.VerticalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager-vpa"}}),
					c.EXPECT().Delete(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}).Return(fakeErr),
				)

				Expect(resourceManager.Destroy(ctx)).To(MatchError(fakeErr))
			})

			It("should fail because the secret cannot be deleted", func() {
				gomock.InOrder(
					c.EXPECT().Delete(ctx, &resourcesv1alpha1.ManagedResource{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "managedresource-shoot-core-gardener-resource-manager"}}),
					c.EXPECT().Get(gomock.Any(), client.ObjectKey{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}, gomock.AssignableToTypeOf(&resourcesv1alpha1.ManagedResource{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
					c.EXPECT().Delete(ctx, &policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &vpaautoscalingv1.VerticalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager-vpa"}}),
					c.EXPECT().Delete(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: secret.Name}}).Return(fakeErr),
				)

				Expect(resourceManager.Destroy(ctx)).To(MatchError(fakeErr))
			})

			It("should fail because the role cannot be deleted", func() {
				gomock.InOrder(
					c.EXPECT().Delete(ctx, &resourcesv1alpha1.ManagedResource{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "managedresource-shoot-core-gardener-resource-manager"}}),
					c.EXPECT().Get(gomock.Any(), client.ObjectKey{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}, gomock.AssignableToTypeOf(&resourcesv1alpha1.ManagedResource{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
					c.EXPECT().Delete(ctx, &policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &vpaautoscalingv1.VerticalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager-vpa"}}),
					c.EXPECT().Delete(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: secret.Name}}),
					c.EXPECT().Delete(ctx, &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}).Return(fakeErr),
				)

				Expect(resourceManager.Destroy(ctx)).To(MatchError(fakeErr))
			})

			It("should fail because the role binding cannot be deleted", func() {
				gomock.InOrder(
					c.EXPECT().Delete(ctx, &resourcesv1alpha1.ManagedResource{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "managedresource-shoot-core-gardener-resource-manager"}}),
					c.EXPECT().Get(gomock.Any(), client.ObjectKey{Namespace: deployNamespace, Name: "shoot-core-gardener-resource-manager"}, gomock.AssignableToTypeOf(&resourcesv1alpha1.ManagedResource{})).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
					c.EXPECT().Delete(ctx, &policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &vpaautoscalingv1.VerticalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager-vpa"}}),
					c.EXPECT().Delete(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: secret.Name}}),
					c.EXPECT().Delete(ctx, &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}).Return(fakeErr),
				)

				Expect(resourceManager.Destroy(ctx)).To(MatchError(fakeErr))
			})
		})

		Context("target equals source cluster", func() {
			BeforeEach(func() {
				cfg.TargetDiffersFromSourceCluster = false
				cfg.WatchedNamespace = nil
				deployment = deploymentFor(cfg.Version, nil, pointer.String(gutil.PathGenericKubeconfig), false)
				resourceManager = New(c, deployNamespace, sm, image, cfg)
			})

			It("should delete all created resources", func() {
				gomock.InOrder(
					c.EXPECT().Delete(ctx, &policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &vpaautoscalingv1.VerticalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager-vpa"}}),
					c.EXPECT().Delete(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &admissionregistrationv1.MutatingWebhookConfiguration{ObjectMeta: metav1.ObjectMeta{Namespace: deployNamespace, Name: "gardener-resource-manager"}}),
					c.EXPECT().Delete(ctx, &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: clusterRoleName}}),
					c.EXPECT().Delete(ctx, &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: clusterRoleName}}),
				)

				Expect(resourceManager.Destroy(ctx)).To(Succeed())
			})
		})
	})

	Describe("#Wait", func() {
		BeforeEach(func() {
			deployment = deploymentFor(cfg.Version, &watchedNamespace, pointer.String(gutil.PathGenericKubeconfig), false)
			resourceManager = New(fakeClient, deployNamespace, nil, image, cfg)
		})

		It("should successfully wait for the deployment to be ready", func() {
			defer test.WithVars(&IntervalWaitForDeployment, time.Millisecond)()
			defer test.WithVars(&TimeoutWaitForDeployment, 100*time.Millisecond)()

			Expect(fakeClient.Create(ctx, deployment)).To(Succeed())
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)).To(Succeed())

			timer := time.AfterFunc(10*time.Millisecond, func() {
				deployment.Status.Conditions = []appsv1.DeploymentCondition{
					{
						Type:   appsv1.DeploymentAvailable,
						Status: corev1.ConditionTrue,
					},
				}
				Expect(fakeClient.Status().Update(ctx, deployment)).To(Succeed())
			})
			defer timer.Stop()

			Expect(resourceManager.Wait(ctx)).To(Succeed())
		})

		It("should fail while waiting for the deployment to be ready", func() {
			defer test.WithVars(&IntervalWaitForDeployment, time.Millisecond)()
			defer test.WithVars(&TimeoutWaitForDeployment, 10*time.Millisecond)()

			deployment.Status.Conditions = []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentAvailable,
					Status: corev1.ConditionFalse,
				},
			}

			Expect(fakeClient.Create(ctx, deployment)).To(Succeed())
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)).To(Succeed())

			Expect(resourceManager.Wait(ctx)).To(MatchError(ContainSubstring(`condition "Available" has invalid status False (expected True)`)))
		})
	})

	Describe("#WaitCleanup", func() {
		It("should return nil as it's not implemented as of now", func() {
			Expect(resourceManager.WaitCleanup(ctx)).To(Succeed())
		})
	})

	Describe("#SetReplicas, #GetReplicas", func() {
		It("should set and return the replicas", func() {
			resourceManager = New(nil, "", nil, "", Values{})
			Expect(resourceManager.GetReplicas()).To(BeZero())

			resourceManager.SetReplicas(&replicas)
			Expect(resourceManager.GetReplicas()).To(PointTo(Equal(replicas)))
		})
	})
})
