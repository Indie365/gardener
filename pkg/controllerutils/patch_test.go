// Copyright 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package controllerutils

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	corescheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"
	"github.com/gardener/gardener/pkg/utils/test"
)

var _ = Describe("Patch", func() {
	var (
		ctx     = context.TODO()
		fakeErr = fmt.Errorf("fake err")

		ctrl   *gomock.Controller
		c      *mockclient.MockClient
		scheme *runtime.Scheme
		obj    *corev1.ServiceAccount
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		c = mockclient.NewMockClient(ctrl)
		obj = &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
			Name:            "foo",
			Namespace:       "bar",
			ResourceVersion: "42",
		}}

		scheme = runtime.NewScheme()
		Expect(corescheme.AddToScheme(scheme)).NotTo(HaveOccurred())

		c.EXPECT().Scheme().Return(scheme).AnyTimes()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("GetAndCreateOr*Patch", func() {
		testSuite := func(f func(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn, opts ...PatchOption) (controllerutil.OperationResult, error), patchType types.PatchType) {
			It("should return an error because reading the object fails", func() {
				c.EXPECT().Get(ctx, client.ObjectKeyFromObject(obj), obj).Return(fakeErr)

				result, err := f(ctx, c, obj, func() error { return nil })
				Expect(result).To(Equal(controllerutil.OperationResultNone))
				Expect(err).To(MatchError(fakeErr))
			})

			It("should return an error because the mutate function returned an error (object found)", func() {
				c.EXPECT().Get(ctx, client.ObjectKeyFromObject(obj), obj)

				result, err := f(ctx, c, obj, func() error { return fakeErr })
				Expect(result).To(Equal(controllerutil.OperationResultNone))
				Expect(err).To(MatchError(fakeErr))
			})

			It("should return an error because the mutate function returned an error (object not found)", func() {
				c.EXPECT().Get(ctx, client.ObjectKeyFromObject(obj), obj).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))

				result, err := f(ctx, c, obj, func() error { return fakeErr })
				Expect(result).To(Equal(controllerutil.OperationResultNone))
				Expect(err).To(MatchError(fakeErr))
			})

			It("should return an error because the create failed", func() {
				gomock.InOrder(
					c.EXPECT().Get(ctx, client.ObjectKeyFromObject(obj), obj).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
					c.EXPECT().Create(ctx, obj).Return(fakeErr),
				)

				result, err := f(ctx, c, obj, func() error { return nil })
				Expect(result).To(Equal(controllerutil.OperationResultNone))
				Expect(err).To(MatchError(fakeErr))
			})

			It("should successfully create the object", func() {
				gomock.InOrder(
					c.EXPECT().Get(ctx, client.ObjectKeyFromObject(obj), obj).Return(apierrors.NewNotFound(schema.GroupResource{}, "")),
					c.EXPECT().Create(ctx, obj),
				)

				result, err := f(ctx, c, obj, func() error { return nil })
				Expect(result).To(Equal(controllerutil.OperationResultCreated))
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return an error because the patch failed", func() {
				gomock.InOrder(
					c.EXPECT().Get(ctx, client.ObjectKeyFromObject(obj), obj),
					test.EXPECTPatch(ctx, c, obj, obj, patchType).Return(fakeErr),
				)

				result, err := f(ctx, c, obj, func() error { return nil })
				Expect(result).To(Equal(controllerutil.OperationResultNone))
				Expect(err).To(MatchError(fakeErr))
			})

			It("should successfully patch the object", func() {
				objCopy := obj.DeepCopy()
				mutateFn := func(o *corev1.ServiceAccount) func() error {
					return func() error {
						o.Labels = map[string]string{"foo": "bar"}
						return nil
					}
				}
				_ = mutateFn(objCopy)()

				gomock.InOrder(
					c.EXPECT().Get(ctx, client.ObjectKeyFromObject(obj), obj),
					test.EXPECTPatch(ctx, c, objCopy, obj, patchType),
				)

				result, err := f(ctx, c, obj, mutateFn(obj))
				Expect(result).To(Equal(controllerutil.OperationResultUpdated))
				Expect(err).NotTo(HaveOccurred())
			})

			It("should successfully patch the object with optimistic locking", func() {
				objCopy := obj.DeepCopy()
				mutateFn := func(o *corev1.ServiceAccount) func() error {
					return func() error {
						o.Labels = map[string]string{"foo": "bar"}
						return nil
					}
				}
				_ = mutateFn(objCopy)()

				gomock.InOrder(
					c.EXPECT().Get(ctx, client.ObjectKeyFromObject(obj), obj),
					test.EXPECTPatchWithOptimisticLock(ctx, c, objCopy, obj, patchType),
				)

				result, err := f(ctx, c, obj, mutateFn(obj), MergeFromOption{client.MergeFromWithOptimisticLock{}})
				Expect(result).To(Equal(controllerutil.OperationResultUpdated))
				Expect(err).NotTo(HaveOccurred())
			})

			It("should skip sending an empty patch", func() {
				objCopy := obj.DeepCopy()
				mutateFn := func(o *corev1.ServiceAccount) func() error {
					return func() error {
						return nil
					}
				}
				_ = mutateFn(objCopy)()

				gomock.InOrder(
					c.EXPECT().Get(ctx, client.ObjectKeyFromObject(obj), obj),
				)

				result, err := f(ctx, c, obj, mutateFn(obj), SkipEmptyPatch{})
				Expect(result).To(Equal(controllerutil.OperationResultNone))
				Expect(err).NotTo(HaveOccurred())
			})
		}

		Describe("#GetAndCreateOrMergePatch", func() { testSuite(GetAndCreateOrMergePatch, types.MergePatchType) })
		Describe("#GetAndCreateOrStrategicMergePatch", func() { testSuite(GetAndCreateOrStrategicMergePatch, types.StrategicMergePatchType) })
	})

	Describe("CreateOrGetAnd*Patch", func() {
		testSuite := func(f func(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn, opts ...client.MergeFromOption) (controllerutil.OperationResult, error), patchType types.PatchType) {
			It("should return an error because the mutate function returned an error", func() {
				result, err := f(ctx, c, obj, func() error { return fakeErr })
				Expect(result).To(Equal(controllerutil.OperationResultNone))
				Expect(err).To(MatchError(fakeErr))
			})

			It("should return an error because the create failed", func() {
				c.EXPECT().Create(ctx, obj).Return(fakeErr)

				result, err := f(ctx, c, obj, func() error { return nil })
				Expect(result).To(Equal(controllerutil.OperationResultNone))
				Expect(err).To(MatchError(fakeErr))
			})

			It("should successfully create the object", func() {
				c.EXPECT().Create(ctx, obj)

				result, err := f(ctx, c, obj, func() error { return nil })
				Expect(result).To(Equal(controllerutil.OperationResultCreated))
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return an error because the get failed", func() {
				gomock.InOrder(
					c.EXPECT().Create(ctx, obj).Return(apierrors.NewAlreadyExists(schema.GroupResource{}, "")),
					c.EXPECT().Get(ctx, client.ObjectKeyFromObject(obj), obj).Return(fakeErr),
				)

				result, err := f(ctx, c, obj, func() error { return nil })
				Expect(result).To(Equal(controllerutil.OperationResultNone))
				Expect(err).To(MatchError(fakeErr))
			})

			It("should return an error because the patch failed", func() {
				gomock.InOrder(
					c.EXPECT().Create(ctx, obj).Return(apierrors.NewAlreadyExists(schema.GroupResource{}, "")),
					c.EXPECT().Get(ctx, client.ObjectKeyFromObject(obj), obj).DoAndReturn(func(_ context.Context, _ client.ObjectKey, objToReturn *corev1.ServiceAccount, _ ...client.GetOption) error {
						obj.DeepCopyInto(objToReturn)
						return nil
					}),
					test.EXPECTPatch(ctx, c, obj, obj, patchType, fakeErr),
				)

				result, err := f(ctx, c, obj, func() error { return nil })
				Expect(result).To(Equal(controllerutil.OperationResultNone))
				Expect(err).To(MatchError(fakeErr))
			})

			It("should successfully patch the object", func() {
				objCopy := obj.DeepCopy()
				mutateFn := func(o *corev1.ServiceAccount) func() error {
					return func() error {
						o.Labels = map[string]string{"foo": "bar"}
						return nil
					}
				}
				_ = mutateFn(objCopy)()

				gomock.InOrder(
					c.EXPECT().Create(ctx, obj).Return(apierrors.NewAlreadyExists(schema.GroupResource{}, "")),
					c.EXPECT().Get(ctx, client.ObjectKeyFromObject(obj), obj).DoAndReturn(func(_ context.Context, _ client.ObjectKey, objToReturn *corev1.ServiceAccount, _ ...client.GetOption) error {
						obj.DeepCopyInto(objToReturn)
						return nil
					}),
					test.EXPECTPatch(ctx, c, objCopy, obj, patchType),
				)

				result, err := f(ctx, c, obj, mutateFn(obj))
				Expect(result).To(Equal(controllerutil.OperationResultUpdated))
				Expect(err).NotTo(HaveOccurred())
			})

			It("should successfully patch the object with optimistic locking", func() {
				objCopy := obj.DeepCopy()
				mutateFn := func(o *corev1.ServiceAccount) func() error {
					return func() error {
						o.Labels = map[string]string{"foo": "bar"}
						return nil
					}
				}
				_ = mutateFn(objCopy)()

				gomock.InOrder(
					c.EXPECT().Create(ctx, obj).Return(apierrors.NewAlreadyExists(schema.GroupResource{}, "")),
					c.EXPECT().Get(ctx, client.ObjectKeyFromObject(obj), obj).DoAndReturn(func(_ context.Context, _ client.ObjectKey, objToReturn *corev1.ServiceAccount, _ ...client.GetOption) error {
						obj.DeepCopyInto(objToReturn)
						return nil
					}),
					test.EXPECTPatchWithOptimisticLock(ctx, c, objCopy, obj, patchType),
				)

				result, err := f(ctx, c, obj, mutateFn(obj), client.MergeFromWithOptimisticLock{})
				Expect(result).To(Equal(controllerutil.OperationResultUpdated))
				Expect(err).NotTo(HaveOccurred())
			})
		}

		Describe("#CreateOrGetAndMergePatch", func() { testSuite(CreateOrGetAndMergePatch, types.MergePatchType) })
		Describe("#CreateOrGetAndStrategicMergePatch", func() { testSuite(CreateOrGetAndStrategicMergePatch, types.StrategicMergePatchType) })
	})
})
