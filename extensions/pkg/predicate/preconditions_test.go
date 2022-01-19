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

package predicate_test

import (
	"context"

	. "github.com/gardener/gardener/extensions/pkg/predicate"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	mockcache "github.com/gardener/gardener/pkg/mock/controller-runtime/cache"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

var _ = Describe("Preconditions", func() {

	Describe("IsInGardenNamespacePredicate", func() {
		var (
			pred predicate.Predicate
			obj  *extensionsv1alpha1.Infrastructure
		)

		BeforeEach(func() {
			pred = IsInGardenNamespacePredicate
			obj = &extensionsv1alpha1.Infrastructure{}
		})

		Describe("#Create, #Update, #Delete, #Generic", func() {
			tests := func(run func(client.Object) interface{}) {
				It("should return false because obj is nil", func() {
					Expect(run(nil)).To(BeFalse())
				})

				It("should return false because obj is not in garden namespace", func() {
					obj.SetNamespace("foo")
					Expect(run(obj)).To(BeFalse())
				})

				It("should return true because obj is in garden namespace", func() {
					obj.SetNamespace("garden")
					Expect(run(obj)).To(BeTrue())
				})
			}

			tests(func(obj client.Object) interface{} { return pred.Create(event.CreateEvent{Object: obj}) })
			tests(func(obj client.Object) interface{} { return pred.Update(event.UpdateEvent{ObjectNew: obj}) })
			tests(func(obj client.Object) interface{} { return pred.Delete(event.DeleteEvent{Object: obj}) })
			tests(func(obj client.Object) interface{} { return pred.Generic(event.GenericEvent{Object: obj}) })
		})
	})

	Describe("#ShootNotFailedPredicate", func() {
		var (
			ctx = context.TODO()

			ctrl  *gomock.Controller
			cache *mockcache.MockCache

			pred     predicate.Predicate
			injector = func(predicate interface{}) {
				Expect(inject.InjectorInto(func(into interface{}) error {
					Expect(inject.StopChannelInto(ctx.Done(), into)).To(BeTrue())
					Expect(inject.CacheInto(cache, into)).To(BeTrue())
					return nil
				}, predicate)).To(BeTrue())
			}

			obj       *extensionsv1alpha1.Infrastructure
			namespace = "shoot--foo--bar"
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			cache = mockcache.NewMockCache(ctrl)

			pred = ShootNotFailedPredicate()
			injector(pred)

			obj = &extensionsv1alpha1.Infrastructure{ObjectMeta: metav1.ObjectMeta{Namespace: namespace}}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		Describe("#Create, #Update", func() {
			tests := func(run func() interface{}) {
				It("should return true because shoot has no last operation", func() {
					cache.EXPECT().Get(gomock.Any(), kutil.Key(namespace), gomock.AssignableToTypeOf(&extensionsv1alpha1.Cluster{})).DoAndReturn(func(_ context.Context, _ client.ObjectKey, actual *extensionsv1alpha1.Cluster) error {
						computeClusterWithShoot(
							namespace,
							nil,
							nil,
							&gardencorev1beta1.ShootStatus{},
						).DeepCopyInto(actual)
						return nil
					})

					Expect(run()).To(BeTrue())
				})

				It("should return true because shoot last operation state is not failed", func() {
					cache.EXPECT().Get(gomock.Any(), kutil.Key(namespace), gomock.AssignableToTypeOf(&extensionsv1alpha1.Cluster{})).DoAndReturn(func(_ context.Context, _ client.ObjectKey, actual *extensionsv1alpha1.Cluster) error {
						computeClusterWithShoot(
							namespace,
							nil,
							nil,
							&gardencorev1beta1.ShootStatus{
								LastOperation: &gardencorev1beta1.LastOperation{},
							},
						).DeepCopyInto(actual)
						return nil
					})

					Expect(run()).To(BeTrue())
				})

				It("should return false because shoot is failed", func() {
					cache.EXPECT().Get(gomock.Any(), kutil.Key(namespace), gomock.AssignableToTypeOf(&extensionsv1alpha1.Cluster{})).DoAndReturn(func(_ context.Context, _ client.ObjectKey, actual *extensionsv1alpha1.Cluster) error {
						computeClusterWithShoot(
							namespace,
							nil,
							nil,
							&gardencorev1beta1.ShootStatus{
								LastOperation: &gardencorev1beta1.LastOperation{
									State: gardencorev1beta1.LastOperationStateFailed,
								},
							},
						).DeepCopyInto(actual)
						return nil
					})

					Expect(run()).To(BeFalse())
				})
			}

			tests(func() interface{} { return pred.Create(event.CreateEvent{Object: obj}) })
			tests(func() interface{} { return pred.Update(event.UpdateEvent{ObjectNew: obj}) })
		})

		Describe("#Delete", func() {
			It("should return false", func() {
				Expect(pred.Delete(event.DeleteEvent{})).To(BeFalse())
			})
		})

		Describe("#Generic", func() {
			It("should return false", func() {
				Expect(pred.Generic(event.GenericEvent{})).To(BeFalse())
			})
		})
	})
})
