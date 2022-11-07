// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/gardener/gardener/pkg/utils/kubernetes"
)

var _ = Describe("HighAvailability", func() {
	DescribeTable("#GetReplicaCount",
		func(criteria string, failureToleranceType *gardencorev1beta1.FailureToleranceType, componentType string, matcher gomegatypes.GomegaMatcher) {
			Expect(GetReplicaCount(criteria, failureToleranceType, componentType)).To(matcher)
		},

		Entry("component type is empty", "foo", nil, "", BeNil()),
		Entry("criteria 'zones', component type 'server'", "zones", nil, "server", Equal(pointer.Int32(2))),
		Entry("criteria 'zones', component type 'controller'", "zones", nil, "controller", Equal(pointer.Int32(2))),
		Entry("criteria 'failure-tolerance-type', component 'server'", "failure-tolerance-type", nil, "server", Equal(pointer.Int32(2))),
		Entry("criteria 'failure-tolerance-type', component 'controller', failure-tolerance-type nil", "failure-tolerance-type", nil, "controller", Equal(pointer.Int32(1))),
		Entry("criteria 'failure-tolerance-type', component 'controller', failure-tolerance-type empty", "failure-tolerance-type", failureToleranceTypePtr(""), "controller", Equal(pointer.Int32(1))),
		Entry("criteria 'failure-tolerance-type', component 'controller', failure-tolerance-type non-empty", "failure-tolerance-type", failureToleranceTypePtr("foo"), "controller", Equal(pointer.Int32(2))),
	)

	zones := []string{"a", "b", "c"}

	DescribeTable("#GetNodeAffinitySelectorTermsForZones",
		func(failureToleranceType *gardencorev1beta1.FailureToleranceType, zones []string, matcher gomegatypes.GomegaMatcher) {
			Expect(GetNodeAffinitySelectorTermsForZones(failureToleranceType, zones)).To(matcher)
		},

		Entry("no zones", nil, nil, BeNil()),
		Entry("no failure-tolerance-type", nil, zones, BeNil()),
		Entry("zones and failure-tolerance-type set", failureToleranceTypePtr(""), zones, ConsistOf(corev1.NodeSelectorTerm{MatchExpressions: []corev1.NodeSelectorRequirement{{Key: corev1.LabelTopologyZone, Operator: corev1.NodeSelectorOpIn, Values: zones}}})),
	)
})

func failureToleranceTypePtr(v gardencorev1beta1.FailureToleranceType) *gardencorev1beta1.FailureToleranceType {
	return &v
}
