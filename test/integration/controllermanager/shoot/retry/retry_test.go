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

package retry_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/controllermanager/apis/config"
	shootcontroller "github.com/gardener/gardener/pkg/controllermanager/controller/shoot"
	retryutils "github.com/gardener/gardener/pkg/utils/retry"
)

const retryPeriod = 10 * time.Second

var _ = Describe("Shoot retry controller tests", func() {
	var shoot *gardencorev1beta1.Shoot

	BeforeEach(func() {
		shoot = &gardencorev1beta1.Shoot{
			ObjectMeta: metav1.ObjectMeta{GenerateName: "test-", Namespace: testNamespace.Name},
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

		By("Create Shoot")
		Expect(testClient.Create(ctx, shoot)).To(Succeed())
		log.Info("Created shoot for test", "shoot", client.ObjectKeyFromObject(shoot))

		DeferCleanup(func() {
			By("Delete Shoot")
			Expect(client.IgnoreNotFound(testClient.Delete(ctx, shoot))).To(Succeed())
		})
	})

	It("should successfully retry a failed Shoot with rate limits exceeded error", func() {
		By("mark the Shoot as failed with rate limits exceeded error code")
		shootCopy := shoot.DeepCopy()
		shoot.Status = gardencorev1beta1.ShootStatus{
			LastOperation: &gardencorev1beta1.LastOperation{
				State:          gardencorev1beta1.LastOperationStateFailed,
				LastUpdateTime: metav1.Time{Time: time.Now().Add(time.Minute * -1)},
			},
			LastErrors: []gardencorev1beta1.LastError{
				{
					Codes: []gardencorev1beta1.ErrorCode{gardencorev1beta1.ErrorInfraRateLimitsExceeded},
				},
			},
			ObservedGeneration: 1,
		}
		Expect(testClient.Status().Patch(ctx, shoot, client.MergeFrom(shootCopy))).To(Succeed())

		By("verify shoot is retried")
		err := waitForShootGenerationToBeIncreased(ctx, testClient, client.ObjectKeyFromObject(shoot), 1)
		Expect(err).NotTo(HaveOccurred())
	})
})

func waitForShootGenerationToBeIncreased(ctx context.Context, c client.Client, objectKey client.ObjectKey, observedGeneration int64) error {
	return retryutils.UntilTimeout(ctx, time.Second, time.Minute, func(ctx context.Context) (bool, error) {
		shoot := &gardencorev1beta1.Shoot{}
		if err := c.Get(ctx, objectKey, shoot); err != nil {
			return retryutils.SevereError(err)
		}

		if shoot.Generation == observedGeneration+1 {
			return retryutils.Ok()
		}

		return retryutils.MinorError(fmt.Errorf("waiting for shoot generation to be increased (metadata.generation=%d, status.observedGeneration=%d)", shoot.Generation, shoot.Status.ObservedGeneration))
	})
}

func addShootRetryControllerToManager(mgr manager.Manager) error {
	c, err := controller.New(
		"shoot-retry-controller",
		mgr,
		controller.Options{
			Reconciler: shootcontroller.NewShootRetryReconciler(testClient, &config.ShootRetryControllerConfiguration{RetryPeriod: &metav1.Duration{Duration: retryPeriod}}),
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
