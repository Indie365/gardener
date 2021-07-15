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

package kubeapiserver_test

import (
	"context"
	"time"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	. "github.com/gardener/gardener/pkg/operation/botanist/component/kubeapiserver"
	"github.com/gardener/gardener/pkg/utils/test"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"

	hvpav1alpha1 "github.com/gardener/hvpa-controller/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2beta1 "k8s.io/api/autoscaling/v2beta1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("KubeAPIServer", func() {
	var (
		ctx = context.TODO()

		namespace          = "some-namespace"
		vpaUpdateMode      = autoscalingv1beta2.UpdateModeOff
		containerPolicyOff = autoscalingv1beta2.ContainerScalingModeOff

		c    client.Client
		kapi Interface

		deployment              *appsv1.Deployment
		horizontalPodAutoscaler *autoscalingv2beta1.HorizontalPodAutoscaler
		verticalPodAutoscaler   *autoscalingv1beta2.VerticalPodAutoscaler
		hvpa                    *hvpav1alpha1.Hvpa
		podDisruptionBudget     *policyv1beta1.PodDisruptionBudget
	)

	BeforeEach(func() {
		c = fakeclient.NewClientBuilder().WithScheme(kubernetes.SeedScheme).Build()
		kapi = New(c, namespace, Values{})

		deployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-apiserver",
				Namespace: namespace,
			},
		}
		horizontalPodAutoscaler = &autoscalingv2beta1.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-apiserver",
				Namespace: namespace,
			},
		}
		verticalPodAutoscaler = &autoscalingv1beta2.VerticalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-apiserver-vpa",
				Namespace: namespace,
			},
		}
		hvpa = &hvpav1alpha1.Hvpa{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-apiserver",
				Namespace: namespace,
			},
		}
		podDisruptionBudget = &policyv1beta1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-apiserver",
				Namespace: namespace,
			},
		}
	})

	Describe("#Deploy", func() {
		Describe("HorizontalPodAutoscaler", func() {
			DescribeTable("should delete the HPA resource",
				func(autoscalingConfig AutoscalingConfig) {
					kapi = New(c, namespace, Values{Autoscaling: autoscalingConfig})

					Expect(c.Create(ctx, horizontalPodAutoscaler)).To(Succeed())
					Expect(c.Get(ctx, client.ObjectKeyFromObject(horizontalPodAutoscaler), horizontalPodAutoscaler)).To(Succeed())
					Expect(kapi.Deploy(ctx)).To(Succeed())
					Expect(c.Get(ctx, client.ObjectKeyFromObject(horizontalPodAutoscaler), horizontalPodAutoscaler)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: autoscalingv2beta1.SchemeGroupVersion.Group, Resource: "horizontalpodautoscalers"}, horizontalPodAutoscaler.Name)))
				},

				Entry("HVPA is enabled", AutoscalingConfig{HVPAEnabled: true}),
				Entry("replicas is nil", AutoscalingConfig{HVPAEnabled: false, Replicas: nil}),
				Entry("replicas is 0", AutoscalingConfig{HVPAEnabled: false, Replicas: pointer.Int32(0)}),
			)

			It("should successfully deploy the HPA resource", func() {
				autoscalingConfig := AutoscalingConfig{
					HVPAEnabled: false,
					Replicas:    pointer.Int32(2),
					MinReplicas: 4,
					MaxReplicas: 6,
				}
				kapi = New(c, namespace, Values{Autoscaling: autoscalingConfig})

				Expect(c.Get(ctx, client.ObjectKeyFromObject(horizontalPodAutoscaler), horizontalPodAutoscaler)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: autoscalingv2beta1.SchemeGroupVersion.Group, Resource: "horizontalpodautoscalers"}, horizontalPodAutoscaler.Name)))
				Expect(kapi.Deploy(ctx)).To(Succeed())
				Expect(c.Get(ctx, client.ObjectKeyFromObject(horizontalPodAutoscaler), horizontalPodAutoscaler)).To(Succeed())
				Expect(horizontalPodAutoscaler).To(DeepEqual(&autoscalingv2beta1.HorizontalPodAutoscaler{
					TypeMeta: metav1.TypeMeta{
						APIVersion: autoscalingv2beta1.SchemeGroupVersion.String(),
						Kind:       "HorizontalPodAutoscaler",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            horizontalPodAutoscaler.Name,
						Namespace:       horizontalPodAutoscaler.Namespace,
						ResourceVersion: "1",
					},
					Spec: autoscalingv2beta1.HorizontalPodAutoscalerSpec{
						MinReplicas: &autoscalingConfig.MinReplicas,
						MaxReplicas: autoscalingConfig.MaxReplicas,
						ScaleTargetRef: autoscalingv2beta1.CrossVersionObjectReference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "kube-apiserver",
						},
						Metrics: []autoscalingv2beta1.MetricSpec{
							{
								Type: "Resource",
								Resource: &autoscalingv2beta1.ResourceMetricSource{
									Name:                     "cpu",
									TargetAverageUtilization: pointer.Int32(80),
								},
							},
							{
								Type: "Resource",
								Resource: &autoscalingv2beta1.ResourceMetricSource{
									Name:                     "memory",
									TargetAverageUtilization: pointer.Int32(80),
								},
							},
						},
					},
				}))
			})
		})

		Describe("VerticalPodAutoscaler", func() {
			It("should delete the VPA resource", func() {
				kapi = New(c, namespace, Values{Autoscaling: AutoscalingConfig{HVPAEnabled: true}})

				Expect(c.Create(ctx, verticalPodAutoscaler)).To(Succeed())
				Expect(c.Get(ctx, client.ObjectKeyFromObject(verticalPodAutoscaler), verticalPodAutoscaler)).To(Succeed())
				Expect(kapi.Deploy(ctx)).To(Succeed())
				Expect(c.Get(ctx, client.ObjectKeyFromObject(verticalPodAutoscaler), verticalPodAutoscaler)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: autoscalingv1beta2.SchemeGroupVersion.Group, Resource: "verticalpodautoscalers"}, verticalPodAutoscaler.Name)))
			})

			It("should successfully deploy the VPA resource", func() {
				autoscalingConfig := AutoscalingConfig{HVPAEnabled: false}
				kapi = New(c, namespace, Values{Autoscaling: autoscalingConfig})

				Expect(c.Get(ctx, client.ObjectKeyFromObject(verticalPodAutoscaler), verticalPodAutoscaler)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: autoscalingv1beta2.SchemeGroupVersion.Group, Resource: "verticalpodautoscalers"}, verticalPodAutoscaler.Name)))
				Expect(kapi.Deploy(ctx)).To(Succeed())
				Expect(c.Get(ctx, client.ObjectKeyFromObject(verticalPodAutoscaler), verticalPodAutoscaler)).To(Succeed())
				Expect(verticalPodAutoscaler).To(DeepEqual(&autoscalingv1beta2.VerticalPodAutoscaler{
					TypeMeta: metav1.TypeMeta{
						APIVersion: autoscalingv1beta2.SchemeGroupVersion.String(),
						Kind:       "VerticalPodAutoscaler",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            verticalPodAutoscaler.Name,
						Namespace:       verticalPodAutoscaler.Namespace,
						ResourceVersion: "1",
					},
					Spec: autoscalingv1beta2.VerticalPodAutoscalerSpec{
						TargetRef: &autoscalingv1.CrossVersionObjectReference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "kube-apiserver",
						},
						UpdatePolicy: &autoscalingv1beta2.PodUpdatePolicy{
							UpdateMode: &vpaUpdateMode,
						},
					},
				}))
			})
		})

		Describe("HVPA", func() {
			DescribeTable("should delete the HVPA resource",
				func(autoscalingConfig AutoscalingConfig) {
					kapi = New(c, namespace, Values{Autoscaling: autoscalingConfig})

					Expect(c.Create(ctx, hvpa)).To(Succeed())
					Expect(c.Get(ctx, client.ObjectKeyFromObject(hvpa), hvpa)).To(Succeed())
					Expect(kapi.Deploy(ctx)).To(Succeed())
					Expect(c.Get(ctx, client.ObjectKeyFromObject(hvpa), hvpa)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: hvpav1alpha1.SchemeGroupVersionHvpa.Group, Resource: "hvpas"}, hvpa.Name)))
				},

				Entry("HVPA disabled", AutoscalingConfig{HVPAEnabled: false}),
				Entry("HVPA enabled but replicas nil", AutoscalingConfig{HVPAEnabled: true}),
				Entry("HVPA enabled but replicas zero", AutoscalingConfig{HVPAEnabled: true, Replicas: pointer.Int32(0)}),
			)

			var (
				defaultExpectedScaleDownUpdateMode = "Auto"
				defaultExpectedHPAMetrics          = []autoscalingv2beta1.MetricSpec{
					{
						Type: "Resource",
						Resource: &autoscalingv2beta1.ResourceMetricSource{
							Name:                     "cpu",
							TargetAverageUtilization: pointer.Int32(80),
						},
					},
				}
				defaultExpectedVPAContainerResourcePolicies = []autoscalingv1beta2.ContainerResourcePolicy{
					{
						ContainerName: "kube-apiserver",
						MinAllowed: corev1.ResourceList{
							"cpu":    resource.MustParse("300m"),
							"memory": resource.MustParse("400M"),
						},
						MaxAllowed: corev1.ResourceList{
							"cpu":    resource.MustParse("8"),
							"memory": resource.MustParse("25G"),
						},
					},
					{
						ContainerName: "vpn-seed",
						Mode:          &containerPolicyOff,
					},
				}
				defaultExpectedWeightBasedScalingIntervals = []hvpav1alpha1.WeightBasedScalingInterval{
					{
						VpaWeight:         100,
						StartReplicaCount: 5,
						LastReplicaCount:  5,
					},
				}
			)

			DescribeTable("should successfully deploy the HVPA resource",
				func(
					autoscalingConfig AutoscalingConfig,
					sniConfig SNIConfig,
					expectedScaleDownUpdateMode string,
					expectedHPAMetrics []autoscalingv2beta1.MetricSpec,
					expectedVPAContainerResourcePolicies []autoscalingv1beta2.ContainerResourcePolicy,
					expectedWeightBasedScalingIntervals []hvpav1alpha1.WeightBasedScalingInterval,
				) {
					kapi = New(c, namespace, Values{
						Autoscaling: autoscalingConfig,
						SNI:         sniConfig,
					})

					Expect(c.Get(ctx, client.ObjectKeyFromObject(hvpa), hvpa)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: hvpav1alpha1.SchemeGroupVersionHvpa.Group, Resource: "hvpas"}, hvpa.Name)))
					Expect(kapi.Deploy(ctx)).To(Succeed())
					Expect(c.Get(ctx, client.ObjectKeyFromObject(hvpa), hvpa)).To(Succeed())
					Expect(hvpa).To(DeepEqual(&hvpav1alpha1.Hvpa{
						TypeMeta: metav1.TypeMeta{
							APIVersion: hvpav1alpha1.SchemeGroupVersionHvpa.String(),
							Kind:       "Hvpa",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:            hvpa.Name,
							Namespace:       hvpa.Namespace,
							ResourceVersion: "1",
						},
						Spec: hvpav1alpha1.HvpaSpec{
							Replicas: pointer.Int32(1),
							Hpa: hvpav1alpha1.HpaSpec{
								Selector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"role": "apiserver-hpa"},
								},
								Deploy: true,
								ScaleUp: hvpav1alpha1.ScaleType{
									UpdatePolicy: hvpav1alpha1.UpdatePolicy{
										UpdateMode: pointer.StringPtr("Auto"),
									},
								},
								ScaleDown: hvpav1alpha1.ScaleType{
									UpdatePolicy: hvpav1alpha1.UpdatePolicy{
										UpdateMode: &expectedScaleDownUpdateMode,
									},
								},
								Template: hvpav1alpha1.HpaTemplate{
									ObjectMeta: metav1.ObjectMeta{
										Labels: map[string]string{"role": "apiserver-hpa"},
									},
									Spec: hvpav1alpha1.HpaTemplateSpec{
										MinReplicas: &autoscalingConfig.MinReplicas,
										MaxReplicas: autoscalingConfig.MaxReplicas,
										Metrics:     expectedHPAMetrics,
									},
								},
							},
							Vpa: hvpav1alpha1.VpaSpec{
								Selector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"role": "apiserver-vpa"},
								},
								Deploy: true,
								ScaleUp: hvpav1alpha1.ScaleType{
									UpdatePolicy: hvpav1alpha1.UpdatePolicy{
										UpdateMode: pointer.StringPtr("Auto"),
									},
									StabilizationDuration: pointer.StringPtr("3m"),
									MinChange: hvpav1alpha1.ScaleParams{
										CPU: hvpav1alpha1.ChangeParams{
											Value:      pointer.StringPtr("300m"),
											Percentage: pointer.Int32Ptr(80),
										},
										Memory: hvpav1alpha1.ChangeParams{
											Value:      pointer.StringPtr("200M"),
											Percentage: pointer.Int32Ptr(80),
										},
									},
								},
								ScaleDown: hvpav1alpha1.ScaleType{
									UpdatePolicy: hvpav1alpha1.UpdatePolicy{
										UpdateMode: &expectedScaleDownUpdateMode,
									},
									StabilizationDuration: pointer.StringPtr("15m"),
									MinChange: hvpav1alpha1.ScaleParams{
										CPU: hvpav1alpha1.ChangeParams{
											Value:      pointer.StringPtr("300m"),
											Percentage: pointer.Int32Ptr(80),
										},
										Memory: hvpav1alpha1.ChangeParams{
											Value:      pointer.StringPtr("200M"),
											Percentage: pointer.Int32Ptr(80),
										},
									},
								},
								LimitsRequestsGapScaleParams: hvpav1alpha1.ScaleParams{
									CPU: hvpav1alpha1.ChangeParams{
										Value:      pointer.StringPtr("1"),
										Percentage: pointer.Int32Ptr(70),
									},
									Memory: hvpav1alpha1.ChangeParams{
										Value:      pointer.StringPtr("1G"),
										Percentage: pointer.Int32Ptr(70),
									},
								},
								Template: hvpav1alpha1.VpaTemplate{
									ObjectMeta: metav1.ObjectMeta{
										Labels: map[string]string{"role": "apiserver-vpa"},
									},
									Spec: hvpav1alpha1.VpaTemplateSpec{
										ResourcePolicy: &autoscalingv1beta2.PodResourcePolicy{
											ContainerPolicies: expectedVPAContainerResourcePolicies,
										},
									},
								},
							},
							WeightBasedScalingIntervals: expectedWeightBasedScalingIntervals,
							TargetRef: &autoscalingv2beta1.CrossVersionObjectReference{
								APIVersion: "apps/v1",
								Kind:       "Deployment",
								Name:       "kube-apiserver",
							},
						},
					}))
				},

				Entry("default behaviour",
					AutoscalingConfig{
						HVPAEnabled: true,
						Replicas:    pointer.Int32(2),
						MinReplicas: 5,
						MaxReplicas: 5,
					},
					SNIConfig{},
					defaultExpectedScaleDownUpdateMode,
					defaultExpectedHPAMetrics,
					defaultExpectedVPAContainerResourcePolicies,
					defaultExpectedWeightBasedScalingIntervals,
				),
				Entry("UseMemoryMetricForHvpaHPA is true",
					AutoscalingConfig{
						HVPAEnabled:               true,
						Replicas:                  pointer.Int32(2),
						UseMemoryMetricForHvpaHPA: true,
						MinReplicas:               5,
						MaxReplicas:               5,
					},
					SNIConfig{},
					defaultExpectedScaleDownUpdateMode,
					[]autoscalingv2beta1.MetricSpec{
						{
							Type: "Resource",
							Resource: &autoscalingv2beta1.ResourceMetricSource{
								Name:                     "cpu",
								TargetAverageUtilization: pointer.Int32(80),
							},
						},
						{
							Type: "Resource",
							Resource: &autoscalingv2beta1.ResourceMetricSource{
								Name:                     "memory",
								TargetAverageUtilization: pointer.Int32(80),
							},
						},
					},
					defaultExpectedVPAContainerResourcePolicies,
					defaultExpectedWeightBasedScalingIntervals,
				),
				Entry("scale down is disabled",
					AutoscalingConfig{
						HVPAEnabled:              true,
						Replicas:                 pointer.Int32(2),
						MinReplicas:              5,
						MaxReplicas:              5,
						ScaleDownDisabledForHvpa: true,
					},
					SNIConfig{},
					"Off",
					defaultExpectedHPAMetrics,
					defaultExpectedVPAContainerResourcePolicies,
					defaultExpectedWeightBasedScalingIntervals,
				),
				Entry("SNI pod mutator is enabled",
					AutoscalingConfig{
						HVPAEnabled: true,
						Replicas:    pointer.Int32(2),
						MinReplicas: 5,
						MaxReplicas: 5,
					},
					SNIConfig{
						PodMutatorEnabled: true,
					},
					defaultExpectedScaleDownUpdateMode,
					defaultExpectedHPAMetrics,
					[]autoscalingv1beta2.ContainerResourcePolicy{
						{
							ContainerName: "kube-apiserver",
							MinAllowed: corev1.ResourceList{
								"cpu":    resource.MustParse("300m"),
								"memory": resource.MustParse("400M"),
							},
							MaxAllowed: corev1.ResourceList{
								"cpu":    resource.MustParse("8"),
								"memory": resource.MustParse("25G"),
							},
						},
						{
							ContainerName: "vpn-seed",
							Mode:          &containerPolicyOff,
						},
						{
							ContainerName: "apiserver-proxy-pod-mutator",
							Mode:          &containerPolicyOff,
						},
					},
					defaultExpectedWeightBasedScalingIntervals,
				),
				Entry("max replicas > min replicas",
					AutoscalingConfig{
						HVPAEnabled: true,
						Replicas:    pointer.Int32(2),
						MinReplicas: 3,
						MaxReplicas: 5,
					},
					SNIConfig{},
					defaultExpectedScaleDownUpdateMode,
					defaultExpectedHPAMetrics,
					defaultExpectedVPAContainerResourcePolicies,
					[]hvpav1alpha1.WeightBasedScalingInterval{
						{
							VpaWeight:         100,
							StartReplicaCount: 5,
							LastReplicaCount:  5,
						},
						{
							VpaWeight:         0,
							StartReplicaCount: 3,
							LastReplicaCount:  4,
						},
					},
				),
			)
		})

		Describe("PodDisruptionBudget", func() {
			It("should successfully deploy the PDB resource", func() {
				Expect(c.Get(ctx, client.ObjectKeyFromObject(podDisruptionBudget), podDisruptionBudget)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: policyv1beta1.SchemeGroupVersion.Group, Resource: "poddisruptionbudgets"}, podDisruptionBudget.Name)))
				Expect(kapi.Deploy(ctx)).To(Succeed())
				Expect(c.Get(ctx, client.ObjectKeyFromObject(podDisruptionBudget), podDisruptionBudget)).To(Succeed())
				Expect(podDisruptionBudget).To(DeepEqual(&policyv1beta1.PodDisruptionBudget{
					TypeMeta: metav1.TypeMeta{
						APIVersion: policyv1beta1.SchemeGroupVersion.String(),
						Kind:       "PodDisruptionBudget",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            podDisruptionBudget.Name,
						Namespace:       podDisruptionBudget.Namespace,
						ResourceVersion: "1",
						Labels: map[string]string{
							"app":  "kubernetes",
							"role": "apiserver",
						},
					},
					Spec: policyv1beta1.PodDisruptionBudgetSpec{
						MaxUnavailable: intOrStrPtr(intstr.FromInt(1)),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app":  "kubernetes",
								"role": "apiserver",
							},
						},
					},
				}))
			})
		})
	})

	Describe("#Destroy", func() {
		It("should successfully delete all expected resources", func() {
			Expect(c.Create(ctx, deployment)).To(Succeed())
			Expect(c.Create(ctx, horizontalPodAutoscaler)).To(Succeed())
			Expect(c.Create(ctx, verticalPodAutoscaler)).To(Succeed())
			Expect(c.Create(ctx, hvpa)).To(Succeed())
			Expect(c.Create(ctx, podDisruptionBudget)).To(Succeed())

			Expect(c.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)).To(Succeed())
			Expect(c.Get(ctx, client.ObjectKeyFromObject(horizontalPodAutoscaler), horizontalPodAutoscaler)).To(Succeed())
			Expect(c.Get(ctx, client.ObjectKeyFromObject(verticalPodAutoscaler), verticalPodAutoscaler)).To(Succeed())
			Expect(c.Get(ctx, client.ObjectKeyFromObject(hvpa), hvpa)).To(Succeed())
			Expect(c.Get(ctx, client.ObjectKeyFromObject(podDisruptionBudget), podDisruptionBudget)).To(Succeed())

			Expect(kapi.Destroy(ctx)).To(Succeed())

			Expect(c.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: appsv1.SchemeGroupVersion.Group, Resource: "deployments"}, deployment.Name)))
			Expect(c.Get(ctx, client.ObjectKeyFromObject(horizontalPodAutoscaler), horizontalPodAutoscaler)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: autoscalingv2beta1.SchemeGroupVersion.Group, Resource: "horizontalpodautoscalers"}, horizontalPodAutoscaler.Name)))
			Expect(c.Get(ctx, client.ObjectKeyFromObject(verticalPodAutoscaler), verticalPodAutoscaler)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: autoscalingv1beta2.SchemeGroupVersion.Group, Resource: "verticalpodautoscalers"}, verticalPodAutoscaler.Name)))
			Expect(c.Get(ctx, client.ObjectKeyFromObject(hvpa), hvpa)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: hvpav1alpha1.SchemeGroupVersionHvpa.Group, Resource: "hvpas"}, hvpa.Name)))
			Expect(c.Get(ctx, client.ObjectKeyFromObject(podDisruptionBudget), podDisruptionBudget)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: policyv1beta1.SchemeGroupVersion.Group, Resource: "poddisruptionbudgets"}, podDisruptionBudget.Name)))
		})
	})

	Describe("#WaitCleanup", func() {
		It("should successfully wait for the deployment to be deleted", func() {
			fakeClient := fakeclient.NewClientBuilder().WithScheme(kubernetes.SeedScheme).Build()
			fakeKubernetesInterface := fakekubernetes.NewClientSetBuilder().WithAPIReader(fakeClient).WithClient(fakeClient).Build()
			kapi = New(fakeKubernetesInterface, namespace, Values{})
			deploy := deployment.DeepCopy()

			defer test.WithVars(&IntervalWaitForDeployment, time.Millisecond)()
			defer test.WithVars(&TimeoutWaitForDeployment, 100*time.Millisecond)()

			Expect(fakeClient.Create(ctx, deploy)).To(Succeed())
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(deploy), deploy)).To(Succeed())

			timer := time.AfterFunc(10*time.Millisecond, func() {
				Expect(fakeClient.Delete(ctx, deploy)).To(Succeed())
				Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(deploy), deploy)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: appsv1.SchemeGroupVersion.Group, Resource: "deployments"}, deploy.Name)))
			})
			defer timer.Stop()

			Expect(kapi.WaitCleanup(ctx)).To(Succeed())
		})

		It("should time out while waiting for the deployment to be deleted", func() {
			defer test.WithVars(&IntervalWaitForDeployment, time.Millisecond)()
			defer test.WithVars(&TimeoutWaitForDeployment, 100*time.Millisecond)()

			Expect(c.Create(ctx, deployment)).To(Succeed())

			Expect(kapi.WaitCleanup(ctx)).To(MatchError(ContainSubstring("context deadline exceeded")))
		})

		It("should abort due to a severe error while waiting for the deployment to be deleted", func() {
			defer test.WithVars(&IntervalWaitForDeployment, time.Millisecond)()

			Expect(c.Create(ctx, deployment)).To(Succeed())

			scheme := runtime.NewScheme()
			clientWithoutScheme := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
			kubernetesInterface2 := fakekubernetes.NewClientSetBuilder().WithClient(clientWithoutScheme).Build()
			kapi = New(kubernetesInterface2, namespace, Values{})

			Expect(runtime.IsNotRegisteredError(kapi.WaitCleanup(ctx))).To(BeTrue())
		})
	})
})

func intOrStrPtr(intOrStr intstr.IntOrString) *intstr.IntOrString {
	return &intOrStr
}
