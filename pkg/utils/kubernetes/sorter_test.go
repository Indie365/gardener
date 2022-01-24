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

package kubernetes_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/gardener/gardener/pkg/utils/kubernetes"
)

var _ = Describe("Sort", func() {
	var (
		listObj = &corev1.PodList{}

		pod1 = corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pod1",
				CreationTimestamp: metav1.Now(),
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{}, {}},
			},
		}
		pod2 = corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pod2",
				CreationTimestamp: metav1.Time{Time: time.Now().Add(-time.Hour)},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{}},
			},
		}
		pod3 = corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pod3",
				CreationTimestamp: metav1.Time{Time: time.Now().Add(time.Hour)},
			},
		}
	)

	Describe("ByName", func() {
		It("should sort correctly", func() {
			listObj.Items = []corev1.Pod{pod2, pod3, pod1}
			ByName().Sort(listObj)
			Expect(listObj.Items).To(Equal([]corev1.Pod{pod1, pod2, pod3}))
		})
	})

	Describe("ByCreationTimestamp", func() {
		It("should sort correctly", func() {
			listObj.Items = []corev1.Pod{pod1, pod2, pod3}
			ByCreationTimestamp().Sort(listObj)
			Expect(listObj.Items).To(Equal([]corev1.Pod{pod2, pod1, pod3}))
		})
	})

	Describe("SortBy", func() {
		It("should sort correctly", func() {
			sortByContainers := func(o1, o2 client.Object) bool {
				obj1, ok1 := o1.(*corev1.Pod)
				obj2, ok2 := o2.(*corev1.Pod)

				if !ok1 || !ok2 {
					return false
				}

				return len(obj1.Spec.Containers) < len(obj2.Spec.Containers)
			}

			listObj.Items = []corev1.Pod{pod1, pod2, pod3}
			SortBy(sortByContainers).Sort(listObj)
			Expect(listObj.Items).To(Equal([]corev1.Pod{pod3, pod2, pod1}))
		})
	})
})
