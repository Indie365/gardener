// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package admissionplugins

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"

	"github.com/gardener/gardener/pkg/apis/core"
	versionutils "github.com/gardener/gardener/pkg/utils/version"
)

// admissionPluginsVersionRanges contains the version ranges for all Kubernetes admission plugins.
// Extracted from https://raw.githubusercontent.com/kubernetes/kubernetes/release-${version}/pkg/kubeapiserver/options/plugins.go
// and https://raw.githubusercontent.com/kubernetes/kubernetes/release-${version}/staging/src/k8s.io/apiserver/pkg/server/plugins.go.
// To maintain this list for each new Kubernetes version:
//   - Run hack/compare-k8s-admission-plugins.sh <old-version> <new-version> (e.g. 'hack/compare-k8s-admission-plugins.sh 1.22 1.23').
//     It will present 2 lists of admission plugins: those added and those removed in <new-version> compared to <old-version> and
//   - Add all added admission plugins to the map with <new-version> as AddedInVersion and no RemovedInVersion.
//   - For any removed admission plugin, add <new-version> as RemovedInVersion to the already existing admission plugin in the map.

var admissionPluginsVersionRanges = map[string]*AdmissionPluginVersionRange{
	"AlwaysAdmit":                          {},
	"AlwaysDeny":                           {},
	"AlwaysPullImages":                     {},
	"CertificateApproval":                  {AddedInVersion: "1.18"},
	"CertificateSigning":                   {AddedInVersion: "1.18"},
	"CertificateSubjectRestriction":        {AddedInVersion: "1.18"},
	"DefaultIngressClass":                  {AddedInVersion: "1.18"},
	"DefaultStorageClass":                  {},
	"DefaultTolerationSeconds":             {},
	"DenyEscalatingExec":                   {RemovedInVersion: "1.21"},
	"DenyExecOnPrivileged":                 {RemovedInVersion: "1.21"},
	"DenyServiceExternalIPs":               {AddedInVersion: "1.21"},
	"EventRateLimit":                       {},
	"ExtendedResourceToleration":           {},
	"ImagePolicyWebhook":                   {},
	"LimitPodHardAntiAffinityTopology":     {},
	"LimitRanger":                          {},
	"MutatingAdmissionWebhook":             {Required: true},
	"NamespaceAutoProvision":               {},
	"NamespaceExists":                      {},
	"NamespaceLifecycle":                   {Required: true},
	"NodeRestriction":                      {Required: true},
	"OwnerReferencesPermissionEnforcement": {},
	"PersistentVolumeClaimResize":          {},
	"PersistentVolumeLabel":                {},
	"PodNodeSelector":                      {},
	"PodPreset":                            {RemovedInVersion: "1.20"},
	"PodSecurity":                          {AddedInVersion: "1.22", Required: true},
	"PodSecurityPolicy":                    {RemovedInVersion: "1.25"},
	"PodTolerationRestriction":             {},
	"Priority":                             {Required: true},
	"ResourceQuota":                        {},
	"RuntimeClass":                         {},
	"SecurityContextDeny":                  {Forbidden: true},
	"ServiceAccount":                       {},
	"StorageObjectInUseProtection":         {Required: true},
	"TaintNodesByCondition":                {},
	"ValidatingAdmissionWebhook":           {Required: true},
}

// IsAdmissionPluginSupported returns true if the given admission plugin is supported for the given Kubernetes version.
// An admission plugin is only supported if it's a known admission plugin and its version range contains the given Kubernetes version.
func IsAdmissionPluginSupported(plugin, version string) (bool, error) {
	vr := admissionPluginsVersionRanges[plugin]
	if vr == nil {
		return false, fmt.Errorf("unknown admission plugin %q", plugin)
	}
	return vr.Contains(version)
}

// AdmissionPluginVersionRange represents a version range of type [AddedInVersion, RemovedInVersion).
type AdmissionPluginVersionRange struct {
	Forbidden        bool
	Required         bool
	AddedInVersion   string
	RemovedInVersion string
}

// Contains returns true if the range contains the given version, false otherwise.
// The range contains the given version only if it's greater or equal than AddedInVersion (always true if AddedInVersion is empty),
// and less than RemovedInVersion (always true if RemovedInVersion is empty).
func (r *AdmissionPluginVersionRange) Contains(version string) (bool, error) {
	var constraint string
	switch {
	case r.AddedInVersion != "" && r.RemovedInVersion == "":
		constraint = fmt.Sprintf(">= %s", r.AddedInVersion)
	case r.AddedInVersion == "" && r.RemovedInVersion != "":
		constraint = fmt.Sprintf("< %s", r.RemovedInVersion)
	case r.AddedInVersion != "" && r.RemovedInVersion != "":
		constraint = fmt.Sprintf(">= %s, < %s", r.AddedInVersion, r.RemovedInVersion)
	default:
		constraint = "*"
	}
	return versionutils.CheckVersionMeetsConstraint(version, constraint)
}

func getAllForbiddenPlugins() []string {
	var allForbiddenPlugins []string
	for name, vr := range admissionPluginsVersionRanges {
		if vr.Forbidden {
			allForbiddenPlugins = append(allForbiddenPlugins, name)
		}
	}
	return allForbiddenPlugins
}

// ValidateAdmissionPlugins validates the given Kubernetes admission plugins against the given Kubernetes version.
func ValidateAdmissionPlugins(admissionPlugins []core.AdmissionPlugin, version string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for i, plugin := range admissionPlugins {
		idxPath := fldPath.Index(i)

		if len(plugin.Name) == 0 {
			allErrs = append(allErrs, field.Required(idxPath.Child("name"), "must provide a name"))
			return allErrs
		}

		supported, err := IsAdmissionPluginSupported(plugin.Name, version)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(idxPath.Child("name"), plugin.Name, err.Error()))
		} else if !supported && !pointer.BoolDeref(plugin.Disabled, false) {
			allErrs = append(allErrs, field.Forbidden(idxPath.Child("name"), fmt.Sprintf("admission plugin %q is not supported in Kubernetes version %s", plugin.Name, version)))
		} else {
			if admissionPluginsVersionRanges[plugin.Name].Forbidden {
				allErrs = append(allErrs, field.Forbidden(idxPath.Child("name"), fmt.Sprintf("forbidden admission plugin was specified - do not use plugins from the following list: %+v", getAllForbiddenPlugins())))
			}
			if pointer.BoolDeref(plugin.Disabled, false) && admissionPluginsVersionRanges[plugin.Name].Required {
				allErrs = append(allErrs, field.Forbidden(idxPath, fmt.Sprintf("admission plugin %q cannot be disabled", plugin.Name)))
			}
		}
	}

	return allErrs
}
