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

package kubeapiserver

import (
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// SecretNameBasicAuth is the name of the secret containing basic authentication credentials for the kube-apiserver.
	SecretNameBasicAuth = "kube-apiserver-basic-auth"
	// SecretNameEtcdEncryption is the name of the secret which contains the EncryptionConfiguration. The
	// EncryptionConfiguration contains a key which the kube-apiserver uses for encrypting selected etcd content.
	SecretNameEtcdEncryption = "etcd-encryption-secret"
	// SecretNameKubeAggregator is the name of the secret for the kube-aggregator when talking to the kube-apiserver.
	SecretNameKubeAggregator = "kube-aggregator"
	// SecretNameKubeAPIServerToKubelet is the name of the secret for the kube-apiserver credentials when talking to
	// kubelets.
	SecretNameKubeAPIServerToKubelet = "kube-apiserver-kubelet"
	// SecretNameServer is the name of the secret for the kube-apiserver server certificates.
	SecretNameServer = "kube-apiserver"
	// SecretNameStaticToken is the name of the secret containing static tokens for the kube-apiserver.
	SecretNameStaticToken = "static-token"
	// SecretNameVPNSeed is the name of the secret containing the certificates for the vpn-seed.
	SecretNameVPNSeed = "vpn-seed"
	// SecretNameVPNSeedTLSAuth is the name of the secret containing the TLS auth for the vpn-seed.
	SecretNameVPNSeedTLSAuth = "vpn-seed-tlsauth"

	containerNameKubeAPIServer            = "kube-apiserver"
	containerNameVPNSeed                  = "vpn-seed"
	containerNameAPIServerProxyPodMutator = "apiserver-proxy-pod-mutator"

	volumeMountPathAdmissionConfiguration = "/etc/kubernetes/admission"
	volumeMountPathHTTPProxy              = "/etc/srv/kubernetes/envoy"
)

func (k *kubeAPIServer) emptyDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: v1beta1constants.DeploymentNameKubeAPIServer, Namespace: k.namespace}}
}
