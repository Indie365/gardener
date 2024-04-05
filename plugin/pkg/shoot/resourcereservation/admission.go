// Copyright 2024 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package resourcereservation

import (
	"context"
	"errors"
	"fmt"
	"io"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/admission"

	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	admissioninitializer "github.com/gardener/gardener/pkg/apiserver/admission/initializer"
	gardencoreinformers "github.com/gardener/gardener/pkg/client/core/informers/externalversions"
	gardencorev1beta1listers "github.com/gardener/gardener/pkg/client/core/listers/core/v1beta1"
	plugin "github.com/gardener/gardener/plugin/pkg"
)

const (
	_ = 1 << (iota * 10)
	KiB
	MiB
	GiB
)

// Register registers a plugin.
func Register(plugins *admission.Plugins) {
	plugins.Register(plugin.PluginNameShootResourceReservation, func(config io.Reader) (admission.Interface, error) {
		cfg, err := LoadConfiguration(config)
		if err != nil {
			return nil, err
		}

		return New(cfg.UseGKEFormula), nil
	})
}

// ResourceReservation contains required information to process admission requests.
type ResourceReservation struct {
	*admission.Handler
	cloudProfileLister gardencorev1beta1listers.CloudProfileLister
	readyFunc          admission.ReadyFunc

	useGKEFormula bool
}

var (
	_ = admissioninitializer.WantsCoreInformerFactory(&ResourceReservation{})

	readyFuncs []admission.ReadyFunc
)

// New creates a new ResourceReservation admission plugin.
func New(useGKEFormula bool) admission.MutationInterface {
	return &ResourceReservation{
		Handler:       admission.NewHandler(admission.Create, admission.Update),
		useGKEFormula: useGKEFormula,
	}
}

// AssignReadyFunc assigns the ready function to the admission handler.
func (c *ResourceReservation) AssignReadyFunc(f admission.ReadyFunc) {
	c.readyFunc = f
	c.SetReadyFunc(f)
}

// SetCoreInformerFactory gets Lister from SharedInformerFactory.
func (c *ResourceReservation) SetCoreInformerFactory(f gardencoreinformers.SharedInformerFactory) {
	cloudProfileInformer := f.Core().V1beta1().CloudProfiles()
	c.cloudProfileLister = cloudProfileInformer.Lister()

	readyFuncs = append(readyFuncs, cloudProfileInformer.Informer().HasSynced)
}

// ValidateInitialization checks whether the plugin was correctly initialized.
func (c *ResourceReservation) ValidateInitialization() error {
	if c.cloudProfileLister == nil {
		return errors.New("missing cloudProfile lister")
	}
	return nil
}

// Admit injects default resource reservations into worker pools of shoot objects
func (c *ResourceReservation) Admit(_ context.Context, a admission.Attributes, _ admission.ObjectInterfaces) error {
	// Wait until the caches have been synced
	if c.readyFunc == nil {
		c.AssignReadyFunc(func() bool {
			for _, readyFunc := range readyFuncs {
				if !readyFunc() {
					return false
				}
			}
			return true
		})
	}
	if !c.WaitForReady() {
		return admission.NewForbidden(a, errors.New("not yet ready to handle request"))
	}

	switch {
	case a.GetKind().GroupKind() != core.Kind("Shoot"),
		a.GetSubresource() != "":
		return nil
	}

	shoot, ok := a.GetObject().(*core.Shoot)
	if !ok {
		return apierrors.NewInternalError(errors.New("could not convert resource into Shoot object"))
	}

	// Pass if the shoot is intended to get deleted
	if shoot.DeletionTimestamp != nil {
		return nil
	}

	if !c.useGKEFormula {
		setStaticResourceReservationDefaults(shoot)
		return nil
	}

	if shoot.Spec.Kubernetes.Kubelet != nil && shoot.Spec.Kubernetes.Kubelet.KubeReserved != nil {
		// Inject static defaults for shoots with global resource reservations
		setStaticResourceReservationDefaults(shoot)
		return nil
	}

	cloudProfile, err := c.cloudProfileLister.Get(shoot.Spec.CloudProfileName)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("could not find referenced cloud profile: %+v", err.Error()))
	}
	machineTypeMap := buildMachineTypeMap(cloudProfile)

	allErrs := field.ErrorList{}
	workersPath := field.NewPath("spec", "provider", "workers")

	for i := 0; i < len(shoot.Spec.Provider.Workers); i++ {
		workerPath := workersPath.Index(i)
		worker := &shoot.Spec.Provider.Workers[i]

		allErrs = injectResourceReservations(worker, machineTypeMap, *workerPath, allErrs)
	}

	if len(allErrs) > 0 {
		return apierrors.NewInvalid(a.GetKind().GroupKind(), a.GetName(), allErrs)
	}

	return nil
}

func setStaticResourceReservationDefaults(shoot *core.Shoot) {
	var (
		kubeReservedMemory = resource.MustParse("1Gi")
		kubeReservedCPU    = resource.MustParse("80m")
		kubeReservedPID    = resource.MustParse("20k")
	)

	if shoot.Spec.Kubernetes.Kubelet == nil {
		shoot.Spec.Kubernetes.Kubelet = &core.KubeletConfig{}
	}
	kubelet := shoot.Spec.Kubernetes.Kubelet

	if kubelet.KubeReserved == nil {
		kubelet.KubeReserved = &core.KubeletConfigReserved{
			CPU:    &kubeReservedCPU,
			Memory: &kubeReservedMemory,
			PID:    &kubeReservedPID,
		}
	} else {
		if kubelet.KubeReserved.Memory == nil {
			kubelet.KubeReserved.Memory = &kubeReservedMemory
		}
		if kubelet.KubeReserved.CPU == nil {
			kubelet.KubeReserved.CPU = &kubeReservedCPU
		}
		if kubelet.KubeReserved.PID == nil {
			kubelet.KubeReserved.PID = &kubeReservedPID
		}
	}
}

func injectResourceReservations(worker *core.Worker, machineTypeMap map[string]gardencorev1beta1.MachineType, path field.Path, allErrs field.ErrorList) field.ErrorList {
	reservation, err := calculateResourceReservationForMachineType(machineTypeMap, worker.Machine.Type)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(path.Child("machine", "type"), worker.Machine.Type, "worker machine type unknown"))
		return allErrs
	}

	if worker.Kubernetes == nil {
		worker.Kubernetes = &core.WorkerKubernetes{}
	}
	if worker.Kubernetes.Kubelet == nil {
		worker.Kubernetes.Kubelet = &core.KubeletConfig{}
	}
	if worker.Kubernetes.Kubelet.KubeReserved == nil {
		worker.Kubernetes.Kubelet.KubeReserved = reservation
	} else {
		kubeReserved := worker.Kubernetes.Kubelet.KubeReserved
		if kubeReserved.CPU == nil {
			kubeReserved.CPU = reservation.CPU
		}
		if kubeReserved.Memory == nil {
			kubeReserved.Memory = reservation.Memory
		}
		if kubeReserved.PID == nil {
			kubeReserved.PID = reservation.PID
		}
	}
	return allErrs
}

func buildMachineTypeMap(cloudProfile *gardencorev1beta1.CloudProfile) map[string]gardencorev1beta1.MachineType {
	types := map[string]gardencorev1beta1.MachineType{}

	for _, machine := range cloudProfile.Spec.MachineTypes {
		types[machine.Name] = machine
	}
	return types
}

func calculateResourceReservationForMachineType(machineTypeMap map[string]gardencorev1beta1.MachineType, machineType string) (*core.KubeletConfigReserved, error) {
	kubeReservedPID := resource.MustParse("20k")
	machine, ok := machineTypeMap[machineType]
	if !ok {
		return nil, fmt.Errorf("unknown machine type %v", machineType)
	}

	cpuReserved := calculateCPUReservation(machine.CPU.MilliValue())
	memoryReserved := calculateMemoryReservation(machine.Memory.Value())

	return &core.KubeletConfigReserved{
		CPU:    resource.NewMilliQuantity(cpuReserved, resource.BinarySI),
		Memory: resource.NewQuantity(memoryReserved, resource.BinarySI),
		PID:    &kubeReservedPID,
	}, nil
}

func calculateCPUReservation(cpuMilli int64) int64 {
	reservation := int64(0)
	// 6% of first core
	if cpuMilli > 0 {
		reservation += 60
	}
	// + 1% of second core
	if cpuMilli > 1000 {
		reservation += 10
	}
	// + 0.5% each for core 3 and 4
	if cpuMilli > 2000 {
		reservation += (min(cpuMilli/1000, 4) - 2) * 5
	}
	// + 0.25% for the remaining CPU cores
	if cpuMilli > 4000 {
		reservation += (cpuMilli/1000 - 4) * 5 / 2
	}

	return reservation
}

func calculateMemoryReservation(memory int64) int64 {
	reservation := int64(0)
	if memory < 1*GiB {
		reservation = 255 * MiB
	}
	// 25% of first 4 GB
	if memory >= 1*GiB {
		reservation += min(memory, 4*GiB) / 4
	}
	// 20% for additional memory between 4GB and 8GB
	if memory >= 4*GiB {
		reservation += (min(memory, 8*GiB) - 4*GiB) / 5
	}
	// 10% for additional memory between 8GB and 16GB
	if memory >= 8*GiB {
		reservation += (min(memory, 16*GiB) - 8*GiB) / 10
	}
	// 6% for additional memory between 16GB and 128GB
	if memory >= 16*GiB {
		reservation += (min(memory, 128*GiB) - 16*GiB) / 100 * 6
	}
	// 2% of remaining memory
	if memory >= 128*GiB {
		reservation += (memory - 128*GiB) / 100 * 2
	}

	return reservation
}
