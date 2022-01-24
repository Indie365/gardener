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

package validation_test

import (
	"github.com/gardener/gardener/pkg/apis/core"
	. "github.com/gardener/gardener/pkg/apis/core/validation"

	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var _ = Describe("Quota Validation Tests ", func() {
	Describe("#ValidateQuota", func() {
		var quota *core.Quota

		BeforeEach(func() {
			quota = &core.Quota{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "quota-1",
					Namespace: "my-namespace",
				},
				Spec: core.QuotaSpec{
					Scope: corev1.ObjectReference{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					Metrics: corev1.ResourceList{
						"cpu":    resource.MustParse("200"),
						"memory": resource.MustParse("4000Gi"),
					},
				},
			}
		})

		It("should not return any errors", func() {
			errorList := ValidateQuota(quota)

			Expect(errorList).To(HaveLen(0))
		})

		DescribeTable("Quota metadata",
			func(objectMeta metav1.ObjectMeta, matcher gomegatypes.GomegaMatcher) {
				quota.ObjectMeta = objectMeta

				errorList := ValidateQuota(quota)

				Expect(errorList).To(matcher)
			},

			Entry("should forbid Quota with empty metadata",
				metav1.ObjectMeta{},
				ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("metadata.name"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("metadata.namespace"),
					})),
				),
			),
			Entry("should forbid Quota with empty name",
				metav1.ObjectMeta{Name: "", Namespace: "my-namespace"},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("metadata.name"),
				}))),
			),
			Entry("should allow Quota with '.' in the name",
				metav1.ObjectMeta{Name: "quota.test", Namespace: "my-namespace"},
				BeEmpty(),
			),
			Entry("should forbid Quota with '_' in the name (not a DNS-1123 subdomain)",
				metav1.ObjectMeta{Name: "quota_test", Namespace: "my-namespace"},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("metadata.name"),
				}))),
			),
		)

		It("should forbid Quota specification with empty or invalid keys", func() {
			quota.ObjectMeta = metav1.ObjectMeta{}
			quota.Spec.Scope = corev1.ObjectReference{}
			quota.Spec.Metrics["key"] = resource.MustParse("-100")

			errorList := ValidateQuota(quota)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("metadata.name"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("metadata.namespace"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeNotSupported),
					"Field": Equal("spec.scope"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.metrics[key]"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.metrics[key]"),
				})),
			))
		})
	})
})
