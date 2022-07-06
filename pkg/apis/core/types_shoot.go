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

package core

import (
	"time"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Shoot represents a Shoot cluster created and managed by Gardener.
type Shoot struct {
	metav1.TypeMeta
	// Standard object metadata.
	metav1.ObjectMeta
	// Specification of the Shoot cluster.
	// If the object's deletion timestamp is set, this field is immutable.
	Spec ShootSpec
	// Most recently observed status of the Shoot cluster.
	Status ShootStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ShootList is a list of Shoot objects.
type ShootList struct {
	metav1.TypeMeta
	// Standard list object metadata.
	metav1.ListMeta
	// Items is the list of Shoots.
	Items []Shoot
}

// ShootTemplate is a template for creating a Shoot object.
type ShootTemplate struct {
	// Standard object metadata.
	metav1.ObjectMeta
	// Specification of the desired behavior of the Shoot.
	Spec ShootSpec
}

// ShootSpec is the specification of a Shoot.
type ShootSpec struct {
	// Addons contains information about enabled/disabled addons and their configuration.
	Addons *Addons
	// CloudProfileName is a name of a CloudProfile object. This field is immutable.
	CloudProfileName string
	// DNS contains information about the DNS settings of the Shoot.
	DNS *DNS
	// Extensions contain type and provider information for Shoot extensions.
	Extensions []Extension
	// Hibernation contains information whether the Shoot is suspended or not.
	Hibernation *Hibernation
	// Kubernetes contains the version and configuration settings of the control plane components.
	Kubernetes Kubernetes
	// Networking contains information about cluster networking such as CNI Plugin type, CIDRs, ...etc.
	Networking Networking
	// Maintenance contains information about the time window for maintenance operations and which
	// operations should be performed.
	Maintenance *Maintenance
	// Monitoring contains information about custom monitoring configurations for the shoot.
	Monitoring *Monitoring
	// Provider contains all provider-specific and provider-relevant information.
	Provider Provider
	// Purpose is the purpose class for this cluster.
	Purpose *ShootPurpose
	// Region is a name of a region. This field is immutable.
	Region string
	// SecretBindingName is the name of the a SecretBinding that has a reference to the provider secret.
	// The credentials inside the provider secret will be used to create the shoot in the respective account.
	// This field is immutable.
	SecretBindingName string
	// SeedName is the name of the seed cluster that runs the control plane of the Shoot.
	// This field is immutable when the SeedChange feature gate is disabled.
	SeedName *string
	// SeedSelector is an optional selector which must match a seed's labels for the shoot to be scheduled on that seed.
	SeedSelector *SeedSelector
	// Resources holds a list of named resource references that can be referred to in extension configs by their names.
	Resources []NamedResourceReference
	// Tolerations contains the tolerations for taints on seed clusters.
	Tolerations []Toleration
	// ExposureClassName is the optional name of an exposure class to apply a control plane endpoint exposure strategy.
	// This field is immutable.
	ExposureClassName *string
	// SystemComponents contains the settings of system components in the control or data plane of the Shoot cluster.
	SystemComponents *SystemComponents
}

// GetProviderType gets the type of the provider.
func (s *Shoot) GetProviderType() string {
	return s.Spec.Provider.Type
}

// ShootStatus holds the most recently observed status of the Shoot cluster.
type ShootStatus struct {
	// Conditions represents the latest available observations of a Shoots's current state.
	Conditions []Condition
	// Constraints represents conditions of a Shoot's current state that constraint some operations on it.
	Constraints []Condition
	// Gardener holds information about the Gardener which last acted on the Shoot.
	Gardener Gardener
	// IsHibernated indicates whether the Shoot is currently hibernated.
	IsHibernated bool
	// LastHibernationTriggerTime indicates the last time when the hibernation controller
	// managed to change the hibernation settings of the cluster
	LastHibernationTriggerTime *metav1.Time
	// LastOperation holds information about the last operation on the Shoot.
	LastOperation *LastOperation
	// LastErrors holds information about the last occurred error(s) during an operation.
	LastErrors []LastError
	// ObservedGeneration is the most recent generation observed for this Shoot. It corresponds to the
	// Shoot's generation, which is updated on mutation by the API Server.
	ObservedGeneration int64
	// RetryCycleStartTime is the start time of the last retry cycle (used to determine how often an operation
	// must be retried until we give up).
	RetryCycleStartTime *metav1.Time
	// SeedName is the name of the seed cluster that runs the control plane of the Shoot. This value is only written
	// after a successful create/reconcile operation. It will be used when control planes are moved between Seeds.
	SeedName *string
	// TechnicalID is the name that is used for creating the Seed namespace, the infrastructure resources, and
	// basically everything that is related to this particular Shoot. This field is immutable.
	TechnicalID string
	// UID is a unique identifier for the Shoot cluster to avoid portability between Kubernetes clusters.
	// It is used to compute unique hashes. This field is immutable.
	UID types.UID
	// ClusterIdentity is the identity of the Shoot cluster. This field is immutable.
	ClusterIdentity *string
	// List of addresses on which the Kube API server can be reached.
	AdvertisedAddresses []ShootAdvertisedAddress
	// MigrationStartTime is the time when a migration to a different seed was initiated.
	MigrationStartTime *metav1.Time
	// Credentials contains information about the shoot credentials.
	Credentials *ShootCredentials
}

// ShootCredentials contains information about the shoot credentials.
type ShootCredentials struct {
	// Rotation contains information about the credential rotations.
	Rotation *ShootCredentialsRotation
}

// ShootCredentialsRotation contains information about the rotation of credentials.
type ShootCredentialsRotation struct {
	// CertificateAuthorities contains information about the certificate authority credential rotation.
	CertificateAuthorities *ShootCARotation
	// Kubeconfig contains information about the kubeconfig credential rotation.
	Kubeconfig *ShootKubeconfigRotation
	// SSHKeypair contains information about the ssh-keypair credential rotation.
	SSHKeypair *ShootSSHKeypairRotation
	// Observability contains information about the observability credential rotation.
	Observability *ShootObservabilityRotation
	// ServiceAccountKey contains information about the service account key credential rotation.
	ServiceAccountKey *ShootServiceAccountKeyRotation
	// ETCDEncryptionKey contains information about the ETCD encryption key credential rotation.
	ETCDEncryptionKey *ShootETCDEncryptionKeyRotation
}

// ShootCARotation contains information about the certificate authority credential rotation.
type ShootCARotation struct {
	// Phase describes the phase of the certificate authority credential rotation.
	Phase ShootCredentialsRotationPhase
	// LastInitiationTime is the most recent time when the certificate authority credential rotation was initiated.
	LastInitiationTime *metav1.Time
	// LastCompletionTime is the most recent time when the certificate authority credential rotation was successfully
	// completed.
	LastCompletionTime *metav1.Time
}

// ShootKubeconfigRotation contains information about the kubeconfig credential rotation.
type ShootKubeconfigRotation struct {
	// LastInitiationTime is the most recent time when the kubeconfig credential rotation was initiated.
	LastInitiationTime *metav1.Time
	// LastCompletionTime is the most recent time when the kubeconfig credential rotation was successfully completed.
	LastCompletionTime *metav1.Time
}

// ShootSSHKeypairRotation contains information about the ssh-keypair credential rotation.
type ShootSSHKeypairRotation struct {
	// LastInitiationTime is the most recent time when the certificate authority credential rotation was initiated.
	LastInitiationTime *metav1.Time
	// LastCompletionTime is the most recent time when the ssh-keypair credential rotation was successfully completed.
	LastCompletionTime *metav1.Time
}

// ShootObservabilityRotation contains information about the observability credential rotation.
type ShootObservabilityRotation struct {
	// LastInitiationTime is the most recent time when the observability credential rotation was initiated.
	LastInitiationTime *metav1.Time
	// LastCompletionTime is the most recent time when the observability credential rotation was successfully completed.
	LastCompletionTime *metav1.Time
}

// ShootServiceAccountKeyRotation contains information about the service account key credential rotation.
type ShootServiceAccountKeyRotation struct {
	// Phase describes the phase of the service account key credential rotation.
	Phase ShootCredentialsRotationPhase
	// LastInitiationTime is the most recent time when the service account key credential rotation was initiated.
	LastInitiationTime *metav1.Time
	// LastCompletionTime is the most recent time when the service account key credential rotation was successfully
	// completed.
	LastCompletionTime *metav1.Time
}

// ShootETCDEncryptionKeyRotation contains information about the ETCD encryption key credential rotation.
type ShootETCDEncryptionKeyRotation struct {
	// Phase describes the phase of the ETCD encryption key credential rotation.
	Phase ShootCredentialsRotationPhase
	// LastInitiationTime is the most recent time when the ETCD encryption key credential rotation was initiated.
	LastInitiationTime *metav1.Time
	// LastCompletionTime is the most recent time when the ETCD encryption key credential rotation was successfully
	// completed.
	LastCompletionTime *metav1.Time
}

// ShootCredentialsRotationPhase is a string alias.
type ShootCredentialsRotationPhase string

const (
	// RotationPreparing is a constant for the credentials rotation phase describing that the procedure is being prepared.
	RotationPreparing ShootCredentialsRotationPhase = "Preparing"
	// RotationPrepared is a constant for the credentials rotation phase describing that the procedure was prepared.
	RotationPrepared ShootCredentialsRotationPhase = "Prepared"
	// RotationCompleting is a constant for the credentials rotation phase describing that the procedure is being
	// completed.
	RotationCompleting ShootCredentialsRotationPhase = "Completing"
	// RotationCompleted is a constant for the credentials rotation phase describing that the procedure was completed.
	RotationCompleted ShootCredentialsRotationPhase = "Completed"
)

// ShootAdvertisedAddress contains information for the shoot's Kube API server.
type ShootAdvertisedAddress struct {
	// Name of the advertised address. e.g. external
	Name string
	// The URL of the API Server. e.g. https://api.foo.bar or https://1.2.3.4
	URL string
}

//////////////////////////////////////////////////////////////////////////////////////////////////
// Addons relevant types                                                                        //
//////////////////////////////////////////////////////////////////////////////////////////////////

// Addons is a collection of configuration for specific addons which are managed by the Gardener.
type Addons struct {
	// KubernetesDashboard holds configuration settings for the kubernetes dashboard addon.
	KubernetesDashboard *KubernetesDashboard
	// NginxIngress holds configuration settings for the nginx-ingress addon.
	NginxIngress *NginxIngress
}

// Addon allows enabling or disabling a specific addon and is used to derive from.
type Addon struct {
	// Enabled indicates whether the addon is enabled or not.
	Enabled bool
}

// KubernetesDashboard describes configuration values for the kubernetes-dashboard addon.
type KubernetesDashboard struct {
	Addon
	// AuthenticationMode defines the authentication mode for the kubernetes-dashboard.
	AuthenticationMode *string
}

const (
	// KubernetesDashboardAuthModeBasic uses basic authentication mode for auth.
	KubernetesDashboardAuthModeBasic = "basic"
	// KubernetesDashboardAuthModeToken uses token-based mode for auth.
	KubernetesDashboardAuthModeToken = "token"
)

// NginxIngress describes configuration values for the nginx-ingress addon.
type NginxIngress struct {
	Addon
	// LoadBalancerSourceRanges is list of allowed IP sources for NginxIngress
	LoadBalancerSourceRanges []string
	// Config contains custom configuration for the nginx-ingress-controller configuration.
	// See https://github.com/kubernetes/ingress-nginx/blob/master/docs/user-guide/nginx-configuration/configmap.md#configuration-options
	Config map[string]string
	// ExternalTrafficPolicy controls the `.spec.externalTrafficPolicy` value of the load balancer `Service`
	// exposing the nginx-ingress. Defaults to `Cluster`.
	ExternalTrafficPolicy *corev1.ServiceExternalTrafficPolicyType
}

//////////////////////////////////////////////////////////////////////////////////////////////////
// DNS relevant types                                                                           //
//////////////////////////////////////////////////////////////////////////////////////////////////

// DNS holds information about the provider, the hosted zone id and the domain.
type DNS struct {
	// Domain is the external available domain of the Shoot cluster. This domain will be written into the
	// kubeconfig that is handed out to end-users. This field is immutable.
	Domain *string
	// Providers is a list of DNS providers that shall be enabled for this shoot cluster. Only relevant if
	// not a default domain is used.
	Providers []DNSProvider
}

// DNSProvider contains information about a DNS provider.
type DNSProvider struct {
	// Domains contains information about which domains shall be included/excluded for this provider.
	Domains *DNSIncludeExclude
	// Primary indicates that this DNSProvider is used for shoot related domains.
	Primary *bool
	// SecretName is a name of a secret containing credentials for the stated domain and the
	// provider. When not specified, the Gardener will use the cloud provider credentials referenced
	// by the Shoot and try to find respective credentials there. Specifying this field may override
	// this behavior, i.e. forcing the Gardener to only look into the given secret.
	SecretName *string
	// Type is the DNS provider type for the Shoot. Only relevant if not the default domain is used for
	// this shoot.
	Type *string
	// Zones contains information about which hosted zones shall be included/excluded for this provider.
	Zones *DNSIncludeExclude
}

// DNSIncludeExclude contains information about which domains shall be included/excluded.
type DNSIncludeExclude struct {
	// Include is a list of domains that shall be included.
	Include []string
	// Exclude is a list of domains that shall be excluded.
	Exclude []string
}

// DefaultDomain is the default value in the Shoot's '.spec.dns.domain' when '.spec.dns.provider' is 'unmanaged'
const DefaultDomain = "cluster.local"

//////////////////////////////////////////////////////////////////////////////////////////////////
// Extension relevant types                                                                     //
//////////////////////////////////////////////////////////////////////////////////////////////////

// Extension contains type and provider information for Shoot extensions.
type Extension struct {
	// Type is the type of the extension resource.
	Type string
	// ProviderConfig is the configuration passed to extension resource.
	ProviderConfig *runtime.RawExtension
	// Disabled allows to disable extensions that were marked as 'globally enabled' by Gardener administrators.
	Disabled *bool
}

//////////////////////////////////////////////////////////////////////////////////////////////////
// NamedResourceReference relevant types                                                        //
//////////////////////////////////////////////////////////////////////////////////////////////////

// NamedResourceReference is a named reference to a resource.
type NamedResourceReference struct {
	// Name of the resource reference.
	Name string
	// ResourceRef is a reference to a resource.
	ResourceRef autoscalingv1.CrossVersionObjectReference
}

//////////////////////////////////////////////////////////////////////////////////////////////////
// Hibernation relevant types                                                                   //
//////////////////////////////////////////////////////////////////////////////////////////////////

// Hibernation contains information whether the Shoot is suspended or not.
type Hibernation struct {
	// Enabled specifies whether the Shoot needs to be hibernated or not. If it is true, the Shoot's desired state is to be hibernated.
	// If it is false or nil, the Shoot's desired state is to be awakened.
	Enabled *bool
	// Schedules determine the hibernation schedules.
	Schedules []HibernationSchedule
}

// HibernationSchedule determines the hibernation schedule of a Shoot.
// A Shoot will be regularly hibernated at each start time and will be woken up at each end time.
// Start or End can be omitted, though at least one of each has to be specified.
type HibernationSchedule struct {
	// Start is a Cron spec at which time a Shoot will be hibernated.
	Start *string
	// End is a Cron spec at which time a Shoot will be woken up.
	End *string
	// Location is the time location in which both start and and shall be evaluated.
	Location *string
}

//////////////////////////////////////////////////////////////////////////////////////////////////
// Kubernetes relevant types                                                                    //
//////////////////////////////////////////////////////////////////////////////////////////////////

// Kubernetes contains the version and configuration variables for the Shoot control plane.
type Kubernetes struct {
	// AllowPrivilegedContainers indicates whether privileged containers are allowed in the Shoot (default: true).
	AllowPrivilegedContainers *bool
	// ClusterAutoscaler contains the configuration flags for the Kubernetes cluster autoscaler.
	ClusterAutoscaler *ClusterAutoscaler
	// KubeAPIServer contains configuration settings for the kube-apiserver.
	KubeAPIServer *KubeAPIServerConfig
	// KubeControllerManager contains configuration settings for the kube-controller-manager.
	KubeControllerManager *KubeControllerManagerConfig
	// KubeScheduler contains configuration settings for the kube-scheduler.
	KubeScheduler *KubeSchedulerConfig
	// KubeProxy contains configuration settings for the kube-proxy.
	KubeProxy *KubeProxyConfig
	// Kubelet contains configuration settings for the kubelet.
	Kubelet *KubeletConfig
	// Version is the semantic Kubernetes version to use for the Shoot cluster.
	Version string
	// VerticalPodAutoscaler contains the configuration flags for the Kubernetes vertical pod autoscaler.
	VerticalPodAutoscaler *VerticalPodAutoscaler
	// EnableStaticTokenKubeconfig indicates whether static token kubeconfig secret will be created for shoot (default: true).
	EnableStaticTokenKubeconfig *bool
}

// ClusterAutoscaler contains the configuration flags for the Kubernetes cluster autoscaler.
type ClusterAutoscaler struct {
	// ScaleDownDelayAfterAdd defines how long after scale up that scale down evaluation resumes (default: 1 hour).
	ScaleDownDelayAfterAdd *metav1.Duration
	// ScaleDownDelayAfterDelete how long after node deletion that scale down evaluation resumes, defaults to scanInterval (default: 0 secs).
	ScaleDownDelayAfterDelete *metav1.Duration
	// ScaleDownDelayAfterFailure how long after scale down failure that scale down evaluation resumes (default: 3 mins).
	ScaleDownDelayAfterFailure *metav1.Duration
	// ScaleDownUnneededTime defines how long a node should be unneeded before it is eligible for scale down (default: 30 mins).
	ScaleDownUnneededTime *metav1.Duration
	// ScaleDownUtilizationThreshold defines the threshold in fraction (0.0 - 1.0) under which a node is being removed (default: 0.5).
	ScaleDownUtilizationThreshold *float64
	// ScanInterval how often cluster is reevaluated for scale up or down (default: 10 secs).
	ScanInterval *metav1.Duration
	// Expander defines the algorithm to use during scale up (default: least-waste).
	// See: https://github.com/gardener/autoscaler/blob/machine-controller-manager-provider/cluster-autoscaler/FAQ.md#what-are-expanders.
	Expander *ExpanderMode
	// MaxNodeProvisionTime defines how long CA waits for node to be provisioned (default: 20 mins).
	MaxNodeProvisionTime *metav1.Duration
	// MaxGracefulTerminationSeconds is the number of seconds CA waits for pod termination when trying to scale down a node (default: 600).
	MaxGracefulTerminationSeconds *int32
	// IgnoreTaints specifies a list of taint keys to ignore in node templates when considering to scale a node group.
	IgnoreTaints []string
}

// ExpanderMode is type used for Expander values
type ExpanderMode string

const (
	// ClusterAutoscalerExpanderLeastWaste selects the node group that will have the least idle CPU (if tied, unused memory) after scale-up.
	// This is useful when you have different classes of nodes, for example, high CPU or high memory nodes, and
	// only want to expand those when there are pending pods that need a lot of those resources.
	// This is the default value.
	ClusterAutoscalerExpanderLeastWaste ExpanderMode = "least-waste"
	// ClusterAutoscalerExpanderMostPods selects the node group that would be able to schedule the most pods when scaling up.
	// This is useful when you are using nodeSelector to make sure certain pods land on certain nodes.
	// Note that this won't cause the autoscaler to select bigger nodes vs. smaller, as it can add multiple smaller nodes at once.
	ClusterAutoscalerExpanderMostPods ExpanderMode = "most-pods"
	// ClusterAutoscalerExpanderPriority selects the node group that has the highest priority assigned by the user. For configurations,
	// See: https://github.com/gardener/autoscaler/blob/machine-controller-manager-provider/cluster-autoscaler/expander/priority/readme.md
	ClusterAutoscalerExpanderPriority ExpanderMode = "priority"
	// ClusterAutoscalerExpanderRandom should be used when you don't have a particular need
	// for the node groups to scale differently.
	ClusterAutoscalerExpanderRandom ExpanderMode = "random"
)

// VerticalPodAutoscaler contains the configuration flags for the Kubernetes vertical pod autoscaler.
type VerticalPodAutoscaler struct {
	// Enabled specifies whether the Kubernetes VPA shall be enabled for the shoot cluster.
	Enabled bool
	// EvictAfterOOMThreshold defines the threshold that will lead to pod eviction in case it OOMed in less than the given
	// threshold since its start and if it has only one container (default: 10m0s).
	EvictAfterOOMThreshold *metav1.Duration
	// EvictionRateBurst defines the burst of pods that can be evicted (default: 1)
	EvictionRateBurst *int32
	// EvictionRateLimit defines the number of pods that can be evicted per second. A rate limit set to 0 or -1 will
	// disable the rate limiter (default: -1).
	EvictionRateLimit *float64
	// EvictionTolerance defines the fraction of replica count that can be evicted for update in case more than one
	// pod can be evicted (default: 0.5).
	EvictionTolerance *float64
	// RecommendationMarginFraction is the fraction of usage added as the safety margin to the recommended request
	// (default: 0.15).
	RecommendationMarginFraction *float64
	// UpdaterInterval is the interval how often the updater should run (default: 1m0s).
	UpdaterInterval *metav1.Duration
	// RecommenderInterval is the interval how often metrics should be fetched (default: 1m0s).
	RecommenderInterval *metav1.Duration
}

// KubernetesConfig contains common configuration fields for the control plane components.
type KubernetesConfig struct {
	// FeatureGates contains information about enabled feature gates.
	FeatureGates map[string]bool
}

// KubeAPIServerConfig contains configuration settings for the kube-apiserver.
type KubeAPIServerConfig struct {
	KubernetesConfig
	// AdmissionPlugins contains the list of user-defined admission plugins (additional to those managed by Gardener), and, if desired, the corresponding
	// configuration.
	AdmissionPlugins []AdmissionPlugin
	// APIAudiences are the identifiers of the API. The service account token authenticator will
	// validate that tokens used against the API are bound to at least one of these audiences.
	// Defaults to ["kubernetes"].
	APIAudiences []string
	// AuditConfig contains configuration settings for the audit of the kube-apiserver.
	AuditConfig *AuditConfig
	// EnableBasicAuthentication defines whether basic authentication should be enabled for this cluster or not.
	EnableBasicAuthentication *bool
	// OIDCConfig contains configuration settings for the OIDC provider.
	OIDCConfig *OIDCConfig
	// RuntimeConfig contains information about enabled or disabled APIs.
	RuntimeConfig map[string]bool
	// ServiceAccountConfig contains configuration settings for the service account handling
	// of the kube-apiserver.
	ServiceAccountConfig *ServiceAccountConfig
	// WatchCacheSizes contains configuration of the API server's watch cache sizes.
	// Configuring these flags might be useful for large-scale Shoot clusters with a lot of parallel update requests
	// and a lot of watching controllers (e.g. large shooted Seed clusters). When the API server's watch cache's
	// capacity is too small to cope with the amount of update requests and watchers for a particular resource, it
	// might happen that controller watches are permanently stopped with `too old resource version` errors.
	// Starting from kubernetes v1.19, the API server's watch cache size is adapted dynamically and setting the watch
	// cache size flags will have no effect, except when setting it to 0 (which disables the watch cache).
	WatchCacheSizes *WatchCacheSizes
	// Requests contains configuration for request-specific settings for the kube-apiserver.
	Requests *KubeAPIServerRequests
	// EnableAnonymousAuthentication defines whether anonymous requests to the secure port
	// of the API server should be allowed (flag `--anonymous-auth`).
	// See: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/
	EnableAnonymousAuthentication *bool
	// EventTTL controls the amount of time to retain events.
	EventTTL *metav1.Duration
}

// KubeAPIServerRequests contains configuration for request-specific settings for the kube-apiserver.
type KubeAPIServerRequests struct {
	// MaxNonMutatingInflight is the maximum number of non-mutating requests in flight at a given time. When the server
	// exceeds this, it rejects requests.
	MaxNonMutatingInflight *int32
	// MaxMutatingInflight is the maximum number of mutating requests in flight at a given time. When the server
	// exceeds this, it rejects requests.
	MaxMutatingInflight *int32
}

// ServiceAccountConfig is the kube-apiserver configuration for service accounts.
type ServiceAccountConfig struct {
	// Issuer is the identifier of the service account token issuer. The issuer will assert this
	// identifier in "iss" claim of issued tokens. This value is used to generate new service account tokens.
	// This value is a string or URI. Defaults to URI of the API server.
	Issuer *string
	// SigningKeySecret is a reference to a secret that contains an optional private key of the
	// service account token issuer. The issuer will sign issued ID tokens with this private key.
	// Only useful if service account tokens are also issued by another external system.
	// Deprecated: This field is deprecated and will be removed in a future version of Gardener. Do not use it.
	SigningKeySecret *corev1.LocalObjectReference
	// ExtendTokenExpiration turns on projected service account expiration extension during token generation, which
	// helps safe transition from legacy token to bound service account token feature. If this flag is enabled,
	// admission injected tokens would be extended up to 1 year to prevent unexpected failure during transition,
	// ignoring value of service-account-max-token-expiration.
	ExtendTokenExpiration *bool
	// MaxTokenExpiration is the maximum validity duration of a token created by the service account token issuer. If an
	// otherwise valid TokenRequest with a validity duration larger than this value is requested, a token will be issued
	// with a validity duration of this value.
	// This field must be within [30d,90d].
	MaxTokenExpiration *metav1.Duration
	// AcceptedIssuers is an additional set of issuers that are used to determine which service account tokens are accepted.
	// These values are not used to generate new service account tokens. Only useful when service account tokens are also
	// issued by another external system or a change of the current issuer that is used for generating tokens is being performed.
	// This field is only available for Kubernetes v1.22 or later.
	AcceptedIssuers []string
}

// AuditConfig contains settings for audit of the api server
type AuditConfig struct {
	// AuditPolicy contains configuration settings for audit policy of the kube-apiserver.
	AuditPolicy *AuditPolicy
}

// AuditPolicy contains audit policy for kube-apiserver
type AuditPolicy struct {
	// ConfigMapRef is a reference to a ConfigMap object in the same namespace,
	// which contains the audit policy for the kube-apiserver.
	ConfigMapRef *corev1.ObjectReference
}

// OIDCConfig contains configuration settings for the OIDC provider.
// Note: Descriptions were taken from the Kubernetes documentation.
type OIDCConfig struct {
	// If set, the OpenID server's certificate will be verified by one of the authorities in the oidc-ca-file, otherwise the host's root CA set will be used.
	CABundle *string
	// ClientAuthentication can optionally contain client configuration used for kubeconfig generation.
	ClientAuthentication *OpenIDConnectClientAuthentication
	// The client ID for the OpenID Connect client, must be set if oidc-issuer-url is set.
	ClientID *string
	// If provided, the name of a custom OpenID Connect claim for specifying user groups. The claim value is expected to be a string or array of strings. This flag is experimental, please see the authentication documentation for further details.
	GroupsClaim *string
	// If provided, all groups will be prefixed with this value to prevent conflicts with other authentication strategies.
	GroupsPrefix *string
	// The URL of the OpenID issuer, only HTTPS scheme will be accepted. If set, it will be used to verify the OIDC JSON Web Token (JWT).
	IssuerURL *string
	// key=value pairs that describes a required claim in the ID Token. If set, the claim is verified to be present in the ID Token with a matching value.
	RequiredClaims map[string]string
	// List of allowed JOSE asymmetric signing algorithms. JWTs with a 'alg' header value not in this list will be rejected. Values are defined by RFC 7518 https://tools.ietf.org/html/rfc7518#section-3.1
	SigningAlgs []string
	// The OpenID claim to use as the user name. Note that claims other than the default ('sub') is not guaranteed to be unique and immutable. This flag is experimental, please see the authentication documentation for further details. (default "sub")
	UsernameClaim *string
	// If provided, all usernames will be prefixed with this value. If not provided, username claims other than 'email' are prefixed by the issuer URL to avoid clashes. To skip any prefixing, provide the value '-'.
	UsernamePrefix *string
}

// OpenIDConnectClientAuthentication contains configuration for OIDC clients.
type OpenIDConnectClientAuthentication struct {
	// Extra configuration added to kubeconfig's auth-provider.
	// Must not be any of idp-issuer-url, client-id, client-secret, idp-certificate-authority, idp-certificate-authority-data, id-token or refresh-token
	ExtraConfig map[string]string
	// The client Secret for the OpenID Connect client.
	Secret *string
}

// AdmissionPlugin contains information about a specific admission plugin and its corresponding configuration.
type AdmissionPlugin struct {
	// Name is the name of the plugin.
	Name string
	// Config is the configuration of the plugin.
	Config *runtime.RawExtension
}

// WatchCacheSizes contains configuration of the API server's watch cache sizes.
type WatchCacheSizes struct {
	// Default configures the default watch cache size of the kube-apiserver
	// (flag `--default-watch-cache-size`, defaults to 100).
	// See: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/
	Default *int32
	// Resources configures the watch cache size of the kube-apiserver per resource
	// (flag `--watch-cache-sizes`).
	// See: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/
	Resources []ResourceWatchCacheSize
}

// ResourceWatchCacheSize contains configuration of the API server's watch cache size for one specific resource.
type ResourceWatchCacheSize struct {
	// APIGroup is the API group of the resource for which the watch cache size should be configured.
	// An unset value is used to specify the legacy core API (e.g. for `secrets`).
	APIGroup *string
	// Resource is the name of the resource for which the watch cache size should be configured
	// (in lowercase plural form, e.g. `secrets`).
	Resource string
	// CacheSize specifies the watch cache size that should be configured for the specified resource.
	CacheSize int32
}

// KubeControllerManagerConfig contains configuration settings for the kube-controller-manager.
type KubeControllerManagerConfig struct {
	KubernetesConfig
	// HorizontalPodAutoscalerConfig contains horizontal pod autoscaler configuration settings for the kube-controller-manager.
	HorizontalPodAutoscalerConfig *HorizontalPodAutoscalerConfig
	// NodeCIDRMaskSize defines the mask size for node cidr in cluster (default is 24). This field is immutable.
	NodeCIDRMaskSize *int32
	// PodEvictionTimeout defines the grace period for deleting pods on failed nodes.
	PodEvictionTimeout *metav1.Duration
	// NodeMonitorGracePeriod defines the grace period before an unresponsive node is marked unhealthy.
	NodeMonitorGracePeriod *metav1.Duration
}

// HorizontalPodAutoscalerConfig contains horizontal pod autoscaler configuration settings for the kube-controller-manager.
// Note: Descriptions were taken from the Kubernetes documentation.
type HorizontalPodAutoscalerConfig struct {
	// The period after which a ready pod transition is considered to be the first.
	CPUInitializationPeriod *metav1.Duration
	// The configurable window at which the controller will choose the highest recommendation for autoscaling.
	DownscaleStabilization *metav1.Duration
	// The configurable period at which the horizontal pod autoscaler considers a Pod “not yet ready” given that it’s unready and it has  transitioned to unready during that time.
	InitialReadinessDelay *metav1.Duration
	// The period for syncing the number of pods in horizontal pod autoscaler.
	SyncPeriod *metav1.Duration
	// The minimum change (from 1.0) in the desired-to-actual metrics ratio for the horizontal pod autoscaler to consider scaling.
	Tolerance *float64
}

// KubeSchedulerConfig contains configuration settings for the kube-scheduler.
type KubeSchedulerConfig struct {
	KubernetesConfig
	// KubeMaxPDVols allows to configure the `KUBE_MAX_PD_VOLS` environment variable for the kube-scheduler.
	// Please find more information here: https://kubernetes.io/docs/concepts/storage/storage-limits/#custom-limits
	// Note that using this field is considered alpha-/experimental-level and is on your own risk. You should be aware
	// of all the side-effects and consequences when changing it.
	KubeMaxPDVols *string
	// Profile configures the scheduling profile for the cluster.
	// If not specified, the used profile is "balanced" (provides the default kube-scheduler behavior).
	Profile *SchedulingProfile
}

// SchedulingProfile is a string alias used for scheduling profile values.
type SchedulingProfile string

const (
	// SchedulingProfileBalanced is a scheduling profile that attempts to spread Pods evenly across Nodes
	// to obtain a more balanced resource usage. This profile provides the default kube-scheduler behavior.
	SchedulingProfileBalanced SchedulingProfile = "balanced"
	// SchedulingProfileBinPacking is a scheduling profile that scores Nodes based on the allocation of resources.
	// It prioritizes Nodes with most allocated resources. This leads the Node count in the cluster to be minimized and
	// the Node resource utilization to be increased.
	SchedulingProfileBinPacking SchedulingProfile = "bin-packing"
)

// KubeProxyConfig contains configuration settings for the kube-proxy.
type KubeProxyConfig struct {
	KubernetesConfig
	// Mode specifies which proxy mode to use.
	// defaults to IPTables.
	Mode *ProxyMode
	// Enabled indicates whether kube-proxy should be deployed or not.
	// Depending on the networking extensions switching kube-proxy off might be rejected. Consulting the respective documentation of the used networking extension is recommended before using this field.
	// defaults to true if not specified.
	Enabled *bool
}

// ProxyMode available in Linux platform: 'userspace' (older, going to be EOL), 'iptables'
// (newer, faster), 'ipvs' (newest, better in performance and scalability).
// As of now only 'iptables' and 'ipvs' is supported by Gardener.
// In Linux platform, if the iptables proxy is selected, regardless of how, but the system's kernel or iptables versions are
// insufficient, this always falls back to the userspace proxy. IPVS mode will be enabled when proxy mode is set to 'ipvs',
// and the fall back path is firstly iptables and then userspace.
type ProxyMode string

const (
	// ProxyModeIPTables uses iptables as proxy implementation.
	ProxyModeIPTables ProxyMode = "IPTables"
	// ProxyModeIPVS uses ipvs as proxy implementation.
	ProxyModeIPVS ProxyMode = "IPVS"
)

// KubeletConfig contains configuration settings for the kubelet.
type KubeletConfig struct {
	KubernetesConfig
	// CPUCFSQuota allows you to disable/enable CPU throttling for Pods.
	CPUCFSQuota *bool
	// CPUManagerPolicy allows to set alternative CPU management policies (default: none).
	CPUManagerPolicy *string
	// EvictionHard describes a set of eviction thresholds (e.g. memory.available<1Gi) that if met would trigger a Pod eviction.
	// Default:
	//   memory.available:   "100Mi/1Gi/5%"
	//   nodefs.available:   "5%"
	//   nodefs.inodesFree:  "5%"
	//   imagefs.available:  "5%"
	//   imagefs.inodesFree: "5%"
	EvictionHard *KubeletConfigEviction
	// EvictionMaxPodGracePeriod describes the maximum allowed grace period (in seconds) to use when terminating pods in response to a soft eviction threshold being met.
	// Default: 90
	EvictionMaxPodGracePeriod *int32
	// EvictionMinimumReclaim configures the amount of resources below the configured eviction threshold that the kubelet attempts to reclaim whenever the kubelet observes resource pressure.
	// Default: 0 for each resource
	EvictionMinimumReclaim *KubeletConfigEvictionMinimumReclaim
	// EvictionPressureTransitionPeriod is the duration for which the kubelet has to wait before transitioning out of an eviction pressure condition.
	// Default: 4m0s
	EvictionPressureTransitionPeriod *metav1.Duration
	// EvictionSoft describes a set of eviction thresholds (e.g. memory.available<1.5Gi) that if met over a corresponding grace period would trigger a Pod eviction.
	// Default:
	//   memory.available:   "200Mi/1.5Gi/10%"
	//   nodefs.available:   "10%"
	//   nodefs.inodesFree:  "10%"
	//   imagefs.available:  "10%"
	//   imagefs.inodesFree: "10%"
	EvictionSoft *KubeletConfigEviction
	// EvictionSoftGracePeriod describes a set of eviction grace periods (e.g. memory.available=1m30s) that correspond to how long a soft eviction threshold must hold before triggering a Pod eviction.
	// Default:
	//   memory.available:   1m30s
	//   nodefs.available:   1m30s
	//   nodefs.inodesFree:  1m30s
	//   imagefs.available:  1m30s
	//   imagefs.inodesFree: 1m30s
	EvictionSoftGracePeriod *KubeletConfigEvictionSoftGracePeriod
	// MaxPods is the maximum number of Pods that are allowed by the Kubelet.
	// Default: 110
	MaxPods *int32
	// PodPIDsLimit is the maximum number of process IDs per pod allowed by the kubelet.
	PodPIDsLimit *int64
	// ImagePullProgressDeadline describes the time limit under which if no pulling progress is made, the image pulling will be cancelled.
	// Default: 1m
	ImagePullProgressDeadline *metav1.Duration
	// FailSwapOn makes the Kubelet fail to start if swap is enabled on the node. (default true).
	FailSwapOn *bool
	// KubeReserved is the configuration for resources reserved for kubernetes node components (mainly kubelet and container runtime).
	// When updating these values, be aware that cgroup resizes may not succeed on active worker nodes. Look for the NodeAllocatableEnforced event to determine if the configuration was applied.
	// Default: cpu=80m,memory=1Gi,pid=20k
	KubeReserved *KubeletConfigReserved
	// SystemReserved is the configuration for resources reserved for system processes not managed by kubernetes (e.g. journald).
	// When updating these values, be aware that cgroup resizes may not succeed on active worker nodes. Look for the NodeAllocatableEnforced event to determine if the configuration was applied.
	SystemReserved *KubeletConfigReserved
	// ImageGCHighThresholdPercent describes the percent of the disk usage which triggers image garbage collection.
	ImageGCHighThresholdPercent *int32
	// ImageGCLowThresholdPercent describes the percent of the disk to which garbage collection attempts to free.
	ImageGCLowThresholdPercent *int32
	// SerializeImagePulls describes whether the images are pulled one at a time.
	SerializeImagePulls *bool
}

// KubeletConfigEviction contains kubelet eviction thresholds supporting either a resource.Quantity or a percentage based value.
type KubeletConfigEviction struct {
	// MemoryAvailable is the threshold for the free memory on the host server.
	MemoryAvailable *string
	// ImageFSAvailable is the threshold for the free disk space in the imagefs filesystem (docker images and container writable layers).
	ImageFSAvailable *string
	// ImageFSInodesFree is the threshold for the available inodes in the imagefs filesystem.
	ImageFSInodesFree *string
	// NodeFSAvailable is the threshold for the free disk space in the nodefs filesystem (docker volumes, logs, etc).
	NodeFSAvailable *string
	// NodeFSInodesFree is the threshold for the available inodes in the nodefs filesystem.
	NodeFSInodesFree *string
}

// KubeletConfigEvictionMinimumReclaim contains configuration for the kubelet eviction minimum reclaim.
type KubeletConfigEvictionMinimumReclaim struct {
	// MemoryAvailable is the threshold for the memory reclaim on the host server.
	MemoryAvailable *resource.Quantity
	// ImageFSAvailable is the threshold for the disk space reclaim in the imagefs filesystem (docker images and container writable layers).
	ImageFSAvailable *resource.Quantity
	// ImageFSInodesFree is the threshold for the inodes reclaim in the imagefs filesystem.
	ImageFSInodesFree *resource.Quantity
	// NodeFSAvailable is the threshold for the disk space reclaim in the nodefs filesystem (docker volumes, logs, etc).
	NodeFSAvailable *resource.Quantity
	// NodeFSInodesFree is the threshold for the inodes reclaim in the nodefs filesystem.
	NodeFSInodesFree *resource.Quantity
}

// KubeletConfigEvictionSoftGracePeriod contains grace periods for kubelet eviction thresholds.
type KubeletConfigEvictionSoftGracePeriod struct {
	// MemoryAvailable is the grace period for the MemoryAvailable eviction threshold.
	MemoryAvailable *metav1.Duration
	// ImageFSAvailable is the grace period for the ImageFSAvailable eviction threshold.
	ImageFSAvailable *metav1.Duration
	// ImageFSInodesFree is the grace period for the ImageFSInodesFree eviction threshold.
	ImageFSInodesFree *metav1.Duration
	// NodeFSAvailable is the grace period for the NodeFSAvailable eviction threshold.
	NodeFSAvailable *metav1.Duration
	// NodeFSInodesFree is the grace period for the NodeFSInodesFree eviction threshold.
	NodeFSInodesFree *metav1.Duration
}

// KubeletConfigReserved contains reserved resources for daemons
type KubeletConfigReserved struct {
	// CPU is the reserved cpu.
	CPU *resource.Quantity
	// Memory is the reserved memory.
	Memory *resource.Quantity
	// EphemeralStorage is the reserved ephemeral-storage.
	EphemeralStorage *resource.Quantity
	// PID is the reserved process-ids.
	PID *resource.Quantity
}

//////////////////////////////////////////////////////////////////////////////////////////////////
// Networking relevant types                                                                    //
//////////////////////////////////////////////////////////////////////////////////////////////////

// Networking defines networking parameters for the shoot cluster.
type Networking struct {
	// Type identifies the type of the networking plugin. This field is immutable.
	Type string
	// ProviderConfig is the configuration passed to network resource.
	ProviderConfig *runtime.RawExtension
	// Pods is the CIDR of the pod network. This field is immutable.
	Pods *string
	// Nodes is the CIDR of the entire node network. This field is immutable.
	Nodes *string
	// Services is the CIDR of the service network. This field is immutable.
	Services *string
}

const (
	// DefaultPodNetworkCIDR is a constant for the default pod network CIDR of a Shoot cluster.
	DefaultPodNetworkCIDR = "100.96.0.0/11"
	// DefaultServiceNetworkCIDR is a constant for the default service network CIDR of a Shoot cluster.
	DefaultServiceNetworkCIDR = "100.64.0.0/13"
)

//////////////////////////////////////////////////////////////////////////////////////////////////
// Maintenance relevant types                                                                   //
//////////////////////////////////////////////////////////////////////////////////////////////////

const (
	// MaintenanceTimeWindowDurationMinimum is the minimum duration for a maintenance time window.
	MaintenanceTimeWindowDurationMinimum = 30 * time.Minute
	// MaintenanceTimeWindowDurationMaximum is the maximum duration for a maintenance time window.
	MaintenanceTimeWindowDurationMaximum = 6 * time.Hour
)

// Maintenance contains information about the time window for maintenance operations and which
// operations should be performed.
type Maintenance struct {
	// AutoUpdate contains information about which constraints should be automatically updated.
	AutoUpdate *MaintenanceAutoUpdate
	// TimeWindow contains information about the time window for maintenance operations.
	TimeWindow *MaintenanceTimeWindow
	// ConfineSpecUpdateRollout prevents that changes/updates to the shoot specification will be rolled out immediately.
	// Instead, they are rolled out during the shoot's maintenance time window. There is one exception that will trigger
	// an immediate roll out which is changes to the Spec.Hibernation.Enabled field.
	ConfineSpecUpdateRollout *bool
}

// MaintenanceAutoUpdate contains information about which constraints should be automatically updated.
type MaintenanceAutoUpdate struct {
	// KubernetesVersion indicates whether the patch Kubernetes version may be automatically updated (default: true).
	KubernetesVersion bool
	// MachineImageVersion indicates whether the machine image version may be automatically updated (default: true).
	MachineImageVersion bool
}

// MaintenanceTimeWindow contains information about the time window for maintenance operations.
type MaintenanceTimeWindow struct {
	// Begin is the beginning of the time window in the format HHMMSS+ZONE, e.g. "220000+0100".
	// If not present, a random value will be computed.
	Begin string
	// End is the end of the time window in the format HHMMSS+ZONE, e.g. "220000+0100".
	// If not present, the value will be computed based on the "Begin" value.
	End string
}

//////////////////////////////////////////////////////////////////////////////////////////////////
// Monitoring relevant types                                                                    //
//////////////////////////////////////////////////////////////////////////////////////////////////

// Monitoring contains information about the monitoring configuration for the shoot.
type Monitoring struct {
	// Alerting contains information about the alerting configuration for the shoot cluster.
	Alerting *Alerting
}

// Alerting contains information about how alerting will be done (i.e. who will receive alerts and how).
type Alerting struct {
	// MonitoringEmailReceivers is a list of recipients for alerts
	EmailReceivers []string
}

//////////////////////////////////////////////////////////////////////////////////////////////////
// Provider relevant types                                                                      //
//////////////////////////////////////////////////////////////////////////////////////////////////

// Provider contains provider-specific information that are handed-over to the provider-specific
// extension controller.
type Provider struct {
	// Type is the type of the provider. This field is immutable.
	Type string
	// ControlPlaneConfig contains the provider-specific control plane config blob. Please look up the concrete
	// definition in the documentation of your provider extension.
	ControlPlaneConfig *runtime.RawExtension
	// InfrastructureConfig contains the provider-specific infrastructure config blob. Please look up the concrete
	// definition in the documentation of your provider extension.
	InfrastructureConfig *runtime.RawExtension
	// Workers is a list of worker groups.
	Workers []Worker
}

// Worker is the base definition of a worker group.
type Worker struct {
	// Annotations is a map of key/value pairs for annotations for all the `Node` objects in this worker pool.
	Annotations map[string]string
	// CABundle is a certificate bundle which will be installed onto every machine of this worker pool.
	CABundle *string
	// CRI contains configurations of CRI support of every machine in the worker pool.
	// Defaults to a CRI with name `containerd` when the Kubernetes version of the `Shoot` is >= 1.22.
	CRI *CRI
	// Kubernetes contains configuration for Kubernetes components related to this worker pool.
	Kubernetes *WorkerKubernetes
	// Labels is a map of key/value pairs for labels for all the `Node` objects in this worker pool.
	Labels map[string]string
	// Name is the name of the worker group.
	Name string
	// Machine contains information about the machine type and image.
	Machine Machine
	// Maximum is the maximum number of VMs to create.
	Maximum int32
	// Minimum is the minimum number of VMs to create.
	Minimum int32
	// MaxSurge is maximum number of VMs that are created during an update.
	MaxSurge *intstr.IntOrString
	// MaxUnavailable is the maximum number of VMs that can be unavailable during an update.
	MaxUnavailable *intstr.IntOrString
	// ProviderConfig is the provider-specific configuration for this worker pool.
	ProviderConfig *runtime.RawExtension
	// SystemComponents contains configuration for system components related to this worker pool
	SystemComponents *WorkerSystemComponents
	// Taints is a list of taints for all the `Node` objects in this worker pool.
	Taints []corev1.Taint
	// Volume contains information about the volume type and size.
	Volume *Volume
	// DataVolumes contains a list of additional worker volumes.
	DataVolumes []DataVolume
	// KubeletDataVolumeName contains the name of a dataVolume that should be used for storing kubelet state.
	KubeletDataVolumeName *string
	// Zones is a list of availability zones that are used to evenly distribute this worker pool. Optional
	// as not every provider may support availability zones.
	Zones []string
	// MachineControllerManagerSettings contains configurations for different worker-pools. Eg. MachineDrainTimeout, MachineHealthTimeout.
	MachineControllerManagerSettings *MachineControllerManagerSettings
}

// MachineControllerManagerSettings contains configurations for different worker-pools. Eg. MachineDrainTimeout, MachineHealthTimeout.
type MachineControllerManagerSettings struct {
	// MachineDrainTimeout is the period after which machine is forcefully deleted.
	MachineDrainTimeout *metav1.Duration
	// MachineHealthTimeout is the period after which machine is declared failed.
	MachineHealthTimeout *metav1.Duration
	// MachineCreationTimeout is the period after which creation of the machine is declared failed.
	MachineCreationTimeout *metav1.Duration
	// MaxEvictRetries are the number of eviction retries on a pod after which drain is declared failed, and forceful deletion is triggered.
	MaxEvictRetries *int32
	// NodeConditions are the set of conditions if set to true for the period of MachineHealthTimeout, machine will be declared failed.
	NodeConditions []string
}

// WorkerSystemComponents contains configuration for system components related to this worker pool
type WorkerSystemComponents struct {
	// Allow determines whether the pool should be allowed to host system components or not (defaults to true)
	Allow bool
}

// WorkerKubernetes contains configuration for Kubernetes components related to this worker pool.
type WorkerKubernetes struct {
	// Kubelet contains configuration settings for all kubelets of this worker pool.
	// If set, all `spec.kubernetes.kubelet` settings will be overwritten for this worker pool (no merge of settings).
	Kubelet *KubeletConfig
	// Version is the semantic Kubernetes version to use for the Kubelet in this Worker Group.
	// If not specified the kubelet version is derived from the global shoot cluster kubernetes version.
	// version must be equal or lower than the version of the shoot kubernetes version.
	// Only one minor version difference to other worker groups and global kubernetes version is allowed.
	Version *string
}

// Machine contains information about the machine type and image.
type Machine struct {
	// Type is the machine type of the worker group.
	Type string
	// Image holds information about the machine image to use for all nodes of this pool. It will default to the
	// latest version of the first image stated in the referenced CloudProfile if no value has been provided.
	Image *ShootMachineImage
	// Architecture is the CPU architecture of the machines in this worker pool.
	Architecture *string
}

// ShootMachineImage defines the name and the version of the shoot's machine image in any environment. Has to be
// defined in the respective CloudProfile.
type ShootMachineImage struct {
	// Name is the name of the image.
	Name string
	// ProviderConfig is the shoot's individual configuration passed to an extension resource.
	ProviderConfig *runtime.RawExtension
	// Version is the version of the shoot's image.
	// If version is not provided, it will be defaulted to the latest version from the CloudProfile.
	Version string
}

// Volume contains information about the volume type and size.
type Volume struct {
	// Name of the volume to make it referencable.
	Name *string
	// Type is the type of the volume.
	Type *string
	// VolumeSize is the size of the volume.
	VolumeSize string
	// Encrypted determines if the volume should be encrypted.
	Encrypted *bool
}

// DataVolume contains information about a data volume.
type DataVolume struct {
	// Name of the volume to make it referencable.
	Name string
	// Type is the type of the volume.
	Type *string
	// VolumeSize is the size of the volume.
	VolumeSize string
	// Encrypted determines if the volume should be encrypted.
	Encrypted *bool
}

// CRI contains information about the Container Runtimes.
type CRI struct {
	// The name of the CRI library
	Name CRIName
	// ContainerRuntimes is the list of the required container runtimes supported for a worker pool.
	ContainerRuntimes []ContainerRuntime
}

// CRIName is a type alias for the CRI name string.
type CRIName string

const (
	// CRINameContainerD is a constant for ContainerD CRI name.
	CRINameContainerD CRIName = "containerd"
	// CRINameDocker is a constant for Docker CRI name.
	CRINameDocker CRIName = "docker"
)

// ContainerRuntime contains information about worker's available container runtime
type ContainerRuntime struct {
	// Type is the type of the Container Runtime.
	Type string
	// ProviderConfig is the configuration passed to the ContainerRuntime resource.
	ProviderConfig *runtime.RawExtension
}

var (
	// DefaultWorkerMaxSurge is the default value for Worker MaxSurge.
	DefaultWorkerMaxSurge = intstr.FromInt(1)
	// DefaultWorkerMaxUnavailable is the default value for Worker MaxUnavailable.
	DefaultWorkerMaxUnavailable = intstr.FromInt(0)
)

//////////////////////////////////////////////////////////////////////////////////////////////////
// System components relevant types                                                             //
//////////////////////////////////////////////////////////////////////////////////////////////////

// SystemComponents contains the settings of system components in the control or data plane of the Shoot cluster.
type SystemComponents struct {
	// CoreDNS contains the settings of the Core DNS components running in the data plane of the Shoot cluster.
	CoreDNS *CoreDNS
	// NodeLocalDNS contains the settings of the node local DNS components running in the data plane of the Shoot cluster.
	NodeLocalDNS *NodeLocalDNS
}

// CoreDNS contains the settings of the Core DNS components running in the data plane of the Shoot cluster.
type CoreDNS struct {
	// Autoscaling contains the settings related to autoscaling of the Core DNS components running in the data plane of the Shoot cluster.
	Autoscaling *CoreDNSAutoscaling
}

// CoreDNSAutoscaling contains the settings related to autoscaling of the Core DNS components running in the data plane of the Shoot cluster.
type CoreDNSAutoscaling struct {
	// The mode of the autoscaling to be used for the Core DNS components running in the data plane of the Shoot cluster.
	// Supported values are `horizontal` and `cluster-proportional`.
	Mode CoreDNSAutoscalingMode
}

// CoreDNSAutoscalingMode is a type alias for the Core DNS autoscaling mode string.
type CoreDNSAutoscalingMode string

const (
	// CoreDNSAutoscalingModeHorizontal is a constant for horizontal Core DNS autoscaling mode.
	CoreDNSAutoscalingModeHorizontal CoreDNSAutoscalingMode = "horizontal"
	// CoreDNSAutoscalingModeClusterProportional is a constant for cluster-proportional Core DNS autoscaling mode.
	CoreDNSAutoscalingModeClusterProportional CoreDNSAutoscalingMode = "cluster-proportional"
)

// NodeLocalDNS contains the settings of the node local DNS components running in the data plane of the Shoot cluster.
type NodeLocalDNS struct {
	// Enabled indicates whether node local DNS is enabled or not.
	Enabled bool
	// ForceTCPToClusterDNS indicates whether the connection from the node local DNS to the cluster DNS (Core DNS) will be forced to TCP or not.
	// Default, if unspecified, is to enforce TCP.
	ForceTCPToClusterDNS *bool
	// ForceTCPToUpstreamDNS indicates whether the connection from the node local DNS to the upstream DNS (infrastructure DNS) will be forced to TCP or not.
	// Default, if unspecified, is to enforce TCP.
	ForceTCPToUpstreamDNS *bool
}

//////////////////////////////////////////////////////////////////////////////////////////////////
// Other/miscellaneous constants and types                                                      //
//////////////////////////////////////////////////////////////////////////////////////////////////

const (
	// ShootEventImageVersionMaintenance indicates that a maintenance operation regarding the image version has been performed.
	ShootEventImageVersionMaintenance = "MachineImageVersionMaintenance"
	// ShootEventK8sVersionMaintenance indicates that a maintenance operation regarding the K8s version has been performed.
	ShootEventK8sVersionMaintenance = "KubernetesVersionMaintenance"
	// ShootEventHibernationEnabled indicates that hibernation started.
	ShootEventHibernationEnabled = "Hibernated"
	// ShootEventHibernationDisabled indicates that hibernation ended.
	ShootEventHibernationDisabled = "WokenUp"
	// ShootEventSchedulingSuccessful indicates that a scheduling decision was taken successfully.
	ShootEventSchedulingSuccessful = "SchedulingSuccessful"
	// ShootEventSchedulingFailed indicates that a scheduling decision failed.
	ShootEventSchedulingFailed = "SchedulingFailed"
)

const (
	// ShootAPIServerAvailable is a constant for a condition type indicating that the Shoot cluster's API server is available.
	ShootAPIServerAvailable ConditionType = "APIServerAvailable"
	// ShootControlPlaneHealthy is a constant for a condition type indicating the control plane health.
	ShootControlPlaneHealthy ConditionType = "ControlPlaneHealthy"
	// ShootEveryNodeReady is a constant for a condition type indicating the node health.
	ShootEveryNodeReady ConditionType = "EveryNodeReady"
	// ShootSystemComponentsHealthy is a constant for a condition type indicating the system components health.
	ShootSystemComponentsHealthy ConditionType = "SystemComponentsHealthy"
	// ShootHibernationPossible is a constant for a condition type indicating whether the Shoot can be hibernated.
	ShootHibernationPossible ConditionType = "HibernationPossible"
	// ShootMaintenancePreconditionsSatisfied is a constant for a condition type indicating whether all preconditions
	// for a shoot maintenance operation are satisfied.
	ShootMaintenancePreconditionsSatisfied ConditionType = "MaintenancePreconditionsSatisfied"
)

// DNSUnmanaged is a constant for the 'unmanaged' DNS provider.
const DNSUnmanaged string = "unmanaged"

// ShootPurpose is a type alias for string.
type ShootPurpose string

const (
	// ShootPurposeEvaluation is a constant for the evaluation purpose.
	ShootPurposeEvaluation ShootPurpose = "evaluation"
	// ShootPurposeTesting is a constant for the testing purpose.
	ShootPurposeTesting ShootPurpose = "testing"
	// ShootPurposeDevelopment is a constant for the development purpose.
	ShootPurposeDevelopment ShootPurpose = "development"
	// ShootPurposeProduction is a constant for the production purpose.
	ShootPurposeProduction ShootPurpose = "production"
	// ShootPurposeInfrastructure is a constant for the infrastructure purpose.
	ShootPurposeInfrastructure ShootPurpose = "infrastructure"
)
