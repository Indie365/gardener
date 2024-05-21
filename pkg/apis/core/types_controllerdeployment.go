// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package core

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControllerDeployment contains information about how this controller is deployed.
type ControllerDeployment struct {
	metav1.TypeMeta
	// Standard object metadata.
	metav1.ObjectMeta
	// Type is the deployment type.
	// This field correlates with the Type field in the v1beta1 API version.
	// It is only set if a custom type (other than helm) is configured in the v1beta1 API version and the object is
	// converted to the internal version. If the helm type is used in v1beta1, the Helm section will be set in the
	// internal API version instead of this field. In the v1 API version, the value is represented using an annotation.
	Type string
	// ProviderConfig contains type-specific configuration. It contains assets that deploy the controller.
	// This field correlates with the ProviderConfig field in the v1beta1 API version.
	// It is only set if a custom type (other than helm) is configured in the v1beta1 API version and the object is
	// converted to the internal version. If the helm type is used in v1beta1, the Helm section will be set in the
	// internal API version instead of this field. In the v1 API version, the value is represented using an annotation.
	ProviderConfig runtime.Object
	// Helm configures that an extension controller is deployed using helm.
	Helm *HelmControllerDeployment
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControllerDeploymentList is a collection of ControllerDeployments.
type ControllerDeploymentList struct {
	metav1.TypeMeta
	// Standard list object metadata.
	metav1.ListMeta
	// Items is the list of ControllerDeployments.
	Items []ControllerDeployment
}

// HelmControllerDeployment configures how an extension controller is deployed using helm.
type HelmControllerDeployment struct {
	// RawChart is the base64-encoded, gzip'ed, tar'ed extension controller chart.
	RawChart []byte
	// Values are the chart values.
	Values *apiextensionsv1.JSON
}
