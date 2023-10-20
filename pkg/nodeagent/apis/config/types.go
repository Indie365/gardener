// Copyright 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package config

import (
	"github.com/Masterminds/semver/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	componentbaseconfig "k8s.io/component-base/config"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeAgentConfiguration defines the configuration for the gardener-node-agent.
type NodeAgentConfiguration struct {
	metav1.TypeMeta
	// ClientConnection specifies the kubeconfig file and the client connection settings for the proxy server to use
	// when communicating with the kube-apiserver of the shoot cluster.
	ClientConnection componentbaseconfig.ClientConnectionConfiguration
	// LogLevel is the level/severity for the logs. Must be one of [info,debug,error].
	LogLevel string
	// LogFormat is the output format for the logs. Must be one of [text,json].
	LogFormat string
	// Server defines the configuration of the HTTP server.
	Server ServerConfiguration
	// Debugging holds configuration for Debugging related features.
	Debugging *componentbaseconfig.DebuggingConfiguration
	// FeatureGates is a map of feature names to bools that enable or disable alpha/experimental features. This field
	// modifies piecemeal the built-in default values from "github.com/gardener/gardener/pkg/operator/features/features.go".
	// Default: nil
	FeatureGates map[string]bool
	// Bootstrap contains configuration for the bootstrap command.
	Bootstrap *BootstrapConfiguration
	// Controllers defines the configuration of the controllers.
	Controllers ControllerConfiguration
}

// BootstrapConfiguration contains configuration for the bootstrap command.
type BootstrapConfiguration struct {
	// KubeletDataVolumeSize sets the data volume size of an unformatted disk on the worker node, which is used for
	// /var/lib on the worker.
	KubeletDataVolumeSize *int64
}

// ControllerConfiguration defines the configuration of the controllers.
type ControllerConfiguration struct {
	// KubeletUpgrade is the configuration for the kubelet upgrade controller.
	KubeletUpgrade KubeletUpgradeControllerConfig
	// OperatingSystemConfig is the configuration for the operating system config controller.
	OperatingSystemConfig OperatingSystemConfigControllerConfig
	// SelfUpgrade is the configuration for the self-upgrade controller.
	SelfUpgrade SelfUpgradeControllerConfig
	// Token is the configuration for the access token controller.
	Token TokenControllerConfig
}

// OperatingSystemConfigControllerConfig defines the configuration of the operating system config controller.
type OperatingSystemConfigControllerConfig struct {
	// SyncPeriod is the duration how often the operating system config is applied.
	SyncPeriod *metav1.Duration
	// SyncJitterPeriod is a jitter duration for the reconciler sync that can be used to distribute the syncs randomly.
	// If its value is greater than 0 then the OSC secret will not be enqueued immediately but only after a random
	// duration between 0 and the configured value. It is defaulted to 5m.
	SyncJitterPeriod *metav1.Duration
	// SecretName defines the name of the secret in the shoot cluster control plane, which contains the operating system
	// config (OSC) for the gardener-node-agent.
	SecretName string
	// KubernetesVersion contains the Kubernetes version of the kubelet, used for annotating the corresponding node
	// resource with a kubernetes version annotation.
	KubernetesVersion *semver.Version
}

// KubeletUpgradeControllerConfig defines the configuration of the kubelet upgrade controller.
type KubeletUpgradeControllerConfig struct {
	// Image is the container image reference to the image containing the kubelet binary (hyperkube image).
	Image string
}

// SelfUpgradeControllerConfig defines the configuration of the self-upgrade controller.
type SelfUpgradeControllerConfig struct {
	// Image is the container image reference to the gardener-node-agent.
	Image string
}

// TokenControllerConfig defines the configuration of the access token controller.
type TokenControllerConfig struct {
	// SecretName defines the name of the secret in the shoot cluster control plane, which contains the `kube-apiserver`
	// access token for the gardener-node-agent.
	SecretName string
}

// ServerConfiguration contains details for the HTTP(S) servers.
type ServerConfiguration struct {
	// HealthProbes is the configuration for serving the healthz and readyz endpoints.
	HealthProbes *Server
	// Metrics is the configuration for serving the metrics endpoint.
	Metrics *Server
}

// Server contains information for HTTP(S) server configuration.
type Server struct {
	// BindAddress is the IP address on which to listen for the specified port.
	BindAddress string
	// Port is the port on which to serve requests.
	Port int
}
