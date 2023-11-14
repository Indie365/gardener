//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by defaulter-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// RegisterDefaults adds defaulters functions to the given scheme.
// Public to allow building arbitrary schemes.
// All generated defaulters are covering - they call all nested defaulters.
func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&ClusterOpenIDConnectPreset{}, func(obj interface{}) { SetObjectDefaults_ClusterOpenIDConnectPreset(obj.(*ClusterOpenIDConnectPreset)) })
	scheme.AddTypeDefaultingFunc(&ClusterOpenIDConnectPresetList{}, func(obj interface{}) {
		SetObjectDefaults_ClusterOpenIDConnectPresetList(obj.(*ClusterOpenIDConnectPresetList))
	})
	scheme.AddTypeDefaultingFunc(&OpenIDConnectPreset{}, func(obj interface{}) { SetObjectDefaults_OpenIDConnectPreset(obj.(*OpenIDConnectPreset)) })
	scheme.AddTypeDefaultingFunc(&OpenIDConnectPresetList{}, func(obj interface{}) { SetObjectDefaults_OpenIDConnectPresetList(obj.(*OpenIDConnectPresetList)) })
	return nil
}

func SetObjectDefaults_ClusterOpenIDConnectPreset(in *ClusterOpenIDConnectPreset) {
	SetDefaults_ClusterOpenIDConnectPreset(in)
	SetDefaults_OpenIDConnectPresetSpec(&in.Spec.OpenIDConnectPresetSpec)
	SetDefaults_KubeAPIServerOpenIDConnect(&in.Spec.OpenIDConnectPresetSpec.Server)
}

func SetObjectDefaults_ClusterOpenIDConnectPresetList(in *ClusterOpenIDConnectPresetList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_ClusterOpenIDConnectPreset(a)
	}
}

func SetObjectDefaults_OpenIDConnectPreset(in *OpenIDConnectPreset) {
	SetDefaults_OpenIDConnectPresetSpec(&in.Spec)
	SetDefaults_KubeAPIServerOpenIDConnect(&in.Spec.Server)
}

func SetObjectDefaults_OpenIDConnectPresetList(in *OpenIDConnectPresetList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_OpenIDConnectPreset(a)
	}
}
