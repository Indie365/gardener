// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
	"time"

	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"github.com/gardener/gardener/pkg/controllermanager/apis/config"
	"github.com/gardener/gardener/pkg/utils/flow"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/go-logr/logr"

	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NewLifecycleDefaultControl returns a new instance of the default implementation that
// implements the documented semantics for checking the lifecycle for Seeds.
// You should use an instance returned from NewLifecycleDefaultControl() for any scenario other than testing.
func NewLifecycleDefaultControl(
	logger logr.Logger,
	gardenClient client.Client,
	config *config.SeedControllerConfiguration,
) *livecycleReconciler {
	return &livecycleReconciler{
		logger:       logger,
		gardenClient: gardenClient,
		config:       config,
	}
}

type livecycleReconciler struct {
	logger       logr.Logger
	gardenClient client.Client
	config       *config.SeedControllerConfiguration
}

func (r *livecycleReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	seed := &gardencorev1beta1.Seed{}
	logger := r.logger.WithValues("seed", req.Name)

	err := r.gardenClient.Get(ctx, req.NamespacedName, seed)
	if apierrors.IsNotFound(err) {
		logger.Info("[SEED LIFECYCLE] Stopping lifecycle operations for Seed since it has been deleted")
		return reconcile.Result{}, nil
	}
	if err != nil {
		logger.Error(err, "[SEED LIFECYCLE] Unable to retrieve object from store")
		return reconcile.Result{}, err
	}

	// New seeds don't have conditions - gardenlet never reported anything yet. Wait for grace period.
	if len(seed.Status.Conditions) == 0 {
		return reconcileAfter(10 * time.Second)
	}

	observedSeedLease := &coordinationv1.Lease{}
	if err := r.gardenClient.Get(ctx, kutil.Key(gardencorev1beta1.GardenerSeedLeaseNamespace, seed.Name), observedSeedLease); client.IgnoreNotFound(err) != nil {
		return reconcile.Result{}, err
	}

	if observedSeedLease != nil && observedSeedLease.Spec.RenewTime != nil {
		if observedSeedLease.Spec.RenewTime.UTC().After(time.Now().UTC().Add(-r.config.MonitorPeriod.Duration)) {
			return reconcileAfter(10 * time.Second)
		}

		// Get the latest Lease object in cases which the LeaseLister cache is outdated, to ensure that the lease is really expired
		latestLeaseObject := &coordinationv1.Lease{}
		if err := r.gardenClient.Get(ctx, kutil.Key(gardencorev1beta1.GardenerSeedLeaseNamespace, seed.Name), latestLeaseObject); err != nil {
			return reconcile.Result{}, err
		}

		if latestLeaseObject.Spec.RenewTime.UTC().After(time.Now().UTC().Add(-r.config.MonitorPeriod.Duration)) {
			return reconcileAfter(10 * time.Second)
		}
	}

	logger.Info("Setting status for seed to 'Unknown' as gardenlet stopped reporting seed status.")

	bldr, err := gardencorev1beta1helper.NewConditionBuilder(gardencorev1beta1.SeedGardenletReady)
	if err != nil {
		return reconcile.Result{}, err
	}

	conditionGardenletReady := gardencorev1beta1helper.GetCondition(seed.Status.Conditions, gardencorev1beta1.SeedGardenletReady)
	if conditionGardenletReady != nil {
		bldr.WithOldCondition(*conditionGardenletReady)
	}

	bldr.WithStatus(gardencorev1beta1.ConditionUnknown)
	bldr.WithReason("SeedStatusUnknown")
	bldr.WithMessage("Gardenlet stopped posting seed status.")
	if newCondition, update := bldr.WithNowFunc(metav1.Now).Build(); update {
		seed.Status.Conditions = gardencorev1beta1helper.MergeConditions(seed.Status.Conditions, newCondition)
		if err := r.gardenClient.Status().Update(ctx, seed); err != nil {
			return reconcile.Result{}, err
		}
	}

	// If the `GardenletReady` condition is `Unknown` for at least the configured `shootMonitorPeriod` then we will mark the conditions
	// and constraints for all the shoots that belong to this seed as `Unknown`. The reason is that the gardenlet didn't send a heartbeat
	// anymore, hence, it most likely didn't check the shoot status. This means that the current shoot status might not reflect the truth
	// anymore. We are indicating this by marking it as `Unknown`.
	if conditionGardenletReady != nil && !conditionGardenletReady.LastTransitionTime.UTC().Before(time.Now().UTC().Add(-r.config.ShootMonitorPeriod.Duration)) {
		return reconcileAfter(10 * time.Second)
	}

	logger.WithValues("gracePeriod", r.config.ShootMonitorPeriod.Duration).Info("Gardenlet didn't send a heartbeat during the grace period - setting the shoot conditions/constraints to 'unknown' for all shoots on this seed")

	shootList := &gardencorev1beta1.ShootList{}
	if err := r.gardenClient.List(ctx, shootList, client.MatchingFields{core.ShootSeedName: seed.Name}); err != nil {
		return reconcile.Result{}, err
	}

	var fns []flow.TaskFn

	for _, s := range shootList.Items {
		shoot := s
		fns = append(fns, func(ctx context.Context) error {
			return setShootStatusToUnknown(ctx, r.gardenClient, &shoot)
		})
	}

	if err := flow.Parallel(fns...)(ctx); err != nil {
		return reconcile.Result{}, err
	}

	return reconcileAfter(1 * time.Minute)
}

func setShootStatusToUnknown(ctx context.Context, c client.StatusClient, shoot *gardencorev1beta1.Shoot) error {
	var (
		reason = "StatusUnknown"
		msg    = "Gardenlet stopped sending heartbeats."

		conditions = map[gardencorev1beta1.ConditionType]gardencorev1beta1.Condition{
			gardencorev1beta1.ShootAPIServerAvailable:      {},
			gardencorev1beta1.ShootControlPlaneHealthy:     {},
			gardencorev1beta1.ShootEveryNodeReady:          {},
			gardencorev1beta1.ShootSystemComponentsHealthy: {},
		}

		constraints = map[gardencorev1beta1.ConditionType]gardencorev1beta1.Condition{
			gardencorev1beta1.ShootHibernationPossible:               {},
			gardencorev1beta1.ShootMaintenancePreconditionsSatisfied: {},
		}
	)

	for conditionType := range conditions {
		c := gardencorev1beta1helper.GetOrInitCondition(shoot.Status.Conditions, conditionType)
		c = gardencorev1beta1helper.UpdatedCondition(c, gardencorev1beta1.ConditionUnknown, reason, msg)
		conditions[conditionType] = c
	}

	for conditionType := range constraints {
		c := gardencorev1beta1helper.GetOrInitCondition(shoot.Status.Constraints, conditionType)
		c = gardencorev1beta1helper.UpdatedCondition(c, gardencorev1beta1.ConditionUnknown, reason, msg)
		constraints[conditionType] = c
	}

	patch := client.StrategicMergeFrom(shoot.DeepCopy())
	shoot.Status.Conditions = gardencorev1beta1helper.MergeConditions(shoot.Status.Conditions, conditionMapToConditions(conditions)...)
	shoot.Status.Constraints = gardencorev1beta1helper.MergeConditions(shoot.Status.Constraints, conditionMapToConditions(constraints)...)
	return c.Status().Patch(ctx, shoot, patch)
}

func conditionMapToConditions(m map[gardencorev1beta1.ConditionType]gardencorev1beta1.Condition) []gardencorev1beta1.Condition {
	output := make([]gardencorev1beta1.Condition, 0, len(m))

	for _, condition := range m {
		output = append(output, condition)
	}

	return output
}
