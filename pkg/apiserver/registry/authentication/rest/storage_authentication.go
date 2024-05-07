// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package rest

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"

	"github.com/gardener/gardener/pkg/api"
	"github.com/gardener/gardener/pkg/apis/authentication"
	authenticationv1alpha1 "github.com/gardener/gardener/pkg/apis/authentication/v1alpha1"
	credentialsbindingstore "github.com/gardener/gardener/pkg/apiserver/registry/authentication/credentialsbinding/storage"
)

// StorageProvider is an empty struct.
type StorageProvider struct{}

// NewRESTStorage creates a new API group info object and registers the v1alpha1 Garden storage.
func (p StorageProvider) NewRESTStorage(restOptionsGetter generic.RESTOptionsGetter) genericapiserver.APIGroupInfo {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(authentication.GroupName, api.Scheme, metav1.ParameterCodec, api.Codecs)
	apiGroupInfo.VersionedResourcesStorageMap[authenticationv1alpha1.SchemeGroupVersion.Version] = p.v1alpha1Storage(restOptionsGetter)
	return apiGroupInfo
}

// GroupName returns the garden group name.
func (p StorageProvider) GroupName() string {
	return authentication.GroupName
}

func (p StorageProvider) v1alpha1Storage(restOptionsGetter generic.RESTOptionsGetter) map[string]rest.Storage {
	storage := map[string]rest.Storage{}

	credentialsBindingStorage := credentialsbindingstore.NewStorage(restOptionsGetter)
	storage["credentialsbindings"] = credentialsBindingStorage.CredentialsBinding

	return storage
}
