// Copyright 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package controllerregistrationfinalizer_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/gardener/pkg/api/indexer"
	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	. "github.com/gardener/gardener/pkg/controllermanager/controller/controllerregistration/controllerregistrationfinalizer"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
)

var _ = Describe("ControllerRegistration", func() {
	var (
		ctx = context.TODO()

		ctrl *gomock.Controller
		c    client.Client

		reconciler             *Reconciler
		controllerRegistration *gardencorev1beta1.ControllerRegistration

		controllerRegistrationName = "controllerRegistration"
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		c = fakeclient.NewClientBuilder().
			WithScheme(kubernetes.GardenScheme).
			WithIndex(&gardencorev1beta1.ControllerInstallation{}, core.RegistrationRefName, indexer.ControllerInstallationRegistrationRefNameIndexerFunc).
			Build()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("Reconciler", func() {
		BeforeEach(func() {
			reconciler = &Reconciler{Client: c}
			controllerRegistration = &gardencorev1beta1.ControllerRegistration{
				ObjectMeta: metav1.ObjectMeta{
					Name: controllerRegistrationName,
				},
			}
		})

		It("should return nil because object not found", func() {
			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: controllerRegistrationName}})
			Expect(result).To(Equal(reconcile.Result{}))
			Expect(err).NotTo(HaveOccurred())
		})

		Context("deletion timestamp not set", func() {
			BeforeEach(func() {
				Expect(c.Create(ctx, controllerRegistration)).To(Succeed())
			})

			It("should ensure the finalizer", func() {
				result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: controllerRegistrationName}})
				Expect(result).To(Equal(reconcile.Result{}))
				Expect(err).NotTo(HaveOccurred())

				Expect(c.Get(ctx, client.ObjectKeyFromObject(controllerRegistration), controllerRegistration)).To(Succeed())
				Expect(controllerRegistration.Finalizers).To(ConsistOf("core.gardener.cloud/controllerregistration"))
			})
		})

		Context("deletion timestamp set", func() {
			BeforeEach(func() {
				Expect(c.Create(ctx, controllerRegistration)).To(Succeed())
				result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: controllerRegistrationName}})
				Expect(result).To(Equal(reconcile.Result{}))
				Expect(err).NotTo(HaveOccurred())
				Expect(c.Get(ctx, client.ObjectKeyFromObject(controllerRegistration), controllerRegistration)).To(Succeed())
				Expect(controllerRegistration.Finalizers).To(ConsistOf("core.gardener.cloud/controllerregistration"))

				Expect(c.Delete(ctx, controllerRegistration)).To(Succeed())
			})

			It("should return an error because installation referencing controllerRegistration exists", func() {
				controllerInstallation := &gardencorev1beta1.ControllerInstallation{
					ObjectMeta: metav1.ObjectMeta{
						Name: "controllerInstallation",
					},
					Spec: gardencorev1beta1.ControllerInstallationSpec{
						RegistrationRef: corev1.ObjectReference{
							Name: controllerRegistrationName,
						},
					},
				}

				Expect(c.Create(ctx, controllerInstallation)).To(Succeed())

				result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: controllerRegistrationName}})
				Expect(result).To(Equal(reconcile.Result{}))
				Expect(err).To(MatchError(ContainSubstring("cannot remove finalizer")))
			})

			It("should remove the finalizer", func() {
				result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: controllerRegistrationName}})
				Expect(result).To(Equal(reconcile.Result{}))
				Expect(err).NotTo(HaveOccurred())

				Expect(c.Get(ctx, client.ObjectKeyFromObject(controllerRegistration), controllerRegistration)).To(BeNotFoundError())
			})
		})
	})
})
