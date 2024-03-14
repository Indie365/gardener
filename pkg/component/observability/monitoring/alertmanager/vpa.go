// Copyright 2024 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package alertmanager

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/utils"
)

func (a *alertManager) vpa() *vpaautoscalingv1.VerticalPodAutoscaler {
	updateMode, controlledValuesRequestsOnly, containerScalingModeOff := vpaautoscalingv1.UpdateModeAuto, vpaautoscalingv1.ContainerControlledValuesRequestsOnly, vpaautoscalingv1.ContainerScalingModeOff

	return &vpaautoscalingv1.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.name(),
			Namespace: a.namespace,
			Labels: utils.MergeStringMaps(a.getLabels(), map[string]string{
				v1beta1constants.LabelObservabilityApplication: a.values.Name}),
		},
		Spec: vpaautoscalingv1.VerticalPodAutoscalerSpec{
			TargetRef: &autoscalingv1.CrossVersionObjectReference{
				APIVersion: appsv1.SchemeGroupVersion.String(),
				Kind:       "StatefulSet",
				Name:       a.name(),
			},
			UpdatePolicy: &vpaautoscalingv1.PodUpdatePolicy{
				UpdateMode: &updateMode,
			},
			ResourcePolicy: &vpaautoscalingv1.PodResourcePolicy{
				ContainerPolicies: []vpaautoscalingv1.ContainerResourcePolicy{
					{
						ContainerName: "alertmanager",
						MinAllowed: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("20Mi"),
						},
						MaxAllowed: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("200Mi"),
						},
						ControlledValues: &controlledValuesRequestsOnly,
					},
					{
						ContainerName: "config-reloader",
						Mode:          &containerScalingModeOff,
					},
				},
			},
		},
	}
}
