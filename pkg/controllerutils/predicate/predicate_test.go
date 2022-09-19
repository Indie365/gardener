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

package predicate_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/gardener/gardener/pkg/controllerutils/predicate"
)

var _ = Describe("Predicate", func() {
	Describe("#IsDeleting", func() {
		var (
			shoot        *gardencorev1beta1.Shoot
			predicate    predicate.Predicate
			createEvent  event.CreateEvent
			updateEvent  event.UpdateEvent
			deleteEvent  event.DeleteEvent
			genericEvent event.GenericEvent
		)

		BeforeEach(func() {
			shoot = &gardencorev1beta1.Shoot{
				ObjectMeta: metav1.ObjectMeta{},
			}

			predicate = IsDeleting()

			createEvent = event.CreateEvent{
				Object: shoot,
			}
			updateEvent = event.UpdateEvent{
				ObjectOld: shoot,
				ObjectNew: shoot,
			}
			deleteEvent = event.DeleteEvent{
				Object: shoot,
			}
			genericEvent = event.GenericEvent{
				Object: shoot,
			}
		})

		Context("shoot doesn't have a deletion timestamp", func() {
			It("should be false", func() {
				gomega.Expect(predicate.Create(createEvent)).To(gomega.BeFalse())
				gomega.Expect(predicate.Update(updateEvent)).To(gomega.BeFalse())
				gomega.Expect(predicate.Delete(deleteEvent)).To(gomega.BeFalse())
				gomega.Expect(predicate.Generic(genericEvent)).To(gomega.BeFalse())
			})
		})

		Context("shoot has a deletion timestamp", func() {
			time := metav1.Now()

			BeforeEach(func() {
				shoot.ObjectMeta.DeletionTimestamp = &time
			})

			It("should be true", func() {
				gomega.Expect(predicate.Create(createEvent)).To(gomega.BeTrue())
				gomega.Expect(predicate.Update(updateEvent)).To(gomega.BeTrue())
				gomega.Expect(predicate.Delete(deleteEvent)).To(gomega.BeTrue())
				gomega.Expect(predicate.Generic(genericEvent)).To(gomega.BeTrue())
			})
		})
	})

	Describe("#ShootIsUnassigned", func() {
		var (
			shoot        *gardencorev1beta1.Shoot
			predicate    predicate.Predicate
			createEvent  event.CreateEvent
			updateEvent  event.UpdateEvent
			deleteEvent  event.DeleteEvent
			genericEvent event.GenericEvent
		)

		BeforeEach(func() {
			shoot = &gardencorev1beta1.Shoot{
				Spec: gardencorev1beta1.ShootSpec{},
			}

			predicate = ShootIsUnassigned()

			createEvent = event.CreateEvent{
				Object: shoot,
			}
			updateEvent = event.UpdateEvent{
				ObjectOld: shoot,
				ObjectNew: shoot,
			}
			deleteEvent = event.DeleteEvent{
				Object: shoot,
			}
			genericEvent = event.GenericEvent{
				Object: shoot,
			}
		})

		Context("shoot is unassigned", func() {
			It("should be true", func() {
				gomega.Expect(predicate.Create(createEvent)).To(gomega.BeTrue())
				gomega.Expect(predicate.Update(updateEvent)).To(gomega.BeTrue())
				gomega.Expect(predicate.Delete(deleteEvent)).To(gomega.BeTrue())
				gomega.Expect(predicate.Generic(genericEvent)).To(gomega.BeTrue())
			})
		})

		Context("shoot is assigned", func() {
			BeforeEach(func() {
				shoot.Spec.SeedName = pointer.String("seed")
			})

			It("should be false", func() {
				gomega.Expect(predicate.Create(createEvent)).To(gomega.BeFalse())
				gomega.Expect(predicate.Update(updateEvent)).To(gomega.BeFalse())
				gomega.Expect(predicate.Delete(deleteEvent)).To(gomega.BeFalse())
				gomega.Expect(predicate.Generic(genericEvent)).To(gomega.BeFalse())
			})
		})
	})

	Describe("#Not", func() {
		It("should invert predicate", func() {
			predicate := Not(predicate.Funcs{
				CreateFunc: func(_ event.CreateEvent) bool {
					return true
				},
				UpdateFunc: func(_ event.UpdateEvent) bool {
					return true
				},
				GenericFunc: func(_ event.GenericEvent) bool {
					return true
				},
				DeleteFunc: func(_ event.DeleteEvent) bool {
					return true
				},
			})

			gomega.Expect(predicate.Create(event.CreateEvent{})).To(gomega.BeFalse())
			gomega.Expect(predicate.Update(event.UpdateEvent{})).To(gomega.BeFalse())
			gomega.Expect(predicate.Delete(event.DeleteEvent{})).To(gomega.BeFalse())
			gomega.Expect(predicate.Generic(event.GenericEvent{})).To(gomega.BeFalse())
		})
	})

	DescribeTable("#ForEventTypes",
		func(events []EventType, createMatcher, updateMatcher, deleteMatcher, genericMatcher gomegatypes.GomegaMatcher) {
			p := ForEventTypes(events...)

			gomega.Expect(p.Create(event.CreateEvent{})).To(createMatcher)
			gomega.Expect(p.Update(event.UpdateEvent{})).To(updateMatcher)
			gomega.Expect(p.Delete(event.DeleteEvent{})).To(deleteMatcher)
			gomega.Expect(p.Generic(event.GenericEvent{})).To(genericMatcher)
		},

		Entry("none", nil, gomega.BeFalse(), gomega.BeFalse(), gomega.BeFalse(), gomega.BeFalse()),
		Entry("create", []EventType{Create}, gomega.BeTrue(), gomega.BeFalse(), gomega.BeFalse(), gomega.BeFalse()),
		Entry("update", []EventType{Update}, gomega.BeFalse(), gomega.BeTrue(), gomega.BeFalse(), gomega.BeFalse()),
		Entry("delete", []EventType{Delete}, gomega.BeFalse(), gomega.BeFalse(), gomega.BeTrue(), gomega.BeFalse()),
		Entry("generic", []EventType{Generic}, gomega.BeFalse(), gomega.BeFalse(), gomega.BeFalse(), gomega.BeTrue()),
		Entry("create, update", []EventType{Create, Update}, gomega.BeTrue(), gomega.BeTrue(), gomega.BeFalse(), gomega.BeFalse()),
		Entry("create, delete", []EventType{Create, Delete}, gomega.BeTrue(), gomega.BeFalse(), gomega.BeTrue(), gomega.BeFalse()),
		Entry("create, generic", []EventType{Create, Generic}, gomega.BeTrue(), gomega.BeFalse(), gomega.BeFalse(), gomega.BeTrue()),
		Entry("update, delete", []EventType{Update, Delete}, gomega.BeFalse(), gomega.BeTrue(), gomega.BeTrue(), gomega.BeFalse()),
		Entry("update, generic", []EventType{Update, Generic}, gomega.BeFalse(), gomega.BeTrue(), gomega.BeFalse(), gomega.BeTrue()),
		Entry("delete, generic", []EventType{Delete, Generic}, gomega.BeFalse(), gomega.BeFalse(), gomega.BeTrue(), gomega.BeTrue()),
		Entry("create, update, delete", []EventType{Create, Update, Delete}, gomega.BeTrue(), gomega.BeTrue(), gomega.BeTrue(), gomega.BeFalse()),
		Entry("create, update, generic", []EventType{Create, Update, Generic}, gomega.BeTrue(), gomega.BeTrue(), gomega.BeFalse(), gomega.BeTrue()),
		Entry("create, delete, generic", []EventType{Create, Delete, Generic}, gomega.BeTrue(), gomega.BeFalse(), gomega.BeTrue(), gomega.BeTrue()),
		Entry("update, delete, generic", []EventType{Update, Delete, Generic}, gomega.BeFalse(), gomega.BeTrue(), gomega.BeTrue(), gomega.BeTrue()),
		Entry("create, update, delete, generic", []EventType{Create, Update, Delete, Generic}, gomega.BeTrue(), gomega.BeTrue(), gomega.BeTrue(), gomega.BeTrue()),
	)
})
