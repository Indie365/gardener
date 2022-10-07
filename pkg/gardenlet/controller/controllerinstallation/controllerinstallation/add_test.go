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

package controllerinstallation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/gardener/gardener/pkg/gardenlet/controller/controllerinstallation/controllerinstallation"
)

var _ = Describe("Add", func() {
	var (
		p                      predicate.Predicate
		controllerInstallation *gardencorev1beta1.ControllerInstallation
	)

	Describe("#ControllerInstallationPredicate", func() {
		BeforeEach(func() {
			p = (&Reconciler{}).ControllerInstallationPredicate()
			controllerInstallation = &gardencorev1beta1.ControllerInstallation{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
			}
		})

		Describe("#Create", func() {
			It("should return false", func() {
				Expect(p.Create(event.CreateEvent{})).To(BeTrue())
			})
		})

		Describe("#Update", func() {
			It("should return true for periodic cache resyncs", func() {
				Expect(p.Update(event.UpdateEvent{ObjectNew: controllerInstallation, ObjectOld: controllerInstallation.DeepCopy()})).To(BeTrue())
			})

			It("should return true if deletion timestamp changed", func() {
				oldControllerInstallation := controllerInstallation.DeepCopy()
				controllerInstallation.ResourceVersion = "2"
				controllerInstallation.DeletionTimestamp = &metav1.Time{}

				Expect(p.Update(event.UpdateEvent{ObjectNew: controllerInstallation, ObjectOld: oldControllerInstallation})).To(BeTrue())
			})

			It("should return true if deployment ref changed", func() {
				oldControllerInstallation := controllerInstallation.DeepCopy()
				controllerInstallation.ResourceVersion = "2"
				controllerInstallation.Spec.DeploymentRef = &corev1.ObjectReference{}

				Expect(p.Update(event.UpdateEvent{ObjectNew: controllerInstallation, ObjectOld: oldControllerInstallation})).To(BeTrue())
			})

			It("should return true if registration ref's resourceVersion changed", func() {
				oldControllerInstallation := controllerInstallation.DeepCopy()
				controllerInstallation.ResourceVersion = "2"
				controllerInstallation.Spec.RegistrationRef.ResourceVersion = "foo"

				Expect(p.Update(event.UpdateEvent{ObjectNew: controllerInstallation, ObjectOld: oldControllerInstallation})).To(BeTrue())
			})

			It("should return true if seed ref's resourceVersion changed", func() {
				oldControllerInstallation := controllerInstallation.DeepCopy()
				controllerInstallation.ResourceVersion = "2"
				controllerInstallation.Spec.SeedRef.ResourceVersion = "foo"

				Expect(p.Update(event.UpdateEvent{ObjectNew: controllerInstallation, ObjectOld: oldControllerInstallation})).To(BeTrue())
			})

			It("should return false if something else changed", func() {
				oldControllerInstallation := controllerInstallation.DeepCopy()
				controllerInstallation.ResourceVersion = "2"
				metav1.SetMetaDataLabel(&controllerInstallation.ObjectMeta, "foo", "bar")

				Expect(p.Update(event.UpdateEvent{ObjectNew: controllerInstallation, ObjectOld: oldControllerInstallation})).To(BeFalse())
			})
		})

		Describe("#Delete", func() {
			It("should return true", func() {
				Expect(p.Delete(event.DeleteEvent{})).To(BeTrue())
			})
		})

		Describe("#Generic", func() {
			It("should return false", func() {
				Expect(p.Generic(event.GenericEvent{})).To(BeTrue())
			})
		})
	})
})
