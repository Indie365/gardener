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

package cloudprofile

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/controllermanager/apis/config"
	"github.com/gardener/gardener/pkg/controllerutils"
)

// Reconciler reconciles CloudProfiles.
type Reconciler struct {
	Client   client.Client
	Config   *config.CloudProfileControllerConfiguration
	Recorder record.EventRecorder
}

// Reconcile performs the main reconciliation logic.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := logf.FromContext(ctx)

	cloudProfile := &gardencorev1beta1.CloudProfile{}
	if err := r.Client.Get(ctx, request.NamespacedName, cloudProfile); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("Object is gone, stop reconciling")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("error retrieving object from store: %w", err)
	}

	// The deletionTimestamp labels the CloudProfile as intended to get deleted. Before deletion, it has to be ensured that
	// no Shoots and Seed are assigned to the CloudProfile anymore. If this is the case then the controller will remove
	// the finalizers from the CloudProfile so that it can be garbage collected.
	if cloudProfile.DeletionTimestamp != nil {
		if !sets.NewString(cloudProfile.Finalizers...).Has(gardencorev1beta1.GardenerName) {
			return reconcile.Result{}, nil
		}

		associatedShoots, err := controllerutils.DetermineShootsAssociatedTo(ctx, r.Client, cloudProfile)
		if err != nil {
			return reconcile.Result{}, err
		}

		if len(associatedShoots) == 0 {
			log.Info("No Shoots are referencing the CloudProfile, deletion accepted")

			if controllerutil.ContainsFinalizer(cloudProfile, gardencorev1beta1.GardenerName) {
				log.Info("Removing finalizer")
				if err := controllerutils.RemoveFinalizers(ctx, r.Client, cloudProfile, gardencorev1beta1.GardenerName); err != nil {
					return reconcile.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
				}
			}

			return reconcile.Result{}, nil
		}

		message := fmt.Sprintf("Cannot delete CloudProfile, because the following Shoots are still referencing it: %+v", associatedShoots)
		r.Recorder.Event(cloudProfile, corev1.EventTypeNormal, v1beta1constants.EventResourceReferenced, message)
		return reconcile.Result{}, fmt.Errorf(message)
	}

	if !controllerutil.ContainsFinalizer(cloudProfile, gardencorev1beta1.GardenerName) {
		log.Info("Adding finalizer")
		if err := controllerutils.AddFinalizers(ctx, r.Client, cloudProfile, gardencorev1beta1.GardenerName); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	// TODO voelzmo - this migration step ensures that all MachineImageVersions in the Cloud Profile contain `docker` in their list of supported Container Runtimes
	// This can be removed in a couple of versions. Note that while this is still in here, it is impossible to add an image without `docker` support!
	migrationHappened := migrateMachineImageVersionCRISupport(cloudProfile)

	if migrationHappened {
		log.Info("Migrated Machine Image Versions to explicitly contain `docker` as supported CRI")
		if err := r.Client.Update(ctx, cloudProfile); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update Cloud Profile spec for machine image CRI migration: %w", err)
		}
	}

	return reconcile.Result{}, nil
}

func migrateMachineImageVersionCRISupport(cloudProfile *gardencorev1beta1.CloudProfile) bool {
	var migrationHappened bool
	for i, image := range cloudProfile.Spec.MachineImages {
		for j, version := range image.Versions {
			if containsDockerCRIName(version.CRI) {
				continue
			}
			cloudProfile.Spec.MachineImages[i].Versions[j].CRI = append(version.CRI, gardencorev1beta1.CRI{Name: gardencorev1beta1.CRINameDocker})
			migrationHappened = true
		}
	}
	return migrationHappened
}

func containsDockerCRIName(cris []gardencorev1beta1.CRI) bool {
	for _, cri := range cris {
		if cri.Name == gardencorev1beta1.CRINameDocker {
			return true
		}
	}
	return false
}
