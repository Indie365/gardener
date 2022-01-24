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

package helper_test

import (
	"github.com/gardener/gardener/pkg/scheduler/apis/config"
	. "github.com/gardener/gardener/pkg/scheduler/apis/config/helper"
	"github.com/gardener/gardener/pkg/scheduler/apis/config/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Helpers test", func() {

	Describe("#ConvertSchedulerConfiguration", func() {
		externalConfiguration := v1alpha1.SchedulerConfiguration{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       "SchedulerConfiguration",
			},
		}

		It("should convert the external SchedulerConfiguration to an internal one", func() {
			result, err := ConvertSchedulerConfiguration(&externalConfiguration)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(&config.SchedulerConfiguration{}))
		})
	})

	Describe("#ConvertSchedulerConfigurationExternal", func() {
		internalConfiguration := v1alpha1.SchedulerConfiguration{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       "SchedulerConfiguration",
			},
		}

		It("should convert the internal SchedulerConfiguration to an external one", func() {
			result, err := ConvertSchedulerConfigurationExternal(&internalConfiguration)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(&v1alpha1.SchedulerConfiguration{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
					Kind:       "SchedulerConfiguration",
				},
			}))
		})
	})
})
