// Copyright 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package webhook

import (
	"reflect"
	"regexp"
	"strings"

	"slices"

	"github.com/coreos/go-systemd/v22/unit"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

// LogMutation provides a log message.
func LogMutation(logger logr.Logger, kind, namespace, name string) {
	logger.Info("Mutating resource", "kind", kind, "namespace", namespace, "name", name)
}

// AppendUniqueUnit appens a unit only if it does not exist.
func AppendUniqueUnit(units *[]extensionsv1alpha1.Unit, unit extensionsv1alpha1.Unit) {
	for _, un := range *units {
		if un.Name == unit.Name {
			return
		}
	}
	*units = append(*units, unit)
}

// splitCommandLineRegex is used to split command line arguments by white space or "\".
var splitCommandLineRegex = regexp.MustCompile(`[\\\s]+`)

// DeserializeCommandLine de-serializes the given string to a slice of command line elements by splitting it
// on white space and the "\" character.
func DeserializeCommandLine(s string) []string {
	return splitCommandLineRegex.Split(s, -1)
}

// SerializeCommandLine serializes the given command line elements slice to a string by joining the first
// n+1 elements with a space " ", and all subsequent elements with the given separator.
func SerializeCommandLine(command []string, n int, sep string) string {
	if len(command) <= n {
		return strings.Join(command, " ")
	}
	if n == 0 {
		return strings.Join(command, sep)
	}
	return strings.Join(command[0:n], " ") + " " + strings.Join(command[n:], sep)
}

// ContainerWithName returns the container with the given name if it exists in the given slice, nil otherwise.
func ContainerWithName(containers []corev1.Container, name string) *corev1.Container {
	for _, container := range containers {
		if container.Name == name {
			return &container
		}
	}
	return nil
}

// PVCWithName returns the PersistentVolumeClaim with the given name if it exists in the given slice, nil otherwise.
func PVCWithName(pvcs []corev1.PersistentVolumeClaim, name string) *corev1.PersistentVolumeClaim {
	for _, pvc := range pvcs {
		if pvc.Name == name {
			return &pvc
		}
	}
	return nil
}

// UnitWithName returns the unit with the given name if it exists in the given slice, nil otherwise.
func UnitWithName(units []extensionsv1alpha1.Unit, name string) *extensionsv1alpha1.Unit {
	for _, unit := range units {
		if unit.Name == name {
			return &unit
		}
	}
	return nil
}

// FileWithPath returns the file with the given path if it exists in the given slice, nil otherwise.
func FileWithPath(files []extensionsv1alpha1.File, path string) *extensionsv1alpha1.File {
	for _, file := range files {
		if file.Path == path {
			return &file
		}
	}
	return nil
}

// UnitOptionWithSectionAndName returns the unit option with the given section and name if it exists in the given slice, nil otherwise.
func UnitOptionWithSectionAndName(opts []*unit.UnitOption, section, name string) *unit.UnitOption {
	for _, opt := range opts {
		if opt.Section == section && opt.Name == name {
			return opt
		}
	}

	return nil
}

// EnsureStringWithPrefix ensures that a string having the given prefix exists in the given slice
// and all matches are with a value equal to prefix + value.
func EnsureStringWithPrefix(items []string, prefix, value string) []string {
	if i := StringWithPrefixIndex(items, prefix); i < 0 {
		return append(items, prefix+value)
	}

	for i, item := range items {
		if !strings.HasPrefix(item, prefix) {
			continue
		}
		if item != prefix+value {
			items = append(append(items[:i], prefix+value), items[i+1:]...)
		}
	}
	return items
}

// EnsureNoStringWithPrefix ensures that a string having the given prefix does not exist in the given slice.
func EnsureNoStringWithPrefix(items []string, prefix string) []string {
	return slices.DeleteFunc(items, func(s string) bool {
		return strings.HasPrefix(s, prefix)
	})
}

// EnsureStringWithPrefixContains ensures that a string having the given prefix exists in the given slice
// and all matches contain the given value in a list separated by sep.
func EnsureStringWithPrefixContains(items []string, prefix, value, sep string) []string {
	if i := StringWithPrefixIndex(items, prefix); i < 0 {
		return append(items, prefix+value)
	}

	for i, item := range items {
		if !strings.HasPrefix(item, prefix) {
			continue
		}
		valuesList := strings.TrimPrefix(items[i], prefix)
		var values []string
		if valuesList != "" {
			values = strings.Split(valuesList, sep)
		}
		if j := StringIndex(values, value); j < 0 {
			values = append(values, value)
			items = append(append(items[:i], prefix+strings.Join(values, sep)), items[i+1:]...)
		}
	}
	return items
}

// EnsureNoStringWithPrefixContains ensures that either a string having the given prefix does not exist in the given slice,
// or it doesn't contain the given value in a list separated by sep.
func EnsureNoStringWithPrefixContains(items []string, prefix, value, sep string) []string {
	if i := StringWithPrefixIndex(items, prefix); i >= 0 {
		values := strings.Split(strings.TrimPrefix(items[i], prefix), sep)
		if j := StringIndex(values, value); j >= 0 {
			values = append(values[:j], values[j+1:]...)
			items = append(append(items[:i], prefix+strings.Join(values, sep)), items[i+1:]...)
		}
	}
	return items
}

// EnsureEnvVarWithName ensures that a EnvVar with a name equal to the name of the given EnvVar exists
// in the given slice and the first item in the list would be equal to the given EnvVar.
func EnsureEnvVarWithName(items []corev1.EnvVar, item corev1.EnvVar) []corev1.EnvVar {
	i := slices.IndexFunc(items, func(ev corev1.EnvVar) bool {
		return ev.Name == item.Name
	})
	if i < 0 {
		return append(items, item)
	}
	return append(append(items[:i], item), items[i+1:]...)
}

// EnsureNoEnvVarWithName ensures that a EnvVar with the given name does not exist in the given slice.
func EnsureNoEnvVarWithName(items []corev1.EnvVar, name string) []corev1.EnvVar {
	return slices.DeleteFunc(items, func(ev corev1.EnvVar) bool {
		return ev.Name == name
	})
}

// EnsureVolumeMountWithName ensures that a VolumeMount with a name equal to the name of the given VolumeMount exists
// in the given slice and the first item in the list would be equal to the given VolumeMount.
func EnsureVolumeMountWithName(items []corev1.VolumeMount, item corev1.VolumeMount) []corev1.VolumeMount {
	i := slices.IndexFunc(items, func(vm corev1.VolumeMount) bool {
		return vm.Name == item.Name
	})
	if i < 0 {
		return append(items, item)
	}
	return append(append(items[:i], item), items[i+1:]...)
}

// EnsureNoVolumeMountWithName ensures that a VolumeMount with the given name does not exist in the given slice.
func EnsureNoVolumeMountWithName(items []corev1.VolumeMount, name string) []corev1.VolumeMount {
	return slices.DeleteFunc(items, func(vm corev1.VolumeMount) bool {
		return vm.Name == name
	})
}

// EnsureVolumeWithName ensures that a Volume with a name equal to the name of the given Volume exists
// in the given slice and the first item in the list would be equal to the given Volume.
func EnsureVolumeWithName(items []corev1.Volume, item corev1.Volume) []corev1.Volume {
	i := slices.IndexFunc(items, func(v corev1.Volume) bool {
		return v.Name == item.Name
	})
	if i < 0 {
		return append(items, item)
	}
	return append(append(items[:i], item), items[i+1:]...)
}

// EnsureNoVolumeWithName ensures that a Volume with the given name does not exist in the given slice.
func EnsureNoVolumeWithName(items []corev1.Volume, name string) []corev1.Volume {
	return slices.DeleteFunc(items, func(v corev1.Volume) bool {
		return v.Name == name
	})
}

// EnsureContainerWithName ensures that a Container with a name equal to the name of the given Container exists
// in the given slice and the first item in the list would be equal to the given Container.
func EnsureContainerWithName(items []corev1.Container, item corev1.Container) []corev1.Container {
	i := slices.IndexFunc(items, func(c corev1.Container) bool {
		return c.Name == item.Name
	})
	if i < 0 {
		return append(items, item)
	}
	return append(append(items[:i], item), items[i+1:]...)
}

// EnsureNoContainerWithName ensures that a Container with the given name does not exist in the given slice.
func EnsureNoContainerWithName(items []corev1.Container, name string) []corev1.Container {
	return slices.DeleteFunc(items, func(c corev1.Container) bool {
		return c.Name == name
	})
}

// EnsureVPAContainerResourcePolicyWithName ensures that a container policy with a name equal to the name of the given
// container policy exists in the given slice and the first item in the list would be equal to the given container policy.
func EnsureVPAContainerResourcePolicyWithName(items []vpaautoscalingv1.ContainerResourcePolicy, item vpaautoscalingv1.ContainerResourcePolicy) []vpaautoscalingv1.ContainerResourcePolicy {
	i := slices.IndexFunc(items, func(crp vpaautoscalingv1.ContainerResourcePolicy) bool {
		return crp.ContainerName == item.ContainerName
	})
	if i < 0 {
		return append(items, item)
	}
	return append(append(items[:i], item), items[i+1:]...)
}

// EnsurePVCWithName ensures that a PVC with a name equal to the name of the given PVC exists
// in the given slice and the first item in the list would be equal to the given PVC.
func EnsurePVCWithName(items []corev1.PersistentVolumeClaim, item corev1.PersistentVolumeClaim) []corev1.PersistentVolumeClaim {
	i := slices.IndexFunc(items, func(pvc corev1.PersistentVolumeClaim) bool {
		return pvc.Name == item.Name
	})
	if i < 0 {
		return append(items, item)
	}
	return append(append(items[:i], item), items[i+1:]...)
}

// EnsureNoPVCWithName ensures that a PVC with the given name does not exist in the given slice.
func EnsureNoPVCWithName(items []corev1.PersistentVolumeClaim, name string) []corev1.PersistentVolumeClaim {
	return slices.DeleteFunc(items, func(pvc corev1.PersistentVolumeClaim) bool {
		return pvc.Name == name
	})
}

// EnsureUnitOption ensures the given unit option exist in the given slice.
func EnsureUnitOption(items []*unit.UnitOption, item *unit.UnitOption) []*unit.UnitOption {
	i := slices.IndexFunc(items, func(uo *unit.UnitOption) bool {
		return reflect.DeepEqual(uo, item)
	})
	if i < 0 {
		return append(items, item)
	}
	return items
}

// EnsureFileWithPath ensures that a file with a path equal to the path of the given file exists in the given slice
// and is equal to the given file.
func EnsureFileWithPath(items []extensionsv1alpha1.File, item extensionsv1alpha1.File) []extensionsv1alpha1.File {
	if i := fileWithPathIndex(items, item.Path); i < 0 {
		items = append(items, item)
	} else if !reflect.DeepEqual(items[i], item) {
		items[i] = item
	}
	return items
}

// EnsureUnitWithName ensures that an unit with a name equal to the name of the given unit exists in the given slice
// and is equal to the given unit.
func EnsureUnitWithName(items []extensionsv1alpha1.Unit, item extensionsv1alpha1.Unit) []extensionsv1alpha1.Unit {
	if i := unitWithNameIndex(items, item.Name); i < 0 {
		items = append(items, item)
	} else if !reflect.DeepEqual(items[i], item) {
		items[i] = item
	}
	return append(append(items[:i], item), items[i+1:]...)
}

// EnsureAnnotationOrLabel ensures the given key/value exists in the annotationOrLabelMap map.
func EnsureAnnotationOrLabel(annotationOrLabelMap map[string]string, key, value string) map[string]string {
	if annotationOrLabelMap == nil {
		annotationOrLabelMap = make(map[string]string, 1)
	}
	annotationOrLabelMap[key] = value
	return annotationOrLabelMap
}

// StringIndex returns the index of the first occurrence of the given string in the given slice, or -1 if not found.
func StringIndex(items []string, value string) int {
	return slices.IndexFunc(items, func(s string) bool {
		return s == value
	})
}

// StringWithPrefixIndex returns the index of the first occurrence of a string having the given prefix in the given slice, or -1 if not found.
func StringWithPrefixIndex(items []string, prefix string) int {
	return slices.IndexFunc(items, func(s string) bool {
		return strings.HasPrefix(s, prefix)
	})
}
