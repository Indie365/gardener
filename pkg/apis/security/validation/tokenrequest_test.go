// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/gardener/pkg/apis/security"
	. "github.com/gardener/gardener/pkg/apis/security/validation"
)

var _ = Describe("TokenRequest Validation Tests", func() {
	Describe("#ValidateTokenRequest", func() {
		var tokenRequest *security.TokenRequest

		BeforeEach(func() {
			tokenRequest = &security.TokenRequest{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: security.TokenRequestSpec{
					Duration:      &metav1.Duration{Duration: time.Hour * 12},
					ContextObject: nil,
				},
			}
		})

		DescribeTable("Duration",
			func(duration time.Duration, matcher gomegatypes.GomegaMatcher) {
				tokenRequest.Spec.Duration.Duration = duration

				errs := ValidateTokenRequest(tokenRequest)
				Expect(errs).To(matcher)
			},
			Entry("should allow min < duration < max",
				time.Hour*12,
				BeEmpty(),
			),
			Entry("should allow duration==min",
				time.Minute*10,
				BeEmpty(),
			),
			Entry("should allow duration==max",
				time.Hour*48,
				BeEmpty(),
			),
			Entry("should forbid duration < min",
				time.Minute*10-time.Second,
				ConsistOf(PointTo(
					MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("spec.duration"),
						"Detail": ContainSubstring("may not specify a duration shorter than"),
					}),
				)),
			),
			Entry("should forbid duration > max",
				time.Hour*48+time.Second,
				ConsistOf(PointTo(
					MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("spec.duration"),
						"Detail": ContainSubstring("may not specify a duration longer than"),
					}),
				)),
			),
		)

		DescribeTable("ContextObject",
			func(ctxObj *security.ContextObject, matcher gomegatypes.GomegaMatcher) {
				tokenRequest.Spec.ContextObject = ctxObj

				errs := ValidateTokenRequest(tokenRequest)
				Expect(errs).To(matcher)
			},
			Entry("should allow nil context object",
				nil,
				BeEmpty(),
			),
			Entry("should allow namespaced context object",
				&security.ContextObject{APIVersion: "foo.bar/v1", Kind: "Baz", Namespace: "default", Name: "foo-bar"},
				BeEmpty(),
			),
			Entry("should allow non-namespaced (cluster scoped) context object",
				&security.ContextObject{APIVersion: "foo.bar/v1", Kind: "Baz", Name: "foo-bar"},
				BeEmpty(),
			),
			Entry("should forbid context object with no APIVersion",
				&security.ContextObject{Kind: "Baz", Name: "foo-bar"},
				ConsistOf(PointTo(
					MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeRequired),
						"Field":  Equal("spec.contextObject.apiVersion"),
						"Detail": Equal("must provide an apiVersion"),
					}),
				)),
			),
			Entry("should forbid context object with no Kind",
				&security.ContextObject{APIVersion: "foo.bar/v1", Name: "foo-bar"},
				ConsistOf(PointTo(
					MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeRequired),
						"Field":  Equal("spec.contextObject.kind"),
						"Detail": Equal("must provide a kind"),
					}),
				)),
			),
			Entry("should forbid context object with no Name",
				&security.ContextObject{APIVersion: "foo.bar/v1", Kind: "Baz"},
				ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeRequired),
						"Field":  Equal("spec.contextObject.name"),
						"Detail": Equal("must provide a name"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("spec.contextObject.name"),
						"Detail": ContainSubstring("a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters"),
					})),
				),
			),
			Entry("should forbid context object with Name that is not DNS1123 subdomain",
				&security.ContextObject{APIVersion: "foo.bar/v1", Kind: "Baz", Name: "Foo-Bar"},
				ConsistOf(PointTo(
					MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("spec.contextObject.name"),
						"Detail": ContainSubstring("a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters"),
					}),
				)),
			),
			Entry("should forbid context object with Namespace that is not DNS1123 subdomain",
				&security.ContextObject{APIVersion: "foo.bar/v1", Kind: "Baz", Name: "foo-bar", Namespace: "Default"},
				ConsistOf(PointTo(
					MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("spec.contextObject.namespace"),
						"Detail": ContainSubstring("a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters"),
					}),
				)),
			),
		)
	})
})
