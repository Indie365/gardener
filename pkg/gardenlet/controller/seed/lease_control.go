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

package seed

import (
	"context"
	"fmt"
	"net/http"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/gardenlet/apis/config"
	"github.com/gardener/gardener/pkg/healthz"

	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const leaseReconcilerName = "lease"

func (c *Controller) seedLeaseAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	c.seedLeaseQueue.Add(key)
}

type leaseReconciler struct {
	gardenClient  client.Client
	seedClientSet kubernetes.Interface
	healthManager healthz.Manager
	nowFunc       func() metav1.Time
	config        *config.GardenletConfiguration
}

// NewLeaseReconciler creates a new reconciler that periodically renews the gardenlet's lease.
func NewLeaseReconciler(
	gardenClient client.Client,
	seedClientSet kubernetes.Interface,
	healthManager healthz.Manager,
	nowFunc func() metav1.Time,
	config *config.GardenletConfiguration,
) reconcile.Reconciler {
	return &leaseReconciler{
		gardenClient:  gardenClient,
		seedClientSet: seedClientSet,
		nowFunc:       nowFunc,
		healthManager: healthManager,
		config:        config,
	}
}

func (r *leaseReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := logf.FromContext(ctx)

	seed := &gardencorev1beta1.Seed{}
	if err := r.gardenClient.Get(ctx, request.NamespacedName, seed); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("Object is gone, stop reconciling")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("error retrieving object from store: %w", err)
	}

	if err := r.reconcile(ctx, seed); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: time.Duration(*r.config.Controllers.Seed.LeaseResyncSeconds) * time.Second}, nil
}

func (r *leaseReconciler) reconcile(ctx context.Context, seed *gardencorev1beta1.Seed) error {
	if err := r.checkSeedConnection(ctx, seed); err != nil {
		r.healthManager.Set(false)
		return fmt.Errorf("cannot establish connection with Seed: %w", err)
	}

	if err := r.renewLeaseForSeed(ctx, seed); err != nil {
		r.healthManager.Set(false)
		return err
	}

	r.healthManager.Set(true)

	bldr, err := helper.NewConditionBuilder(gardencorev1beta1.SeedGardenletReady)
	if err != nil {
		return err
	}

	condition := helper.GetCondition(seed.Status.Conditions, gardencorev1beta1.SeedGardenletReady)
	if condition != nil {
		bldr.WithOldCondition(*condition)
	}

	bldr.WithStatus(gardencorev1beta1.ConditionTrue)
	bldr.WithReason("GardenletReady")
	bldr.WithMessage("Gardenlet is posting ready status.")

	newCondition, needsUpdate := bldr.WithNowFunc(r.nowFunc).Build()
	if !needsUpdate {
		return nil
	}

	// patch GardenletReady condition
	patch := client.StrategicMergeFrom(seed.DeepCopy())
	seed.Status.Conditions = helper.MergeConditions(seed.Status.Conditions, newCondition)
	return r.gardenClient.Status().Patch(ctx, seed, patch)
}

func (r *leaseReconciler) checkSeedConnection(ctx context.Context, seed *gardencorev1beta1.Seed) error {
	result := r.seedClientSet.RESTClient().Get().AbsPath("/healthz").Do(ctx)
	if result.Error() != nil {
		return fmt.Errorf("failed to execute call to Kubernetes API Server: %v", result.Error())
	}

	var statusCode int
	result.StatusCode(&statusCode)
	if statusCode != http.StatusOK {
		return fmt.Errorf("API Server returned unexpected status code: %d", statusCode)
	}

	return nil
}

func (r *leaseReconciler) renewLeaseForSeed(ctx context.Context, seed *gardencorev1beta1.Seed) error {
	var (
		holderIdentity = seed.Name
		ownerReference = metav1.OwnerReference{
			APIVersion: gardencorev1beta1.SchemeGroupVersion.String(),
			Kind:       "Seed",
			Name:       seed.GetName(),
			UID:        seed.GetUID(),
		}
	)

	lease := &coordinationv1.Lease{}
	if err := r.gardenClient.Get(ctx, client.ObjectKey{Name: holderIdentity, Namespace: gardencorev1beta1.GardenerSeedLeaseNamespace}, lease); err != nil {
		if apierrors.IsNotFound(err) {
			return r.gardenClient.Create(ctx, r.newOrRenewedLease(nil, holderIdentity, ownerReference))
		}
		return err
	}

	return r.gardenClient.Update(ctx, r.newOrRenewedLease(lease, holderIdentity, ownerReference))
}

func (r *leaseReconciler) newOrRenewedLease(lease *coordinationv1.Lease, holderIdentity string, ownerReference metav1.OwnerReference) *coordinationv1.Lease {
	if lease == nil {
		lease = &coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      holderIdentity,
				Namespace: gardencorev1beta1.GardenerSeedLeaseNamespace,
			},
		}
	}

	lease.OwnerReferences = []metav1.OwnerReference{ownerReference}
	lease.Spec = coordinationv1.LeaseSpec{
		HolderIdentity:       pointer.String(holderIdentity),
		LeaseDurationSeconds: r.config.Controllers.Seed.LeaseResyncSeconds,
		RenewTime:            &metav1.MicroTime{Time: r.nowFunc().Time},
	}
	return lease
}
