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

package managedseedset

import (
	"context"
	"fmt"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	seedmanagementv1alpha1 "github.com/gardener/gardener/pkg/apis/seedmanagement/v1alpha1"
	"github.com/gardener/gardener/pkg/controllermanager/apis/config"
	"github.com/gardener/gardener/pkg/controllerutils"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/go-logr/logr"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconciler implements the reconcile.Reconciler interface for ManagedSeedSet reconciliation.
type reconciler struct {
	gardenClient client.Client
	actuator     Actuator
	cfg          *config.ManagedSeedSetControllerConfiguration
	logger       logr.Logger
}

// NewReconciler creates and returns a new ManagedSeedSet reconciler with the given parameters.
func NewReconciler(
	gardenClient client.Client,
	actuator Actuator,
	cfg *config.ManagedSeedSetControllerConfiguration,
	logger logr.Logger,
) *reconciler {
	return &reconciler{
		gardenClient: gardenClient,
		actuator:     actuator,
		cfg:          cfg,
		logger:       logger,
	}
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := r.logger.WithValues("managedseedset", request)

	set := &seedmanagementv1alpha1.ManagedSeedSet{}
	if err := r.gardenClient.Get(ctx, request.NamespacedName, set); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Object is gone, stop reconciling")
			return reconcile.Result{}, nil
		}

		logger.Error(err, "Unable to retrieve object from store")
		return reconcile.Result{}, err
	}

	if set.DeletionTimestamp != nil {
		return r.delete(ctx, set, logger)
	}
	return r.reconcile(ctx, set, logger)
}

func (r *reconciler) reconcile(ctx context.Context, set *seedmanagementv1alpha1.ManagedSeedSet, logger logr.Logger) (result reconcile.Result, err error) {
	// Ensure gardener finalizer
	logger.Info("Ensuring finalizer")
	if err := controllerutils.PatchAddFinalizers(ctx, r.gardenClient, set, gardencorev1beta1.GardenerName); err != nil {
		return reconcile.Result{}, fmt.Errorf("could not ensure gardener finalizer: %w", err)
	}

	var status *seedmanagementv1alpha1.ManagedSeedSetStatus
	defer func() {
		// Update status, on failure return the update error unless there is another error
		if updateErr := r.updateStatus(ctx, set, status); updateErr != nil && err == nil {
			err = fmt.Errorf("could not update status: %w", updateErr)
		}
	}()

	// Reconcile creation or update
	logger.Info("Reconciling creation or update")
	if status, _, err = r.actuator.Reconcile(ctx, set); err != nil {
		return reconcile.Result{}, fmt.Errorf("could not reconcile ManagedSeedSet %s creation or update: %w", kutil.ObjectName(set), err)
	}
	logger.Info("Creation or update reconciled")

	// Return success result
	return reconcile.Result{RequeueAfter: r.cfg.SyncPeriod.Duration}, nil
}

func (r *reconciler) delete(ctx context.Context, set *seedmanagementv1alpha1.ManagedSeedSet, logger logr.Logger) (result reconcile.Result, err error) {
	// Check gardener finalizer
	if !controllerutil.ContainsFinalizer(set, gardencorev1beta1.GardenerName) {
		logger.Info("Skipping as it does not have a finalizer")
		return reconcile.Result{}, nil
	}

	var status *seedmanagementv1alpha1.ManagedSeedSetStatus
	var removeFinalizer bool
	defer func() {
		// Only update status if the finalizer is not removed to prevent errors if the object is already gone
		if !removeFinalizer {
			// Update status, on failure return the update error unless there is another error
			if updateErr := r.updateStatus(ctx, set, status); updateErr != nil && err == nil {
				err = fmt.Errorf("could not update status: %w", updateErr)
			}
		}
	}()

	// Reconcile deletion
	logger.Info("Reconciling deletion")
	if status, removeFinalizer, err = r.actuator.Reconcile(ctx, set); err != nil {
		return reconcile.Result{}, fmt.Errorf("could not reconcile ManagedSeedSet %s deletion: %w", kutil.ObjectName(set), err)
	}
	logger.Info("Deletion reconciled")

	// Remove gardener finalizer if requested by the actuator
	if removeFinalizer {
		logger.Info("Removing finalizer")
		if err := controllerutils.PatchRemoveFinalizers(ctx, r.gardenClient, set, gardencorev1beta1.GardenerName); err != nil {
			return reconcile.Result{}, fmt.Errorf("could not remove gardener finalizer: %w", err)
		}
		return reconcile.Result{}, nil
	}

	// Return success result
	return reconcile.Result{RequeueAfter: r.cfg.SyncPeriod.Duration}, nil
}

func (r *reconciler) updateStatus(ctx context.Context, set *seedmanagementv1alpha1.ManagedSeedSet, status *seedmanagementv1alpha1.ManagedSeedSetStatus) error {
	if status == nil {
		return nil
	}
	patch := client.StrategicMergeFrom(set.DeepCopy())
	set.Status = *status
	return r.gardenClient.Status().Patch(ctx, set, patch)
}
