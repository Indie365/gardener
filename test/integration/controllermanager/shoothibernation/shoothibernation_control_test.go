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

package shoot_test

import (
	"context"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/controllermanager/apis/config"
	"github.com/gardener/gardener/pkg/controllermanager/controller/shoot"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var _ = Describe("Shoot hibernation controller tests", func() {

	var (
		ctx = context.Background()

		namespace *corev1.Namespace
		shoot     *gardencorev1beta1.Shoot
	)

	BeforeEach(func() {
		By("create shoot namespace")
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "garden-dev"},
		}
		Expect(testClient.Create(ctx, namespace)).To(Or(Succeed(), BeAlreadyExistsError()))

		By("create shoot")
		shoot = &gardencorev1beta1.Shoot{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "garden-dev"},
			Spec: gardencorev1beta1.ShootSpec{
				SecretBindingName: "my-provider-account",
				CloudProfileName:  "cloudprofile1",
				Region:            "europe-central-1",
				Provider: gardencorev1beta1.Provider{
					Type: "foo-provider",
					Workers: []gardencorev1beta1.Worker{
						{
							Name:    "cpu-worker",
							Minimum: 3,
							Maximum: 3,
							Machine: gardencorev1beta1.Machine{
								Type: "large",
							},
						},
					},
				},
				Kubernetes: gardencorev1beta1.Kubernetes{
					Version: "1.20.1",
				},
				Networking: gardencorev1beta1.Networking{
					Type: "foo-networking",
				},
			},
		}

		Expect(testClient.Create(ctx, shoot)).To(Or(Succeed(), BeAlreadyExistsError()))
	})

	It("should successfully hibernate the shoot based on schedule", func() {
		patch := client.MergeFrom(shoot.DeepCopy())
		shoot.Spec.Hibernation = &gardencorev1beta1.Hibernation{
			Schedules: []gardencorev1beta1.HibernationSchedule{
				{
					Start: pointer.String("*/1 * * * *"),
					End:   nil,
				},
			},
		}
		Expect(testClient.Patch(ctx, shoot, patch)).To(Succeed())

		Eventually(func(g Gomega) {
			g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(shoot), shoot)).To(Succeed())
			g.Expect(shoot.Spec.Hibernation.Enabled).To(PointTo(Equal(true)))
		}, 90*time.Second).Should(Succeed())
	})

	It("should successfully wakeup the shoot based on schedule", func() {
		patch := client.MergeFrom(shoot.DeepCopy())
		shoot.Spec.Hibernation = &gardencorev1beta1.Hibernation{
			Schedules: []gardencorev1beta1.HibernationSchedule{
				{
					Start: nil,
					End:   pointer.String("*/1 * * * *"),
				},
			},
		}
		Expect(testClient.Patch(ctx, shoot, patch)).To(Succeed())

		Eventually(func(g Gomega) {
			g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(shoot), shoot)).To(Succeed())
			g.Expect(shoot.Spec.Hibernation.Enabled).To(PointTo(Equal(false)))
		}, 90*time.Second).Should(Succeed())
	})
})

func addShootHibernationControllerToManager(mgr manager.Manager) error {
	recorder := mgr.GetEventRecorderFor("shoot-hibernation-controller")
	c, err := controller.New(
		"shoot-hibernation",
		mgr,
		controller.Options{
			Reconciler: shoot.NewShootHibernationReconciler(testClient, config.ShootHibernationControllerConfiguration{TriggerDeadlineDuration: &metav1.Duration{Duration: time.Minute}}, recorder, clock.RealClock{}),
		},
	)
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &gardencorev1beta1.Shoot{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}
