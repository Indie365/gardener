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

package quotavalidator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gardener/gardener/pkg/apis/core"
	"github.com/gardener/gardener/pkg/apis/garden"
	"github.com/gardener/gardener/pkg/apis/garden/helper"
	admissioninitializer "github.com/gardener/gardener/pkg/apiserver/admission/initializer"
	informers "github.com/gardener/gardener/pkg/client/garden/informers/internalversion"
	listers "github.com/gardener/gardener/pkg/client/garden/listers/garden/internalversion"
	"github.com/gardener/gardener/pkg/operation/common"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apiserver/pkg/admission"
)

const (
	// PluginName is the name of this admission plugin.
	PluginName = "ShootQuotaValidator"
)

var (
	quotaMetricNames = [6]corev1.ResourceName{
		garden.QuotaMetricCPU,
		garden.QuotaMetricGPU,
		garden.QuotaMetricMemory,
		garden.QuotaMetricStorageStandard,
		garden.QuotaMetricStoragePremium,
		garden.QuotaMetricLoadbalancer}
)

type quotaWorker struct {
	garden.Worker
	// VolumeType is the type of the root volumes.
	VolumeType string
	// VolumeSize is the size of the root volume.
	VolumeSize resource.Quantity
}

// Register registers a plugin.
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return New()
	})
}

// QuotaValidator contains listers and and admission handler.
type QuotaValidator struct {
	*admission.Handler
	shootLister         listers.ShootLister
	cloudProfileLister  listers.CloudProfileLister
	secretBindingLister listers.SecretBindingLister
	quotaLister         listers.QuotaLister
	readyFunc           admission.ReadyFunc
}

var (
	_ = admissioninitializer.WantsInternalGardenInformerFactory(&QuotaValidator{})

	readyFuncs = []admission.ReadyFunc{}
)

// New creates a new QuotaValidator admission plugin.
func New() (*QuotaValidator, error) {
	return &QuotaValidator{
		Handler: admission.NewHandler(admission.Create, admission.Update),
	}, nil
}

// AssignReadyFunc assigns the ready function to the admission handler.
func (q *QuotaValidator) AssignReadyFunc(f admission.ReadyFunc) {
	q.readyFunc = f
	q.SetReadyFunc(f)
}

// SetInternalGardenInformerFactory gets Lister from SharedInformerFactory.
func (q *QuotaValidator) SetInternalGardenInformerFactory(f informers.SharedInformerFactory) {
	shootInformer := f.Garden().InternalVersion().Shoots()
	q.shootLister = shootInformer.Lister()

	cloudProfileInformer := f.Garden().InternalVersion().CloudProfiles()
	q.cloudProfileLister = cloudProfileInformer.Lister()

	secretBindingInformer := f.Garden().InternalVersion().SecretBindings()
	q.secretBindingLister = secretBindingInformer.Lister()

	quotaInformer := f.Garden().InternalVersion().Quotas()
	q.quotaLister = quotaInformer.Lister()

	readyFuncs = append(readyFuncs, shootInformer.Informer().HasSynced, cloudProfileInformer.Informer().HasSynced, secretBindingInformer.Informer().HasSynced, quotaInformer.Informer().HasSynced)
}

// ValidateInitialization checks whether the plugin was correctly initialized.
func (q *QuotaValidator) ValidateInitialization() error {
	if q.shootLister == nil {
		return errors.New("missing shoot lister")
	}
	if q.cloudProfileLister == nil {
		return errors.New("missing cloudProfile lister")
	}
	if q.secretBindingLister == nil {
		return errors.New("missing secretBinding lister")
	}
	if q.quotaLister == nil {
		return errors.New("missing quota lister")
	}
	return nil
}

var _ admission.ValidationInterface = &QuotaValidator{}

// Validate checks that the requested Shoot resources do not exceed the quota limits.
func (q *QuotaValidator) Validate(ctx context.Context, a admission.Attributes, o admission.ObjectInterfaces) error {
	// Wait until the caches have been synced
	if q.readyFunc == nil {
		q.AssignReadyFunc(func() bool {
			for _, readyFunc := range readyFuncs {
				if !readyFunc() {
					return false
				}
			}
			return true
		})
	}
	if !q.WaitForReady() {
		return admission.NewForbidden(a, errors.New("not yet ready to handle request"))
	}

	// Ignore all kinds other than Shoot
	if a.GetKind().GroupKind() != garden.Kind("Shoot") && a.GetKind().GroupKind() != core.Kind("Shoot") {
		return nil
	}
	if a.GetSubresource() != "" {
		return nil
	}

	shoot, ok := a.GetObject().(*garden.Shoot)
	if !ok {
		return apierrors.NewBadRequest("could not convert resource into Shoot object")
	}

	// Pass if the shoot is intended to get deleted
	if shoot.DeletionTimestamp != nil {
		return nil
	}

	var (
		oldShoot         *garden.Shoot
		maxShootLifetime *int
		checkLifetime    = false
		checkQuota       = false
	)

	if a.GetOperation() == admission.Create {
		checkQuota = true
	}

	if a.GetOperation() == admission.Update {
		oldShoot, ok = a.GetOldObject().(*garden.Shoot)
		if !ok {
			return apierrors.NewBadRequest("could not convert resource into Shoot object")
		}

		checkQuota = quotaVerificationNeeded(*shoot, *oldShoot)
		checkLifetime = lifetimeVerificationNeeded(*shoot, *oldShoot)
	}

	secretBinding, err := q.secretBindingLister.SecretBindings(shoot.Namespace).Get(shoot.Spec.SecretBindingName)
	if err != nil {
		return apierrors.NewInternalError(err)
	}

	// Quotas are cumulative, means each quota must be not exceeded that the admission pass.
	for _, quotaRef := range secretBinding.Quotas {
		quota, err := q.quotaLister.Quotas(quotaRef.Namespace).Get(quotaRef.Name)
		if err != nil {
			return apierrors.NewInternalError(err)
		}

		// Get the max clusterLifeTime
		if checkLifetime && quota.Spec.ClusterLifetimeDays != nil {
			if maxShootLifetime == nil {
				maxShootLifetime = quota.Spec.ClusterLifetimeDays
			}
			if *maxShootLifetime > *quota.Spec.ClusterLifetimeDays {
				maxShootLifetime = quota.Spec.ClusterLifetimeDays
			}
		}

		if checkQuota {
			exceededMetrics, err := q.isQuotaExceeded(*shoot, *quota)
			if err != nil {
				return apierrors.NewInternalError(err)
			}
			if exceededMetrics != nil {
				message := ""
				for _, metric := range *exceededMetrics {
					message = message + metric.String() + " "
				}
				return admission.NewForbidden(a, fmt.Errorf("Quota limits exceeded. Unable to allocate further %s", message))
			}
		}
	}

	// Admit Shoot lifetime changes
	if lifetime, exists := shoot.Annotations[common.ShootExpirationTimestamp]; checkLifetime && exists && maxShootLifetime != nil {
		var (
			plannedExpirationTime     time.Time
			oldExpirationTime         time.Time
			maxPossibleExpirationTime time.Time
		)

		plannedExpirationTime, err = time.Parse(time.RFC3339, lifetime)
		if err != nil {
			return apierrors.NewInternalError(err)
		}

		oldLifetime, exists := oldShoot.Annotations[common.ShootExpirationTimestamp]
		if !exists {
			// The old version of the Shoot has no clusterLifetime annotation yet.
			// Therefore we have to calculate the lifetime based on the maxShootLifetime.
			oldLifetime = oldShoot.CreationTimestamp.Time.Add(time.Duration(*maxShootLifetime*24) * time.Hour).Format(time.RFC3339)
		}
		oldExpirationTime, err = time.Parse(time.RFC3339, oldLifetime)
		if err != nil {
			return apierrors.NewInternalError(err)
		}

		maxPossibleExpirationTime = oldExpirationTime.Add(time.Duration(*maxShootLifetime*24) * time.Hour)
		if plannedExpirationTime.After(maxPossibleExpirationTime) {
			return admission.NewForbidden(a, fmt.Errorf("Requested shoot expiration time to long. Can only be extended by %d day(s)", *maxShootLifetime))
		}
	}

	return nil
}

func (q *QuotaValidator) isQuotaExceeded(shoot garden.Shoot, quota garden.Quota) (*[]corev1.ResourceName, error) {
	allocatedResources, err := q.determineAllocatedResources(quota, shoot)
	if err != nil {
		return nil, err
	}
	requiredResources, err := q.determineRequiredResources(allocatedResources, shoot)
	if err != nil {
		return nil, err
	}

	exceededMetrics := make([]corev1.ResourceName, 0)
	for _, metric := range quotaMetricNames {
		if _, ok := quota.Spec.Metrics[metric]; !ok {
			continue
		}
		if !hasSufficientQuota(quota.Spec.Metrics[metric], requiredResources[metric]) {
			exceededMetrics = append(exceededMetrics, metric)
		}
	}
	if len(exceededMetrics) != 0 {
		return &exceededMetrics, nil
	}
	return nil, nil
}

func (q *QuotaValidator) determineAllocatedResources(quota garden.Quota, shoot garden.Shoot) (corev1.ResourceList, error) {
	shoots, err := q.findShootsReferQuota(quota, shoot)
	if err != nil {
		return nil, err
	}

	// Collect the resources which are allocated according to the shoot specs
	allocatedResources := make(corev1.ResourceList)
	for _, s := range shoots {
		shootResources, err := q.getShootResources(s)
		if err != nil {
			return nil, err
		}
		for _, metric := range quotaMetricNames {
			allocatedResources[metric] = sumQuantity(allocatedResources[metric], shootResources[metric])
		}
	}

	// TODO: We have to determine and add the amount of storage, which is allocated by manually created persistent volumes
	// and the count of loadbalancer, which are created due to manually created services of type loadbalancer

	return allocatedResources, nil
}

func (q *QuotaValidator) findShootsReferQuota(quota garden.Quota, shoot garden.Shoot) ([]garden.Shoot, error) {
	var (
		shootsReferQuota []garden.Shoot
		secretBindings   []garden.SecretBinding
	)

	scope, err := helper.QuotaScope(quota.Spec.Scope)
	if err != nil {
		return nil, err
	}

	namespace := corev1.NamespaceAll
	if scope == "project" {
		namespace = shoot.Namespace
	}
	allSecretBindings, err := q.secretBindingLister.SecretBindings(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	for _, binding := range allSecretBindings {
		for _, quotaRef := range binding.Quotas {
			if quota.Name == quotaRef.Name && quota.Namespace == quotaRef.Namespace {
				secretBindings = append(secretBindings, *binding)
			}
		}
	}

	for _, binding := range secretBindings {
		shoots, err := q.shootLister.Shoots(binding.Namespace).List(labels.Everything())
		if err != nil {
			return nil, err
		}
		for _, s := range shoots {
			if shoot.Namespace == s.Namespace && shoot.Name == s.Name {
				continue
			}
			if s.Spec.SecretBindingName == binding.Name {
				shootsReferQuota = append(shootsReferQuota, *s)
			}
		}
	}
	return shootsReferQuota, nil
}

func (q *QuotaValidator) determineRequiredResources(allocatedResources corev1.ResourceList, shoot garden.Shoot) (corev1.ResourceList, error) {
	shootResources, err := q.getShootResources(shoot)
	if err != nil {
		return nil, err
	}

	requiredResources := make(corev1.ResourceList)
	for _, metric := range quotaMetricNames {
		requiredResources[metric] = sumQuantity(allocatedResources[metric], shootResources[metric])
	}
	return requiredResources, nil
}

func (q *QuotaValidator) getShootResources(shoot garden.Shoot) (corev1.ResourceList, error) {
	cloudProfile, err := q.cloudProfileLister.Get(shoot.Spec.CloudProfileName)
	if err != nil {
		return nil, apierrors.NewBadRequest("could not find referenced cloud profile")
	}

	var (
		countLB      int64 = 1
		resources          = make(corev1.ResourceList)
		workers            = getShootWorkerResources(&shoot, cloudProfile)
		machineTypes       = cloudProfile.Spec.MachineTypes
		volumeTypes        = cloudProfile.Spec.VolumeTypes
	)

	for _, worker := range workers {
		var (
			machineType *garden.MachineType
			volumeType  *garden.VolumeType
		)

		// Get the proper machineType
		for _, element := range machineTypes {
			if element.Name == worker.Machine.Type {
				machineType = &element
				break
			}
		}
		if machineType == nil {
			return nil, fmt.Errorf("MachineType %s not found in CloudProfile %s", worker.Machine.Type, cloudProfile.Name)
		}

		if worker.Volume != nil {
			if machineType.Storage != nil {
				volumeType = &garden.VolumeType{
					Class: machineType.Storage.Class,
				}
			} else {
				// Get the proper VolumeType
				for _, element := range volumeTypes {
					if worker.Volume.Type != nil && element.Name == *worker.Volume.Type {
						volumeType = &element
						break
					}
				}
			}
		}
		if volumeType == nil {
			return nil, fmt.Errorf("VolumeType %s not found in CloudProfile %s", worker.Machine.Type, cloudProfile.Name)
		}

		// For now we always use the max. amount of resources for quota calculation
		resources[garden.QuotaMetricCPU] = sumQuantity(resources[garden.QuotaMetricCPU], multiplyQuantity(machineType.CPU, worker.Maximum))
		resources[garden.QuotaMetricGPU] = sumQuantity(resources[garden.QuotaMetricGPU], multiplyQuantity(machineType.GPU, worker.Maximum))
		resources[garden.QuotaMetricMemory] = sumQuantity(resources[garden.QuotaMetricMemory], multiplyQuantity(machineType.Memory, worker.Maximum))

		size, _ := resource.ParseQuantity("0Gi")
		if worker.Volume != nil {
			size, err = resource.ParseQuantity(worker.Volume.Size)
			if err != nil {
				return nil, err
			}
		}

		switch volumeType.Class {
		case garden.VolumeClassStandard:
			resources[garden.QuotaMetricStorageStandard] = sumQuantity(resources[garden.QuotaMetricStorageStandard], multiplyQuantity(size, worker.Maximum))
		case garden.VolumeClassPremium:
			resources[garden.QuotaMetricStoragePremium] = sumQuantity(resources[garden.QuotaMetricStoragePremium], multiplyQuantity(size, worker.Maximum))
		default:
			return nil, fmt.Errorf("Unknown volumeType class %s", volumeType.Class)
		}
	}

	if shoot.Spec.Addons != nil && shoot.Spec.Addons.NginxIngress != nil && shoot.Spec.Addons.NginxIngress.Addon.Enabled {
		countLB++
	}
	resources[garden.QuotaMetricLoadbalancer] = *resource.NewQuantity(countLB, resource.DecimalSI)

	return resources, nil
}

func getShootWorkerResources(shoot *garden.Shoot, cloudProfile *garden.CloudProfile) []garden.Worker {
	workers := make([]garden.Worker, 0, len(shoot.Spec.Provider.Workers))

	for _, worker := range shoot.Spec.Provider.Workers {
		workerCopy := worker.DeepCopy()

		if worker.Volume == nil {
			for _, machineType := range cloudProfile.Spec.MachineTypes {
				if worker.Machine.Type == machineType.Name && machineType.Storage != nil {
					workerCopy.Volume = &garden.Volume{
						Type: &machineType.Storage.Type,
						Size: machineType.Storage.Size.String(),
					}
				}
			}
		}

		workers = append(workers, *workerCopy)
	}

	return workers
}

func lifetimeVerificationNeeded(new, old garden.Shoot) bool {
	oldLifetime, exits := old.Annotations[common.ShootExpirationTimestamp]
	if !exits {
		oldLifetime = old.CreationTimestamp.String()
	}
	if oldLifetime != new.Annotations[common.ShootExpirationTimestamp] {
		return true
	}
	return false
}

func quotaVerificationNeeded(new, old garden.Shoot) bool {
	// Check for diff on addon nginx-ingress (addon requires to deploy a load balancer)
	var (
		oldNginxIngressEnabled bool
		newNginxIngressEnabled bool
	)

	if old.Spec.Addons != nil && old.Spec.Addons.NginxIngress != nil {
		oldNginxIngressEnabled = old.Spec.Addons.NginxIngress.Enabled
	}

	if new.Spec.Addons != nil && new.Spec.Addons.NginxIngress != nil {
		newNginxIngressEnabled = new.Spec.Addons.NginxIngress.Enabled
	}

	if oldNginxIngressEnabled == false && newNginxIngressEnabled == true {
		return true
	}

	// Check for diffs on workers
	for _, worker := range new.Spec.Provider.Workers {
		oldHasWorker := false
		for _, oldWorker := range old.Spec.Provider.Workers {
			if worker.Name == oldWorker.Name {
				oldHasWorker = true
				if worker.Machine.Type != oldWorker.Machine.Type || worker.Maximum != oldWorker.Maximum || !apiequality.Semantic.DeepEqual(worker.Volume, oldWorker.Volume) {
					return true
				}
			}
		}
		if !oldHasWorker {
			return true
		}
	}

	return false
}

func hasSufficientQuota(limit, required resource.Quantity) bool {
	compareCode := limit.Cmp(required)
	return compareCode != -1
}

func sumQuantity(values ...resource.Quantity) resource.Quantity {
	res := resource.Quantity{}
	for _, v := range values {
		res.Add(v)
	}
	return res
}

func multiplyQuantity(quantity resource.Quantity, multiplier int) resource.Quantity {
	res := resource.Quantity{}
	for i := 0; i < multiplier; i++ {
		res.Add(quantity)
	}
	return res
}
