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

package backupentry

import (
	"context"
	"fmt"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	v1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/gardener/gardener/pkg/gardenlet/apis/config"
	confighelper "github.com/gardener/gardener/pkg/gardenlet/apis/config/helper"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const migrationReconcilerName = "migration"

func (c *Controller) backupEntryMigrationAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	c.backupEntryMigrationQueue.Add(key)
}

func (c *Controller) backupEntryMigrationUpdate(oldObj, newObj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		return
	}

	backupEntry, ok := newObj.(*gardencorev1beta1.BackupEntry)
	if !ok {
		return
	}
	if backupEntry.Generation == backupEntry.Status.ObservedGeneration && !v1beta1helper.HasOperationAnnotation(backupEntry.ObjectMeta) {
		return
	}

	c.backupEntryMigrationQueue.Add(key)
}

func (c *Controller) backupEntryMigrationDelete(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	c.backupEntryMigrationQueue.Add(key)
}

// newMigrationReconciler returns an implementation of reconcile.Reconciler that forces the backup entry's restoration
// to this seed during control plane migration if the preparation for migration in the source seed is not finished
// after a certain grace period and is considered unlikely to succeed ("bad case" scenario).
func newMigrationReconciler(gardenClient client.Client, config *config.GardenletConfiguration) reconcile.Reconciler {
	return &migrationReconciler{
		gardenClient: gardenClient,
		config:       config,
	}
}

type migrationReconciler struct {
	gardenClient client.Client
	config       *config.GardenletConfiguration
}

func (r *migrationReconciler) Reconcile(ctx context.Context, req reconcile.Request) (result reconcile.Result, err error) {
	log := logf.FromContext(ctx)

	backupEntry := &gardencorev1beta1.BackupEntry{}
	if err := r.gardenClient.Get(ctx, req.NamespacedName, backupEntry); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("Object is gone, stop reconciling")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("error retrieving object from store: %w", err)
	}

	// If the backup entry is being deleted or no longer being migrated to this seed, clear the migration start time
	if backupEntry.DeletionTimestamp != nil || !controllerutils.BackupEntryIsBeingMigratedToSeed(ctx, r.gardenClient, backupEntry, confighelper.SeedNameFromSeedConfig(r.config.SeedConfig)) {
		log.V(1).Info("Clearing migration start time")
		if err := setMigrationStartTime(ctx, r.gardenClient, backupEntry, nil); err != nil {
			return reconcile.Result{}, fmt.Errorf("could not clear migration start time: %w", err)
		}

		// Return without requeue as the backup entry is no longer being migrated (we should not force restore)
		return reconcile.Result{}, nil
	}

	// Set the migration start time if needed
	if backupEntry.Status.MigrationStartTime == nil {
		log.V(1).Info("Setting migration start time to current time")
		if err := setMigrationStartTime(ctx, r.gardenClient, backupEntry, &metav1.Time{Time: time.Now().UTC()}); err != nil {
			return reconcile.Result{}, fmt.Errorf("could not set migration start time: %w", err)
		}
	}

	// If the force-restore annotation is set or the grace period is elapsed and migration is not currently in progress,
	// update the backup entry status to force the restoration (fallback to the "bad case" scenario)
	log.V(1).Info("Checking whether restoration should be forceful")
	if hasForceRestoreAnnotation(backupEntry) || r.isGracePeriodElapsed(backupEntry) && !r.isMigrationInProgress(backupEntry) {

		log.Info("Updating status to force restoration")
		if err := updateStatusForRestore(ctx, r.gardenClient, backupEntry); err != nil {
			return reconcile.Result{}, fmt.Errorf("could not update backup entry status to force restoration: %w", err)
		}

		if hasForceRestoreAnnotation(backupEntry) {
			log.V(1).Info("Removing force-restore annotation")
			if err := removeForceRestoreAnnotation(ctx, r.gardenClient, backupEntry); err != nil {
				return reconcile.Result{}, fmt.Errorf("could not remove force-restore annotation: %w", err)
			}
		}

		// Return without requeue as the backup entry is no longer being migrated (we just forced the restoration)
		return reconcile.Result{}, nil
	}

	// Requeue after the configured sync period as the backup entry is still being migrated,
	// so we might need to force the restoration
	return reconcile.Result{RequeueAfter: r.config.Controllers.BackupEntryMigration.SyncPeriod.Duration}, nil
}

func (r *migrationReconciler) isGracePeriodElapsed(backupEntry *gardencorev1beta1.BackupEntry) bool {
	return time.Now().UTC().After(backupEntry.Status.MigrationStartTime.Add(r.config.Controllers.BackupEntryMigration.GracePeriod.Duration))
}

func (r *migrationReconciler) isMigrationInProgress(backupEntry *gardencorev1beta1.BackupEntry) bool {
	staleCutoffTime := metav1.NewTime(time.Now().UTC().Add(-r.config.Controllers.BackupEntryMigration.LastOperationStaleDuration.Duration))
	lastOperation := backupEntry.Status.LastOperation
	return lastOperation != nil &&
		lastOperation.Type == gardencorev1beta1.LastOperationTypeMigrate &&
		(lastOperation.State == gardencorev1beta1.LastOperationStateProcessing || lastOperation.State == gardencorev1beta1.LastOperationStateError) &&
		!lastOperation.LastUpdateTime.Before(&staleCutoffTime)
}

func setMigrationStartTime(ctx context.Context, c client.Client, backupEntry *gardencorev1beta1.BackupEntry, migrationStartTime *metav1.Time) error {
	patch := client.MergeFrom(backupEntry.DeepCopy())
	backupEntry.Status.MigrationStartTime = migrationStartTime
	return c.Status().Patch(ctx, backupEntry, patch)
}

func updateStatusForRestore(ctx context.Context, c client.Client, backupEntry *gardencorev1beta1.BackupEntry) error {
	patch := client.StrategicMergeFrom(backupEntry.DeepCopy())

	backupEntry.Status.LastOperation = &gardencorev1beta1.LastOperation{
		Type:           gardencorev1beta1.LastOperationTypeMigrate,
		State:          gardencorev1beta1.LastOperationStateAborted,
		Description:    "BackupEntry preparation for migration has been aborted.",
		LastUpdateTime: metav1.NewTime(time.Now().UTC()),
	}
	backupEntry.Status.LastError = nil
	backupEntry.Status.ObservedGeneration = backupEntry.Generation
	backupEntry.Status.SeedName = nil
	backupEntry.Status.MigrationStartTime = nil

	return c.Status().Patch(ctx, backupEntry, patch)
}

func hasForceRestoreAnnotation(backupEntry *gardencorev1beta1.BackupEntry) bool {
	return kutil.HasMetaDataAnnotation(backupEntry, v1beta1constants.AnnotationShootForceRestore, "true")
}

func removeForceRestoreAnnotation(ctx context.Context, c client.Client, backupEntry *gardencorev1beta1.BackupEntry) error {
	patch := client.MergeFrom(backupEntry.DeepCopy())
	delete(backupEntry.GetAnnotations(), v1beta1constants.AnnotationShootForceRestore)
	return c.Patch(ctx, backupEntry, patch)
}
