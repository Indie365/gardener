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

package extensionscheck_test

import (
	"context"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	. "github.com/gardener/gardener/pkg/controllermanager/controller/seed/extensionscheck"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Add", func() {
	var (
		reconciler             *Reconciler
		controllerInstallation *gardencorev1beta1.ControllerInstallation
	)

	BeforeEach(func() {
		reconciler = &Reconciler{}
		controllerInstallation = &gardencorev1beta1.ControllerInstallation{
			Spec: gardencorev1beta1.ControllerInstallationSpec{
				SeedRef: corev1.ObjectReference{
					Name: "seed",
				},
			},
		}
	})

	Describe("ControllerInstallationPredicate", func() {
		var p predicate.Predicate

		BeforeEach(func() {
			p = reconciler.ControllerInstallationPredicate()
		})

		Describe("#Create", func() {
			It("should return true", func() {
				Expect(p.Create(event.CreateEvent{})).To(BeTrue())
			})
		})

		Describe("#Update", func() {
			It("should return false because object is no ControllerInstallation", func() {
				Expect(p.Update(event.UpdateEvent{})).To(BeFalse())
			})

			It("should return false because old object is no ControllerInstallation", func() {
				Expect(p.Update(event.UpdateEvent{ObjectNew: controllerInstallation})).To(BeFalse())
			})

			It("should return false because there is no relevant change", func() {
				Expect(p.Update(event.UpdateEvent{ObjectNew: controllerInstallation, ObjectOld: controllerInstallation})).To(BeFalse())
			})

			tests := func(conditionType gardencorev1beta1.ConditionType) {
				It("should return true because condition was added", func() {
					oldControllerInstallation := controllerInstallation.DeepCopy()
					controllerInstallation.Status.Conditions = []gardencorev1beta1.Condition{{Type: conditionType}}
					Expect(p.Update(event.UpdateEvent{ObjectNew: controllerInstallation, ObjectOld: oldControllerInstallation})).To(BeTrue())
				})

				It("should return true because condition was removed", func() {
					controllerInstallation.Status.Conditions = []gardencorev1beta1.Condition{{Type: conditionType}}
					oldControllerInstallation := controllerInstallation.DeepCopy()
					controllerInstallation.Status.Conditions = nil
					Expect(p.Update(event.UpdateEvent{ObjectNew: controllerInstallation, ObjectOld: oldControllerInstallation})).To(BeTrue())
				})

				It("should return true because condition status was changed", func() {
					controllerInstallation.Status.Conditions = []gardencorev1beta1.Condition{{Type: conditionType}}
					oldControllerInstallation := controllerInstallation.DeepCopy()
					controllerInstallation.Status.Conditions[0].Status = gardencorev1beta1.ConditionTrue
					Expect(p.Update(event.UpdateEvent{ObjectNew: controllerInstallation, ObjectOld: oldControllerInstallation})).To(BeTrue())
				})

				It("should return true because condition reason was changed", func() {
					controllerInstallation.Status.Conditions = []gardencorev1beta1.Condition{{Type: conditionType}}
					oldControllerInstallation := controllerInstallation.DeepCopy()
					controllerInstallation.Status.Conditions[0].Reason = "reason"
					Expect(p.Update(event.UpdateEvent{ObjectNew: controllerInstallation, ObjectOld: oldControllerInstallation})).To(BeTrue())
				})

				It("should return true because condition message was changed", func() {
					controllerInstallation.Status.Conditions = []gardencorev1beta1.Condition{{Type: conditionType}}
					oldControllerInstallation := controllerInstallation.DeepCopy()
					controllerInstallation.Status.Conditions[0].Message = "message"
					Expect(p.Update(event.UpdateEvent{ObjectNew: controllerInstallation, ObjectOld: oldControllerInstallation})).To(BeTrue())
				})
			}

			Context("Valid condition", func() {
				tests(gardencorev1beta1.ControllerInstallationValid)
			})

			Context("Installed condition", func() {
				tests(gardencorev1beta1.ControllerInstallationInstalled)
			})

			Context("Healthy condition", func() {
				tests(gardencorev1beta1.ControllerInstallationHealthy)
			})

			Context("Progressing condition", func() {
				tests(gardencorev1beta1.ControllerInstallationProgressing)
			})
		})

		Describe("#Delete", func() {
			It("should return true", func() {
				Expect(p.Delete(event.DeleteEvent{})).To(BeTrue())
			})
		})

		Describe("#Generic", func() {
			It("should return true", func() {
				Expect(p.Generic(event.GenericEvent{})).To(BeTrue())
			})
		})
	})

	Describe("#MapControllerInstallationToSeed", func() {
		var (
			ctx        = context.TODO()
			log        logr.Logger
			fakeClient client.Client
		)

		BeforeEach(func() {
			log = logr.Discard()
			fakeClient = fakeclient.NewClientBuilder().WithScheme(kubernetes.GardenScheme).Build()
		})

		It("should do nothing if the object is no ControllerInstallation", func() {
			Expect(reconciler.MapControllerInstallationToSeed(ctx, log, fakeClient, &corev1.Secret{})).To(BeEmpty())
		})

		It("should map the ControllerInstallation to the Seed", func() {
			Expect(reconciler.MapControllerInstallationToSeed(ctx, log, fakeClient, controllerInstallation)).To(ConsistOf(
				reconcile.Request{NamespacedName: types.NamespacedName{Name: controllerInstallation.Spec.SeedRef.Name}},
			))
		})
	})
})
