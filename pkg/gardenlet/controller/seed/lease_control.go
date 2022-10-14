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
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/gardener/gardener/pkg/gardenlet/apis/config"
	"github.com/gardener/gardener/pkg/healthz"

	"github.com/go-logr/logr"
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/clock"
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
	gardenClient   client.Client
	seedClientSet  kubernetes.Interface
	healthManager  healthz.Manager
	clock          clock.Clock
	config         *config.GardenletConfiguration
	leaseNamespace string
}

// NewLeaseReconciler creates a new reconciler that periodically renews the gardenlet's lease.
func NewLeaseReconciler(
	gardenClient client.Client,
	seedClientSet kubernetes.Interface,
	healthManager healthz.Manager,
	clock clock.Clock,
	config *config.GardenletConfiguration,
	leaseNamespace string,
) reconcile.Reconciler {
	return &leaseReconciler{
		gardenClient:   gardenClient,
		seedClientSet:  seedClientSet,
		clock:          clock,
		healthManager:  healthManager,
		config:         config,
		leaseNamespace: leaseNamespace,
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

	if err := r.reconcile(ctx, log, seed); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: time.Duration(*r.config.Controllers.Seed.LeaseResyncSeconds) * time.Second}, nil
}

func (r *leaseReconciler) reconcile(ctx context.Context, log logr.Logger, seed *gardencorev1beta1.Seed) error {
	if err := CheckSeedConnection(ctx, r.seedClientSet.RESTClient()); err != nil {
		log.Info("Set health status to false")
		r.healthManager.Set(false)
		return fmt.Errorf("cannot establish connection with Seed: %w", err)
	}

	if err := r.renewLeaseForSeed(ctx, seed); err != nil {
		r.healthManager.Set(false)
		return err
	}

	log.Info("Set health status to true and renew Lease")
	r.healthManager.Set(true)
	return r.maintainGardenletReadyCondition(ctx, seed)
}

// CheckSeedConnection is a function which checks the connection to the seed.
// Exposed for testing.
var CheckSeedConnection = func(ctx context.Context, client rest.Interface) error {
	result := client.Get().AbsPath("/healthz").Do(ctx)
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
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      seed.Name,
			Namespace: r.leaseNamespace,
		},
	}

	_, err := controllerutils.CreateOrGetAndMergePatch(ctx, r.gardenClient, lease, func() error {
		lease.OwnerReferences = []metav1.OwnerReference{{
			APIVersion: gardencorev1beta1.SchemeGroupVersion.String(),
			Kind:       "Seed",
			Name:       seed.GetName(),
			UID:        seed.GetUID(),
		}}
		lease.Spec.HolderIdentity = pointer.String(seed.Name)
		lease.Spec.LeaseDurationSeconds = r.config.Controllers.Seed.LeaseResyncSeconds
		lease.Spec.RenewTime = &metav1.MicroTime{Time: r.clock.Now()}
		return nil
	})
	return err
}

func (r *leaseReconciler) maintainGardenletReadyCondition(ctx context.Context, seed *gardencorev1beta1.Seed) error {
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

	newCondition, needsUpdate := bldr.WithClock(r.clock).Build()
	if !needsUpdate {
		return nil
	}

	patch := client.StrategicMergeFrom(seed.DeepCopy())
	seed.Status.Conditions = helper.MergeConditions(seed.Status.Conditions, newCondition)
	return r.gardenClient.Status().Patch(ctx, seed, patch)
}
