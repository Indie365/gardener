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

package bastion

import (
	"context"

	"github.com/gardener/gardener/pkg/apis/core"
	v1alpha1constants "github.com/gardener/gardener/pkg/apis/core/v1alpha1/constants"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("ToSelectableFields", func() {
	It("should return correct fields", func() {
		result := ToSelectableFields(newBastion("foo"))

		Expect(result).To(HaveLen(3))
		Expect(result.Has(core.BastionSeedName)).To(BeTrue())
		Expect(result.Get(core.BastionSeedName)).To(Equal("foo"))
	})
})

var _ = Describe("GetAttrs", func() {
	It("should return error when object is not Bastion", func() {
		_, _, err := GetAttrs(&core.Seed{})
		Expect(err).To(HaveOccurred())
	})

	It("should return correct result", func() {
		ls, fs, err := GetAttrs(newBastion("foo"))

		Expect(ls).To(HaveLen(1))
		Expect(ls.Get("foo")).To(Equal("bar"))
		Expect(fs.Get(core.BastionSeedName)).To(Equal("foo"))
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("SeedNameTriggerFunc", func() {
	It("should return spec.seedName", func() {
		actual := SeedNameTriggerFunc(newBastion("foo"))
		Expect(actual).To(Equal("foo"))
	})
})

var _ = Describe("MatchBastion", func() {
	It("should return correct predicate", func() {
		ls, _ := labels.Parse("app=test")
		fs := fields.OneTermEqualSelector(core.BastionSeedName, "foo")

		result := MatchBastion(ls, fs)

		Expect(result.Label).To(Equal(ls))
		Expect(result.Field).To(Equal(fs))
		Expect(result.IndexFields).To(ConsistOf(core.BastionSeedName))
	})
})

var _ = Describe("PrepareForCreate", func() {
	It("should perform an initial heartbeat", func() {
		bastion := core.Bastion{}

		Strategy.PrepareForCreate(context.TODO(), &bastion)

		Expect(bastion.Generation).NotTo(BeZero())
		Expect(bastion.Status.LastHeartbeatTimestamp).NotTo(BeNil())
		Expect(bastion.Status.ExpirationTimestamp).NotTo(BeNil())
		Expect(bastion.Annotations[v1alpha1constants.GardenerOperation]).To(BeEmpty())
	})

	It("should remove operation annotation even on creates", func() {
		bastion := core.Bastion{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					v1alpha1constants.GardenerOperation: v1alpha1constants.GardenerOperationKeepalive,
				},
			},
		}

		Strategy.PrepareForCreate(context.TODO(), &bastion)
		Expect(bastion.Annotations[v1alpha1constants.GardenerOperation]).To(BeEmpty())
	})
})

var _ = Describe("PrepareForUpdate", func() {
	It("should not perform heartbeat if no annotation is set", func() {
		bastion := core.Bastion{}

		Strategy.PrepareForUpdate(context.TODO(), &bastion, &bastion)

		Expect(bastion.Status.LastHeartbeatTimestamp).To(BeNil())
		Expect(bastion.Status.ExpirationTimestamp).To(BeNil())
	})

	It("should perform the heartbeat when the annotation is set", func() {
		bastion := core.Bastion{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					v1alpha1constants.GardenerOperation: v1alpha1constants.GardenerOperationKeepalive,
				},
			},
		}

		Strategy.PrepareForUpdate(context.TODO(), &bastion, &bastion)
		Expect(bastion.Status.LastHeartbeatTimestamp).NotTo(BeNil())
		Expect(bastion.Status.ExpirationTimestamp).NotTo(BeNil())
		Expect(bastion.Annotations[v1alpha1constants.GardenerOperation]).To(BeEmpty())
	})
})

var _ = Describe("heartbeat", func() {
	It("should delete keepalive annotation", func() {
		bastion := core.Bastion{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					v1alpha1constants.GardenerOperation: v1alpha1constants.GardenerOperationKeepalive,
				},
			},
		}

		Strategy.heartbeat(&bastion)

		Expect(bastion.Annotations[v1alpha1constants.GardenerOperation]).To(BeEmpty())
	})

	It("should create expirations that are after the heartbeat", func() {
		bastion := core.Bastion{}

		Strategy.heartbeat(&bastion)

		Expect(bastion.Status.LastHeartbeatTimestamp).NotTo(BeNil())
		Expect(bastion.Status.ExpirationTimestamp).NotTo(BeNil())

		heartbeat := bastion.Status.LastHeartbeatTimestamp.Time
		expires := bastion.Status.ExpirationTimestamp.Time

		Expect(expires).Should(BeTemporally(">", heartbeat))
	})
})

func newBastion(seedName string) *core.Bastion {
	return &core.Bastion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-namespace",
			Labels:    map[string]string{"foo": "bar"},
		},
		Spec: core.BastionSpec{
			SeedName: &seedName,
		},
	}
}
