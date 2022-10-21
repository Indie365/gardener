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

package maintenance

import (
	"context"
	"fmt"
	"strconv"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"github.com/gardener/gardener/pkg/controllermanager/apis/config"
	"github.com/gardener/gardener/pkg/controllerutils"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	"github.com/Masterminds/semver"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
	"k8s.io/utils/pointer"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const maxTrackedMaintenanceOperations = 3

// Reconciler reconciles Shoots and maintains them by updating versions or triggering operations.
type Reconciler struct {
	Client   client.Client
	Config   config.ShootMaintenanceControllerConfiguration
	Clock    clock.Clock
	Recorder record.EventRecorder
}

// Reconcile reconciles Shoots and maintains them by updating versions or triggering operations.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := logf.FromContext(ctx)

	shoot := &gardencorev1beta1.Shoot{}
	if err := r.Client.Get(ctx, request.NamespacedName, shoot); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("Object is gone, stop reconciling")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("error retrieving object from store: %w", err)
	}

	if shoot.DeletionTimestamp != nil {
		log.V(1).Info("Skipping Shoot because it is marked for deletion")
		return reconcile.Result{}, nil
	}

	requeueAfter, nextMaintenance := requeueAfterDuration(shoot)

	if !mustMaintainNow(shoot) {
		log.V(1).Info("Skipping Shoot because it doesn't need to be maintained now")
		log.V(1).Info("Scheduled next maintenance for Shoot", "duration", requeueAfter.Round(time.Minute), "nextMaintenance", nextMaintenance.Round(time.Minute))
		return reconcile.Result{RequeueAfter: requeueAfter}, nil
	}

	if err := r.reconcile(ctx, log, shoot); err != nil {
		return reconcile.Result{}, err
	}

	log.V(1).Info("Scheduled next maintenance for Shoot", "duration", requeueAfter.Round(time.Minute), "nextMaintenance", nextMaintenance.Round(time.Minute))
	return reconcile.Result{RequeueAfter: requeueAfter}, nil
}

func requeueAfterDuration(shoot *gardencorev1beta1.Shoot) (time.Duration, time.Time) {
	var (
		now             = time.Now()
		window          = gutil.EffectiveShootMaintenanceTimeWindow(shoot)
		duration        = window.RandomDurationUntilNext(now, false)
		nextMaintenance = time.Now().UTC().Add(duration)
	)

	return duration, nextMaintenance
}

func (r *Reconciler) reconcile(ctx context.Context, log logr.Logger, shoot *gardencorev1beta1.Shoot) error {
	log.Info("Maintaining Shoot")

	var (
		maintainedShoot  = shoot.DeepCopy()
		operations       []string
		lastMaintenances = shoot.Status.LastMaintenances
	)

	cloudProfile := &gardencorev1beta1.CloudProfile{}
	if err := r.Client.Get(ctx, kutil.Key(shoot.Spec.CloudProfileName), cloudProfile); err != nil {
		return err
	}

	reasonForImageUpdatePerPool, err := MaintainMachineImages(log, maintainedShoot, cloudProfile)
	if err != nil {
		// continue execution to allow the kubernetes version update
		log.Error(err, "Failed to maintain Shoot machine images")
	}
	operations = append(operations, reasonForImageUpdatePerPool...)

	reasonForKubernetesUpdate, err := maintainKubernetesVersion(log, maintainedShoot.Spec.Kubernetes.Version, maintainedShoot.Spec.Maintenance.AutoUpdate.KubernetesVersion, cloudProfile, func(v string) error {
		maintainedShoot.Spec.Kubernetes.Version = v
		return nil
	})
	if err != nil {
		// continue execution to allow the machine image version update
		log.Error(err, "Failed to maintain Shoot kubernetes version")
	}
	if reasonForKubernetesUpdate != "" {
		operations = append(operations, "Shoot "+reasonForKubernetesUpdate)
	}

	shootSemver, err := semver.NewVersion(maintainedShoot.Spec.Kubernetes.Version)
	if err != nil {
		return err
	}

	// Now it's time to update worker pool kubernetes version if specified
	var reasonsForWorkerPoolKubernetesUpdate = make(map[string]string)
	for i, w := range maintainedShoot.Spec.Provider.Workers {
		if w.Kubernetes == nil || w.Kubernetes.Version == nil {
			continue
		}

		workerLog := log.WithValues("worker", w.Name)

		reasonForWorkerPoolKubernetesUpdate, err := maintainKubernetesVersion(workerLog, *w.Kubernetes.Version, maintainedShoot.Spec.Maintenance.AutoUpdate.KubernetesVersion, cloudProfile, func(v string) error {
			workerPoolSemver, err := semver.NewVersion(v)
			if err != nil {
				return err
			}
			// If during autoupdate a worker pool kubernetes gets forcefully updated to the next minor which might be higher than the same minor of the shoot, take this
			if workerPoolSemver.GreaterThan(shootSemver) {
				workerPoolSemver = shootSemver
			}
			v = workerPoolSemver.String()
			maintainedShoot.Spec.Provider.Workers[i].Kubernetes.Version = &v
			return nil
		})
		if err != nil {
			// continue execution to allow the machine image version update
			workerLog.Error(err, "Could not maintain kubernetes version for worker pool")
		}
		reasonsForWorkerPoolKubernetesUpdate[w.Name] = reasonForWorkerPoolKubernetesUpdate
	}

	for workerPool, reason := range reasonsForWorkerPoolKubernetesUpdate {
		if reason != "" {
			operation := fmt.Sprintf("For worker pool %s: %s", workerPool, reason)
			operations = append(operations, operation)
		}
	}

	operation := maintainOperation(maintainedShoot)
	if operation != "" {
		operations = append(operations, operation)
	}

	tasks := maintainTasks(maintainedShoot, r.Config)
	operations = append(operations, tasks...)

	patch := client.MergeFrom(shoot.DeepCopy())
	lastMaintenances = append([]gardencorev1beta1.LastMaintenance{
		{
			Operations:    operations,
			TriggeredTime: metav1.Time{Time: r.Clock.Now()},
		},
	}, lastMaintenances...)

	if len(lastMaintenances) > maxTrackedMaintenanceOperations {
		shoot.Status.LastMaintenances = lastMaintenances[0:3]
	} else {
		shoot.Status.LastMaintenances = lastMaintenances
	}

	// here `gardener.cloud/operation`: `maintain` is removed so when patch call triggers
	// another reconciliation it should not be picked by the controller
	if hasMaintainNowAnnotation(shoot) {
		delete(shoot.Annotations, v1beta1constants.GardenerOperation)
	}

	if err := r.Client.Status().Patch(ctx, shoot, patch); err != nil {
		log.Error(err, "Failed to patch shoot status")
		return err
	}

	// update shoot spec changes in maintenance call
	shoot.Spec = *maintainedShoot.Spec.DeepCopy()
	_ = maintainOperation(shoot)
	_ = maintainTasks(shoot, r.Config)

	// try to maintain shoot, but don't retry on conflict, because a conflict means that we potentially operated on stale
	// data (e.g. when calculating the updated k8s version), so rather return error and backoff
	if err := r.Client.Update(ctx, shoot); err != nil {
		return err
	}

	for _, reason := range reasonForImageUpdatePerPool {
		r.Recorder.Eventf(shoot, corev1.EventTypeNormal, gardencorev1beta1.ShootEventImageVersionMaintenance, "%s",
			fmt.Sprintf("Updated %s.", reason))
	}

	if reasonForKubernetesUpdate != "" {
		r.Recorder.Eventf(shoot, corev1.EventTypeNormal, gardencorev1beta1.ShootEventK8sVersionMaintenance, "%s",
			fmt.Sprintf("Updated %s.", reasonForKubernetesUpdate))
	}

	for name, reason := range reasonsForWorkerPoolKubernetesUpdate {
		r.Recorder.Eventf(shoot, corev1.EventTypeNormal, gardencorev1beta1.ShootEventK8sVersionMaintenance, "%s",
			fmt.Sprintf("Updated worker pool %q %s.", name, reason))
	}

	log.Info("Shoot maintenance completed")
	return nil
}

func maintainOperation(shoot *gardencorev1beta1.Shoot) string {
	var operation string
	if hasMaintainNowAnnotation(shoot) {
		delete(shoot.Annotations, v1beta1constants.GardenerOperation)
	}

	if shoot.Status.LastOperation == nil {
		return ""
	}

	switch {
	case shoot.Status.LastOperation.State == gardencorev1beta1.LastOperationStateFailed:
		if needsRetry(shoot) {
			metav1.SetMetaDataAnnotation(&shoot.ObjectMeta, v1beta1constants.GardenerOperation, v1beta1constants.ShootOperationRetry)
			operation = v1beta1constants.GardenerOperation + ": " + v1beta1constants.ShootOperationRetry
			delete(shoot.Annotations, v1beta1constants.FailedShootNeedsRetryOperation)
		}
	default:
		operation = getOperation(shoot)
		metav1.SetMetaDataAnnotation(&shoot.ObjectMeta, v1beta1constants.GardenerOperation, operation)
		operation = v1beta1constants.GardenerOperation + ": " + operation
		delete(shoot.Annotations, v1beta1constants.GardenerMaintenanceOperation)
	}

	return operation
}

func maintainTasks(shoot *gardencorev1beta1.Shoot, config config.ShootMaintenanceControllerConfiguration) []string {
	var operations []string

	if shoot.Annotations == nil {
		shoot.Annotations = map[string]string{}
	}

	controllerutils.AddTasks(shoot.Annotations,
		v1beta1constants.ShootTaskDeployInfrastructure,
		v1beta1constants.ShootTaskDeployDNSRecordInternal,
		v1beta1constants.ShootTaskDeployDNSRecordExternal,
		v1beta1constants.ShootTaskDeployDNSRecordIngress,
	)

	operations = append(operations, "Reconcile following resources: Infrastructure, DNSRecordInternal, DNSRecordExternal, DNSRecordIngress")

	if pointer.BoolDeref(config.EnableShootControlPlaneRestarter, false) {
		controllerutils.AddTasks(shoot.Annotations, v1beta1constants.ShootTaskRestartControlPlanePods)
		operations = append(operations, "Restart Control Plane pods")
	}

	if pointer.BoolDeref(config.EnableShootCoreAddonRestarter, false) {
		controllerutils.AddTasks(shoot.Annotations, v1beta1constants.ShootTaskRestartCoreAddons)
		operations = append(operations, "Restart Core Addon Pods")
	}

	return operations
}

// MaintainMachineImages updates the machine images of a Shoot's worker pools if necessary
func MaintainMachineImages(log logr.Logger, shoot *gardencorev1beta1.Shoot, cloudProfile *gardencorev1beta1.CloudProfile) ([]string, error) {
	var reasonsForUpdate []string

	for i, worker := range shoot.Spec.Provider.Workers {
		workerImage := worker.Machine.Image
		workerLog := log.WithValues("worker", worker.Name, "image", workerImage.Name, "version", workerImage.Version)

		machineImageFromCloudProfile, err := determineMachineImage(cloudProfile, workerImage)
		if err != nil {
			return nil, err
		}

		filteredMachineImageVersionsFromCloudProfile := filterForArchitecture(&machineImageFromCloudProfile, worker.Machine.Architecture)
		filteredMachineImageVersionsFromCloudProfile = filterForCRI(filteredMachineImageVersionsFromCloudProfile, worker.CRI)
		shouldBeUpdated, reason, updatedMachineImage, err := shouldMachineImageBeUpdated(workerLog, shoot.Spec.Maintenance.AutoUpdate.MachineImageVersion, filteredMachineImageVersionsFromCloudProfile, workerImage)
		if err != nil {
			return nil, err
		}

		if !shouldBeUpdated {
			continue
		}

		shoot.Spec.Provider.Workers[i].Machine.Image = updatedMachineImage

		workerLog.Info("MachineImage will be updated", "newVersion", *updatedMachineImage.Version, "reason", reason)
		reasonsForUpdate = append(reasonsForUpdate, fmt.Sprintf("Machine image of worker-pool %q from %q version %q to version %q. Reason: %s", worker.Name, workerImage.Name, *workerImage.Version, *updatedMachineImage.Version, reason))
	}

	return reasonsForUpdate, nil
}

// maintainKubernetesVersion updates a Shoot's Kubernetes version if necessary and returns the reason why an update was done
func maintainKubernetesVersion(log logr.Logger, kubernetesVersion string, autoUpdate bool, profile *gardencorev1beta1.CloudProfile, updateFunc func(string) error) (string, error) {
	shouldBeUpdated, reason, isExpired, err := shouldKubernetesVersionBeUpdated(kubernetesVersion, autoUpdate, profile)
	if err != nil {
		return "", err
	}
	if !shouldBeUpdated {
		return "", nil
	}

	updatedKubernetesVersion, err := determineKubernetesVersion(kubernetesVersion, profile, isExpired)
	if err != nil {
		return "", err
	}
	if updatedKubernetesVersion == "" {
		return "", nil
	}

	err = updateFunc(updatedKubernetesVersion)
	if err != nil {
		return "", err
	}

	log.Info("Kubernetes version will be updated", "version", kubernetesVersion, "newVersion", updatedKubernetesVersion, "reason", reason)
	return fmt.Sprintf("Kubernetes version %q to version %q. Reason: %s", kubernetesVersion, updatedKubernetesVersion, reason), err
}

func determineKubernetesVersion(kubernetesVersion string, profile *gardencorev1beta1.CloudProfile, isExpired bool) (string, error) {
	// get latest version that qualifies for a patch update
	newerPatchVersionFound, latestPatchVersion, err := gardencorev1beta1helper.GetKubernetesVersionForPatchUpdate(profile, kubernetesVersion)
	if err != nil {
		return "", fmt.Errorf("failure while determining the latest Kubernetes patch version in the CloudProfile: %s", err.Error())
	}
	if newerPatchVersionFound {
		return latestPatchVersion, nil
	}
	// no newer patch version found & is expired -> forcefully update to latest patch of next minor version
	if isExpired {
		// get latest version that qualifies for a minor update
		newMinorAvailable, latestPatchVersionNewMinor, err := gardencorev1beta1helper.GetKubernetesVersionForMinorUpdate(profile, kubernetesVersion)
		if err != nil {
			return "", fmt.Errorf("failure while determining newer Kubernetes minor version in the CloudProfile: %s", err.Error())
		}
		// cannot update as there is no consecutive minor version available (e.g shoot is on 1.20.X, but there is only 1.22.X, available and not 1.21.X)
		if !newMinorAvailable {
			return "", fmt.Errorf("cannot perform minor Kubernetes version update for expired Kubernetes version %q. No suitable version found in CloudProfile - this is most likely a misconfiguration of the CloudProfile", kubernetesVersion)
		}

		return latestPatchVersionNewMinor, nil
	}
	return "", nil
}

func shouldKubernetesVersionBeUpdated(kubernetesVersion string, autoUpdate bool, profile *gardencorev1beta1.CloudProfile) (shouldBeUpdated bool, reason string, isExpired bool, error error) {
	versionExistsInCloudProfile, version, err := gardencorev1beta1helper.KubernetesVersionExistsInCloudProfile(profile, kubernetesVersion)
	if err != nil {
		return false, "", false, err
	}

	var updateReason string
	if !versionExistsInCloudProfile {
		updateReason = "Version does not exist in CloudProfile"
		return true, updateReason, true, nil
	}

	if ExpirationDateExpired(version.ExpirationDate) {
		updateReason = "Kubernetes version expired - force update required"
		return true, updateReason, true, nil
	}

	if autoUpdate {
		updateReason = "AutoUpdate of Kubernetes version configured"
		return true, updateReason, false, nil
	}

	return false, "", false, nil
}

func mustMaintainNow(shoot *gardencorev1beta1.Shoot) bool {
	return hasMaintainNowAnnotation(shoot) || gutil.IsNowInEffectiveShootMaintenanceTimeWindow(shoot)
}

func hasMaintainNowAnnotation(shoot *gardencorev1beta1.Shoot) bool {
	operation, ok := shoot.Annotations[v1beta1constants.GardenerOperation]
	return ok && operation == v1beta1constants.ShootOperationMaintain
}

func needsRetry(shoot *gardencorev1beta1.Shoot) bool {
	needsRetryOperation := false

	if val, ok := shoot.Annotations[v1beta1constants.FailedShootNeedsRetryOperation]; ok {
		needsRetryOperation, _ = strconv.ParseBool(val)
	}

	return needsRetryOperation
}

func getOperation(shoot *gardencorev1beta1.Shoot) string {
	var (
		operation            = v1beta1constants.GardenerOperationReconcile
		maintenanceOperation = shoot.Annotations[v1beta1constants.GardenerMaintenanceOperation]
	)

	if maintenanceOperation != "" {
		operation = maintenanceOperation
	}

	return operation
}

func filterForArchitecture(machineImageFromCloudProfile *gardencorev1beta1.MachineImage, arch *string) *gardencorev1beta1.MachineImage {
	filteredMachineImages := gardencorev1beta1.MachineImage{Name: machineImageFromCloudProfile.Name,
		Versions: []gardencorev1beta1.MachineImageVersion{}}

	for _, cloudProfileVersion := range machineImageFromCloudProfile.Versions {
		if slices.Contains(cloudProfileVersion.Architectures, *arch) {
			filteredMachineImages.Versions = append(filteredMachineImages.Versions, cloudProfileVersion)
		}
	}

	return &filteredMachineImages
}

func filterForCRI(machineImageFromCloudProfile *gardencorev1beta1.MachineImage, workerCRI *gardencorev1beta1.CRI) *gardencorev1beta1.MachineImage {
	if workerCRI == nil {
		return filterForCRI(machineImageFromCloudProfile, &gardencorev1beta1.CRI{Name: gardencorev1beta1.CRINameDocker})
	}

	filteredMachineImages := gardencorev1beta1.MachineImage{Name: machineImageFromCloudProfile.Name,
		Versions: []gardencorev1beta1.MachineImageVersion{}}

	for _, cloudProfileVersion := range machineImageFromCloudProfile.Versions {
		criFromCloudProfileVersion, found := findCRIByName(workerCRI.Name, cloudProfileVersion.CRI)
		if !found {
			continue
		}

		if !areAllWorkerCRsPartOfCloudProfileVersion(workerCRI.ContainerRuntimes, criFromCloudProfileVersion.ContainerRuntimes) {
			continue
		}

		filteredMachineImages.Versions = append(filteredMachineImages.Versions, cloudProfileVersion)
	}

	return &filteredMachineImages
}

func findCRIByName(wanted gardencorev1beta1.CRIName, cris []gardencorev1beta1.CRI) (gardencorev1beta1.CRI, bool) {
	for _, cri := range cris {
		if cri.Name == wanted {
			return cri, true
		}
	}
	return gardencorev1beta1.CRI{}, false
}

func areAllWorkerCRsPartOfCloudProfileVersion(workerCRs []gardencorev1beta1.ContainerRuntime, crsFromCloudProfileVersion []gardencorev1beta1.ContainerRuntime) bool {
	if workerCRs == nil {
		return true
	}
	for _, workerCr := range workerCRs {
		if !isWorkerCRPartOfCloudProfileVersionCRs(workerCr, crsFromCloudProfileVersion) {
			return false
		}
	}
	return true
}

func isWorkerCRPartOfCloudProfileVersionCRs(wanted gardencorev1beta1.ContainerRuntime, cloudProfileVersionCRs []gardencorev1beta1.ContainerRuntime) bool {
	for _, cr := range cloudProfileVersionCRs {
		if wanted.Type == cr.Type {
			return true
		}
	}
	return false
}

func determineMachineImage(cloudProfile *gardencorev1beta1.CloudProfile, shootMachineImage *gardencorev1beta1.ShootMachineImage) (gardencorev1beta1.MachineImage, error) {
	machineImagesFound, machineImageFromCloudProfile := gardencorev1beta1helper.DetermineMachineImageForName(cloudProfile, shootMachineImage.Name)
	if !machineImagesFound {
		return gardencorev1beta1.MachineImage{}, fmt.Errorf("failure while determining the default machine image in the CloudProfile: no machineImage with name %q (specified in shoot) could be found in the cloudProfile %q", shootMachineImage.Name, cloudProfile.Name)
	}

	return machineImageFromCloudProfile, nil
}

// shouldMachineImageBeUpdated determines if a machine image should be updated based on whether it exists in the CloudProfile, auto update applies or a force update is required.
func shouldMachineImageBeUpdated(log logr.Logger, autoUpdateMachineImageVersion bool, machineImage *gardencorev1beta1.MachineImage, shootMachineImage *gardencorev1beta1.ShootMachineImage) (shouldBeUpdated bool, reason string, updatedMachineImage *gardencorev1beta1.ShootMachineImage, error error) {
	versionExistsInCloudProfile, versionIndex := gardencorev1beta1helper.ShootMachineImageVersionExists(*machineImage, *shootMachineImage)
	var reasonForUpdate string

	forceUpdateRequired := ForceMachineImageUpdateRequired(shootMachineImage, *machineImage)
	if !versionExistsInCloudProfile || autoUpdateMachineImageVersion || forceUpdateRequired {
		// safe operation, as Shoot machine image version can only be a valid semantic version
		shootSemanticVersion := *semver.MustParse(*shootMachineImage.Version)

		// get latest version qualifying for Shoot machine image update
		qualifyingVersionFound, latestShootMachineImage, err := gardencorev1beta1helper.GetLatestQualifyingShootMachineImage(*machineImage, gardencorev1beta1helper.FilterLowerVersion(shootSemanticVersion))
		if err != nil {
			return false, "", nil, fmt.Errorf("an error occured while determining the latest Shoot Machine Image for machine image %q: %w", machineImage.Name, err)
		}

		// this is a special case when a Shoot is using a preview version. Preview versions should not be updated-to and are therefore not part of the qualifying versions.
		// if no qualifying version can be found and the Shoot is already using a preview version, then do nothing.
		if !qualifyingVersionFound && versionExistsInCloudProfile && machineImage.Versions[versionIndex].Classification != nil && *machineImage.Versions[versionIndex].Classification == gardencorev1beta1.ClassificationPreview {
			log.V(1).Info("MachineImage update not required, shoot worker is already using preview version")
			return false, "", nil, nil
		}

		// otherwise, there should always be a qualifying version (at least the Shoot's machine image version itself).
		if !qualifyingVersionFound {
			return false, "", nil, fmt.Errorf("no latest qualifying Shoot machine image could be determined for machine image %q. Either the machine image is reaching end of life and migration to another machine image is required or there is a misconfiguration in the CloudProfile. If it is the latter, make sure the machine image in the CloudProfile has at least one version that is not expired, not in preview and greater or equal to the current Shoot image version %q", machineImage.Name, *shootMachineImage.Version)
		}

		if *latestShootMachineImage.Version == *shootMachineImage.Version {
			log.V(1).Info("MachineImage update not required, shoot worker is already up to date")
			return false, "", nil, nil
		}

		if !versionExistsInCloudProfile {
			// deletion a machine image that is still in use by a Shoot is blocked in the apiserver. However it is still required,
			// because old installations might still have shoot's that have no corresponding version in the CloudProfile.
			reasonForUpdate = "Version does not exist in CloudProfile"
		} else if autoUpdateMachineImageVersion {
			reasonForUpdate = "AutoUpdate of MachineImage configured"
		} else if forceUpdateRequired {
			reasonForUpdate = "MachineImage expired - force update required"
		}

		return true, reasonForUpdate, latestShootMachineImage, nil
	}

	return false, "", nil, nil
}

// ForceMachineImageUpdateRequired checks if the shoots current machine image has to be forcefully updated
func ForceMachineImageUpdateRequired(shootCurrentImage *gardencorev1beta1.ShootMachineImage, imageCloudProvider gardencorev1beta1.MachineImage) bool {
	for _, image := range imageCloudProvider.Versions {
		if shootCurrentImage.Version != nil && *shootCurrentImage.Version != image.Version {
			continue
		}
		return ExpirationDateExpired(image.ExpirationDate)
	}
	return false
}

// ExpirationDateExpired returns if now is equal or after the given expirationDate
func ExpirationDateExpired(timestamp *metav1.Time) bool {
	if timestamp == nil {
		return false
	}
	return time.Now().UTC().After(timestamp.Time) || time.Now().UTC().Equal(timestamp.Time)
}
