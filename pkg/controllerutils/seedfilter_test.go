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

package controllerutils_test

import (
	"context"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	seedmanagementv1alpha1 "github.com/gardener/gardener/pkg/apis/seedmanagement/v1alpha1"
	"github.com/gardener/gardener/pkg/controllerutils"
	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	name      = "test"
	namespace = "garden"
	seedName  = "test-seed"
)

var _ = Describe("secretref", func() {
	var (
		ctrl *gomock.Controller
		c    *mockclient.MockClient

		ctx context.Context

		managedSeed *seedmanagementv1alpha1.ManagedSeed
		shoot       *gardencorev1beta1.Shoot
		seed        *gardencorev1beta1.Seed
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		c = mockclient.NewMockClient(ctrl)

		ctx = context.TODO()

		managedSeed = &seedmanagementv1alpha1.ManagedSeed{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: seedmanagementv1alpha1.ManagedSeedSpec{
				Shoot: seedmanagementv1alpha1.Shoot{
					Name: name,
				},
			},
		}
		shoot = &gardencorev1beta1.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: gardencorev1beta1.ShootSpec{
				SeedName: pointer.StringPtr(seedName),
			},
			Status: gardencorev1beta1.ShootStatus{
				SeedName: pointer.StringPtr(seedName),
			},
		}
		seed = &gardencorev1beta1.Seed{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				Labels: map[string]string{
					"test-label": "test",
				},
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	var (
		expectGetShoot = func() {
			c.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&gardencorev1beta1.Shoot{})).DoAndReturn(
				func(_ context.Context, _ client.ObjectKey, s *gardencorev1beta1.Shoot) error {
					*s = *shoot
					return nil
				},
			)
		}

		expectGetShootNotFound = func() {
			c.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&gardencorev1beta1.Shoot{})).DoAndReturn(
				func(_ context.Context, _ client.ObjectKey, _ *gardencorev1beta1.Shoot) error {
					return apierrors.NewNotFound(gardencorev1beta1.Resource("shoot"), name)
				},
			)
		}

		expectGetSeed = func() {
			c.EXPECT().Get(ctx, kutil.Key(seedName), gomock.AssignableToTypeOf(&gardencorev1beta1.Seed{})).DoAndReturn(
				func(_ context.Context, _ client.ObjectKey, s *gardencorev1beta1.Seed) error {
					*s = *seed
					return nil
				},
			)
		}
	)

	Describe("#ManagedSeedFilterFunc", func() {
		It("should return false if the specified object is not a managed seed", func() {
			f := controllerutils.ManagedSeedFilterFunc(ctx, c, seedName, nil)
			Expect(f(shoot)).To(BeFalse())
		})

		It("should return false with a shoot that is not found", func() {
			expectGetShootNotFound()
			f := controllerutils.ManagedSeedFilterFunc(ctx, c, seedName, nil)
			Expect(f(managedSeed)).To(BeFalse())
		})

		It("should return false with a shoot that is not yet scheduled on a seed", func() {
			shoot.Spec.SeedName = nil
			expectGetShoot()
			f := controllerutils.ManagedSeedFilterFunc(ctx, c, seedName, nil)
			Expect(f(managedSeed)).To(BeFalse())
		})

		It("should return true with a shoot that is scheduled on the specified seed", func() {
			expectGetShoot()
			f := controllerutils.ManagedSeedFilterFunc(ctx, c, seedName, nil)
			Expect(f(managedSeed)).To(BeTrue())
		})

		It("should return true with a shoot that is scheduled on the specified seed (status different from spec)", func() {
			shoot.Spec.SeedName = pointer.StringPtr("foo")
			expectGetShoot()
			f := controllerutils.ManagedSeedFilterFunc(ctx, c, seedName, nil)
			Expect(f(managedSeed)).To(BeTrue())
		})

		It("should return false with a shoot that is scheduled on a different seed", func() {
			expectGetShoot()
			f := controllerutils.ManagedSeedFilterFunc(ctx, c, "foo", nil)
			Expect(f(managedSeed)).To(BeFalse())
		})

		It("should return true with a shoot that is scheduled on a seed selected by the specified label selector", func() {
			expectGetShoot()
			expectGetSeed()
			f := controllerutils.ManagedSeedFilterFunc(ctx, c, "", &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"test-label": "test",
				},
			})
			Expect(f(managedSeed)).To(BeTrue())
		})

		It("should return true with a shoot that is scheduled on a seed selected by the specified label selector (status different from spec)", func() {
			shoot.Spec.SeedName = pointer.StringPtr("foo")
			expectGetShoot()
			expectGetSeed()
			f := controllerutils.ManagedSeedFilterFunc(ctx, c, "", &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"test-label": "test",
				},
			})
			Expect(f(managedSeed)).To(BeTrue())
		})

		It("should return false with a shoot that is scheduled on a seed not selected by the specified label selector", func() {
			expectGetShoot()
			expectGetSeed()
			f := controllerutils.ManagedSeedFilterFunc(ctx, c, "", &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"foo": "bar",
				},
			})
			Expect(f(managedSeed)).To(BeFalse())
		})
	})
})
