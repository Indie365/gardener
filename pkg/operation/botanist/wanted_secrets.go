// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package botanist

import (
	"fmt"
	"net"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"github.com/gardener/gardener/pkg/operation/botanist/component/dependencywatchdog"
	"github.com/gardener/gardener/pkg/operation/botanist/component/etcd"
	"github.com/gardener/gardener/pkg/operation/botanist/component/kubeapiserver"
	"github.com/gardener/gardener/pkg/operation/botanist/component/resourcemanager"
	"github.com/gardener/gardener/pkg/operation/botanist/component/vpnseedserver"
	"github.com/gardener/gardener/pkg/operation/botanist/component/vpnshoot"
	"github.com/gardener/gardener/pkg/operation/common"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	"github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/utils/secrets"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// generateWantedSecrets returns a list of Secret configuration objects satisfying the secret config interface,
// each containing their specific configuration for the creation of certificates (server/client), RSA key pairs, basic
// authentication credentials, etc.
func (b *Botanist) generateWantedSecretConfigs(certificateAuthorities map[string]*secrets.Certificate) ([]secrets.ConfigInterface, error) {
	var (
		apiServerIPAddresses = []net.IP{
			net.ParseIP("127.0.0.1"),
			b.Shoot.Networks.APIServer,
		}
		apiServerCertDNSNames = append([]string{
			v1beta1constants.DeploymentNameKubeAPIServer,
			fmt.Sprintf("%s.%s", v1beta1constants.DeploymentNameKubeAPIServer, b.Shoot.SeedNamespace),
			fmt.Sprintf("%s.%s.svc", v1beta1constants.DeploymentNameKubeAPIServer, b.Shoot.SeedNamespace),
			gutil.GetAPIServerDomain(b.Shoot.InternalClusterDomain),
			b.Shoot.GetInfo().Status.TechnicalID,
		}, kubernetes.DNSNamesForService("kubernetes", metav1.NamespaceDefault)...)

		gardenerResourceManagerCertDNSNames = kubernetes.DNSNamesForService(resourcemanager.ServiceName, b.Shoot.SeedNamespace)

		etcdCertDNSNames = append(
			b.Shoot.Components.ControlPlane.EtcdMain.ServiceDNSNames(),
			b.Shoot.Components.ControlPlane.EtcdEvents.ServiceDNSNames()...,
		)

		endUserCrtValidity = common.EndUserCrtValidity
	)

	if !b.Seed.GetInfo().Spec.Settings.ShootDNS.Enabled {
		if addr := net.ParseIP(b.APIServerAddress); addr != nil {
			apiServerIPAddresses = append(apiServerIPAddresses, addr)
		} else {
			apiServerCertDNSNames = append(apiServerCertDNSNames, b.APIServerAddress)
		}
	}

	if b.Shoot.ExternalClusterDomain != nil {
		apiServerCertDNSNames = append(apiServerCertDNSNames, *(b.Shoot.GetInfo().Spec.DNS.Domain), gutil.GetAPIServerDomain(*b.Shoot.ExternalClusterDomain))
	}

	secretList := []secrets.ConfigInterface{
		// Secret definition for kube-apiserver
		&secrets.ControlPlaneSecretConfig{
			Name: kubeapiserver.SecretNameServer,
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				CommonName:   v1beta1constants.DeploymentNameKubeAPIServer,
				Organization: nil,
				DNSNames:     apiServerCertDNSNames,
				IPAddresses:  apiServerIPAddresses,

				CertType:  secrets.ServerCert,
				SigningCA: certificateAuthorities[v1beta1constants.SecretNameCACluster],
			},
		},
		// Secret definition for kube-apiserver to kubelets communication
		&secrets.ControlPlaneSecretConfig{
			Name: kubeapiserver.SecretNameKubeAPIServerToKubelet,
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				CommonName:   kubeapiserver.UserName,
				Organization: nil,
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[v1beta1constants.SecretNameCAKubelet],
			},
		},

		// Secret definition for kube-aggregator
		&secrets.ControlPlaneSecretConfig{
			Name: kubeapiserver.SecretNameKubeAggregator,
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				CommonName:   "system:kube-aggregator",
				Organization: nil,
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[v1beta1constants.SecretNameCAFrontProxy],
			},
		},

		// Secret definition for gardener-resource-manager server
		&secrets.CertificateSecretConfig{
			Name: resourcemanager.SecretNameServer,

			CommonName:   v1beta1constants.DeploymentNameGardenerResourceManager,
			Organization: nil,
			DNSNames:     gardenerResourceManagerCertDNSNames,
			IPAddresses:  nil,

			CertType:  secrets.ServerCert,
			SigningCA: certificateAuthorities[v1beta1constants.SecretNameCACluster],
		},

		// Secret definition for prometheus
		// TODO(rfranzke): Delete this in a future release once all monitoring configurations of extensions have been
		// adapted.
		&secrets.ControlPlaneSecretConfig{
			Name: "prometheus",
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				CommonName:   "gardener.cloud:monitoring:prometheus",
				Organization: []string{"gardener.cloud:monitoring"},
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[v1beta1constants.SecretNameCACluster],
			},

			KubeConfigRequests: []secrets.KubeConfigRequest{{
				ClusterName:   b.Shoot.SeedNamespace,
				APIServerHost: b.Shoot.ComputeInClusterAPIServerAddress(true),
			}},
		},

		// Secret definition for prometheus to kubelets communication
		&secrets.ControlPlaneSecretConfig{
			Name: "prometheus-kubelet",
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				CommonName:   "gardener.cloud:monitoring:prometheus",
				Organization: []string{"gardener.cloud:monitoring"},
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[v1beta1constants.SecretNameCAKubelet],
			},
		},

		// Secret definition for monitoring
		&secrets.BasicAuthSecretConfig{
			Name:   common.MonitoringIngressCredentials,
			Format: secrets.BasicAuthFormatNormal,

			Username:       "admin",
			PasswordLength: 32,
		},

		// Secret definition for monitoring for shoot owners
		&secrets.BasicAuthSecretConfig{
			Name:   common.MonitoringIngressCredentialsUsers,
			Format: secrets.BasicAuthFormatNormal,

			Username:       "admin",
			PasswordLength: 32,
		},

		// Secret definition for service-account-key
		&secrets.RSASecretConfig{
			Name:       v1beta1constants.SecretNameServiceAccountKey,
			Bits:       4096,
			UsedForSSH: false,
		},

		// Secret definition for etcd server
		&secrets.CertificateSecretConfig{
			Name: etcd.SecretNameServer,

			CommonName:   "etcd-server",
			Organization: nil,
			DNSNames:     etcdCertDNSNames,
			IPAddresses:  nil,

			CertType:  secrets.ServerClientCert,
			SigningCA: certificateAuthorities[v1beta1constants.SecretNameCAETCD],
		},

		// Secret definition for etcd server
		&secrets.CertificateSecretConfig{
			Name: etcd.SecretNameClient,

			CommonName:   "etcd-client",
			Organization: nil,
			DNSNames:     nil,
			IPAddresses:  nil,

			CertType:  secrets.ClientCert,
			SigningCA: certificateAuthorities[v1beta1constants.SecretNameCAETCD],
		},

		// Secret definition for alertmanager (ingress)
		&secrets.CertificateSecretConfig{
			Name: common.AlertManagerTLS,

			CommonName:   "alertmanager",
			Organization: []string{"gardener.cloud:monitoring:ingress"},
			DNSNames:     b.ComputeAlertManagerHosts(),
			IPAddresses:  nil,

			CertType:  secrets.ServerCert,
			SigningCA: certificateAuthorities[v1beta1constants.SecretNameCACluster],
			Validity:  &endUserCrtValidity,
		},

		// Secret definition for grafana (ingress)
		&secrets.CertificateSecretConfig{
			Name: common.GrafanaTLS,

			CommonName:   "grafana",
			Organization: []string{"gardener.cloud:monitoring:ingress"},
			DNSNames:     b.ComputeGrafanaHosts(),
			IPAddresses:  nil,

			CertType:  secrets.ServerCert,
			SigningCA: certificateAuthorities[v1beta1constants.SecretNameCACluster],
			Validity:  &endUserCrtValidity,
		},

		// Secret definition for prometheus (ingress)
		&secrets.CertificateSecretConfig{
			Name: common.PrometheusTLS,

			CommonName:   "prometheus",
			Organization: []string{"gardener.cloud:monitoring:ingress"},
			DNSNames:     b.ComputePrometheusHosts(),
			IPAddresses:  nil,

			CertType:  secrets.ServerCert,
			SigningCA: certificateAuthorities[v1beta1constants.SecretNameCACluster],
			Validity:  &endUserCrtValidity,
		},
	}

	if b.isShootNodeLoggingEnabled() {
		// Secret definition for loki (ingress)
		secretList = append(secretList, &secrets.CertificateSecretConfig{
			Name: common.LokiTLS,

			CommonName:   b.ComputeLokiHost(),
			Organization: []string{"gardener.cloud:monitoring:ingress"},
			DNSNames:     b.ComputeLokiHosts(),
			IPAddresses:  nil,

			CertType:  secrets.ServerCert,
			SigningCA: certificateAuthorities[v1beta1constants.SecretNameCACluster],
			Validity:  &endUserCrtValidity,
		})
	}

	if gardencorev1beta1helper.SeedSettingDependencyWatchdogProbeEnabled(b.Seed.GetInfo().Spec.Settings) {
		// Secret definitions for dependency-watchdog-internal and external probes
		secretList = append(secretList, &secrets.ControlPlaneSecretConfig{
			Name: kubeapiserver.DependencyWatchdogInternalProbeSecretName,
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				CommonName:   dependencywatchdog.UserName,
				Organization: nil,
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[v1beta1constants.SecretNameCACluster],
			},
			KubeConfigRequests: []secrets.KubeConfigRequest{{
				ClusterName:   b.Shoot.SeedNamespace,
				APIServerHost: b.Shoot.ComputeInClusterAPIServerAddress(false),
			}},
		}, &secrets.ControlPlaneSecretConfig{
			Name: kubeapiserver.DependencyWatchdogExternalProbeSecretName,
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				CommonName:   dependencywatchdog.UserName,
				Organization: nil,
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[v1beta1constants.SecretNameCACluster],
			},
			KubeConfigRequests: []secrets.KubeConfigRequest{{
				ClusterName:   b.Shoot.SeedNamespace,
				APIServerHost: b.Shoot.ComputeOutOfClusterAPIServerAddress(b.APIServerAddress, true),
			}},
		})
	}

	if b.Shoot.ReversedVPNEnabled {
		secretList = append(secretList,
			// Secret definition for vpn-shoot (OpenVPN client side)
			&secrets.CertificateSecretConfig{
				Name:       vpnshoot.SecretNameVPNShootClient,
				CommonName: "vpn-shoot-client",
				CertType:   secrets.ClientCert,
				SigningCA:  certificateAuthorities[v1beta1constants.SecretNameCAVPN],
			},

			// Secret definition for vpn-seed-server (OpenVPN server side)
			&secrets.CertificateSecretConfig{
				Name:       "vpn-seed-server",
				CommonName: "vpn-seed-server",
				DNSNames:   kubernetes.DNSNamesForService(vpnseedserver.ServiceName, b.Shoot.SeedNamespace),
				CertType:   secrets.ServerCert,
				SigningCA:  certificateAuthorities[v1beta1constants.SecretNameCAVPN],
			},

			&secrets.VPNTLSAuthConfig{
				Name: vpnseedserver.VpnSeedServerTLSAuth,
			},

			// Secret definition for kube-apiserver http proxy client
			&secrets.CertificateSecretConfig{
				Name:       kubeapiserver.SecretNameHTTPProxy,
				CommonName: "kube-apiserver-http-proxy",
				CertType:   secrets.ClientCert,
				SigningCA:  certificateAuthorities[v1beta1constants.SecretNameCAVPN],
			},
		)
	} else {
		secretList = append(secretList,
			// Secret definition for vpn-shoot (OpenVPN server side)
			&secrets.CertificateSecretConfig{
				Name:       vpnshoot.SecretNameVPNShoot,
				CommonName: "vpn-shoot",
				CertType:   secrets.ServerCert,
				SigningCA:  certificateAuthorities[v1beta1constants.SecretNameCACluster],
			},

			// Secret definition for vpn-seed (OpenVPN client side)
			&secrets.CertificateSecretConfig{
				Name:       kubeapiserver.SecretNameVPNSeed,
				CommonName: kubeapiserver.UserNameVPNSeed,
				CertType:   secrets.ClientCert,
				SigningCA:  certificateAuthorities[v1beta1constants.SecretNameCACluster],
			},

			&secrets.VPNTLSAuthConfig{
				Name: kubeapiserver.SecretNameVPNSeedTLSAuth,
			},
		)
	}

	if b.Shoot.WantsVerticalPodAutoscaler {
		var (
			commonName = fmt.Sprintf("vpa-webhook.%s.svc", b.Shoot.SeedNamespace)
			dnsNames   = []string{
				"vpa-webhook",
				fmt.Sprintf("vpa-webhook.%s", b.Shoot.SeedNamespace),
				commonName,
			}
		)

		secretList = append(secretList, &secrets.CertificateSecretConfig{
			Name:       common.VPASecretName,
			CommonName: commonName,
			DNSNames:   dnsNames,
			CertType:   secrets.ServerCert,
			SigningCA:  certificateAuthorities[v1beta1constants.SecretNameCACluster],
		})
	}

	return secretList, nil
}
