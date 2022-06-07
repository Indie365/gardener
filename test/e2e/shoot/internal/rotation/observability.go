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

package rotation

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	"github.com/gardener/gardener/test/framework"
)

// ObservabilityVerifier verifies the observability credentials rotation.
type ObservabilityVerifier struct {
	*framework.ShootCreationFramework

	oldKeypairData map[string][]byte
}

// Before is called before the rotation is started.
func (v *ObservabilityVerifier) Before(ctx context.Context) {
	By("Verify old observability secret")
	Eventually(func(g Gomega) {
		secret := &corev1.Secret{}
		g.Expect(v.GardenClient.Client().Get(ctx, client.ObjectKey{Namespace: v.Shoot.Namespace, Name: gutil.ComputeShootProjectSecretName(v.Shoot.Name, "monitoring")}, secret)).To(Succeed())
		g.Expect(secret.Data).To(And(
			HaveKeyWithValue("username", Not(BeEmpty())),
			HaveKeyWithValue("password", Not(BeEmpty())),
		))
		v.oldKeypairData = secret.Data
	}).Should(Succeed(), "old observability secret should be present")
}

// ExpectPreparingStatus is called while waiting for the Preparing status.
func (v *ObservabilityVerifier) ExpectPreparingStatus(g Gomega) {
	g.Expect(time.Now().UTC().Sub(v.Shoot.Status.Credentials.Rotation.Observability.LastInitiationTime.Time.UTC())).To(BeNumerically("<=", time.Minute))
}

// AfterPrepared is called when the Shoot is in Prepared status.
func (v *ObservabilityVerifier) AfterPrepared(ctx context.Context) {
	observabilityRotation := v.Shoot.Status.Credentials.Rotation.Observability
	Expect(observabilityRotation.LastCompletionTime.Time.UTC().After(observabilityRotation.LastInitiationTime.Time.UTC())).To(BeTrue())

	By("Verify new observability secret")
	Eventually(func(g Gomega) {
		secret := &corev1.Secret{}
		g.Expect(v.GardenClient.Client().Get(ctx, client.ObjectKey{Namespace: v.Shoot.Namespace, Name: gutil.ComputeShootProjectSecretName(v.Shoot.Name, "monitoring")}, secret)).To(Succeed())
		g.Expect(secret.Data).To(And(
			HaveKeyWithValue("username", Equal(v.oldKeypairData["username"])),
			HaveKeyWithValue("password", Not(Equal(v.oldKeypairData["password"]))),
		))
	}).Should(Succeed(), "observability secret should have been rotated")
}

// observability credentials rotation is completed after one reconciliation (there is no second phase)
// hence, there is nothing to check in the second part of the credentials rotation

// ExpectCompletingStatus is called while waiting for the Completing status.
func (v *ObservabilityVerifier) ExpectCompletingStatus(g Gomega) {}

// AfterCompleted is called when the Shoot is in Completed status.
func (v *ObservabilityVerifier) AfterCompleted(ctx context.Context) {}
