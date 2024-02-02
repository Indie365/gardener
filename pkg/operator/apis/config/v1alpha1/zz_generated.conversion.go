//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Code generated by conversion-gen. DO NOT EDIT.

package v1alpha1

import (
	unsafe "unsafe"

	apisconfig "github.com/gardener/gardener/pkg/gardenlet/apis/config"
	configv1alpha1 "github.com/gardener/gardener/pkg/gardenlet/apis/config/v1alpha1"
	config "github.com/gardener/gardener/pkg/operator/apis/config"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
	componentbaseconfig "k8s.io/component-base/config"
	componentbaseconfigv1alpha1 "k8s.io/component-base/config/v1alpha1"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*ConditionThreshold)(nil), (*config.ConditionThreshold)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_ConditionThreshold_To_config_ConditionThreshold(a.(*ConditionThreshold), b.(*config.ConditionThreshold), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*config.ConditionThreshold)(nil), (*ConditionThreshold)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_config_ConditionThreshold_To_v1alpha1_ConditionThreshold(a.(*config.ConditionThreshold), b.(*ConditionThreshold), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*ControllerConfiguration)(nil), (*config.ControllerConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_ControllerConfiguration_To_config_ControllerConfiguration(a.(*ControllerConfiguration), b.(*config.ControllerConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*config.ControllerConfiguration)(nil), (*ControllerConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_config_ControllerConfiguration_To_v1alpha1_ControllerConfiguration(a.(*config.ControllerConfiguration), b.(*ControllerConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*GardenCareControllerConfiguration)(nil), (*config.GardenCareControllerConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_GardenCareControllerConfiguration_To_config_GardenCareControllerConfiguration(a.(*GardenCareControllerConfiguration), b.(*config.GardenCareControllerConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*config.GardenCareControllerConfiguration)(nil), (*GardenCareControllerConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_config_GardenCareControllerConfiguration_To_v1alpha1_GardenCareControllerConfiguration(a.(*config.GardenCareControllerConfiguration), b.(*GardenCareControllerConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*GardenControllerConfig)(nil), (*config.GardenControllerConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_GardenControllerConfig_To_config_GardenControllerConfig(a.(*GardenControllerConfig), b.(*config.GardenControllerConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*config.GardenControllerConfig)(nil), (*GardenControllerConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_config_GardenControllerConfig_To_v1alpha1_GardenControllerConfig(a.(*config.GardenControllerConfig), b.(*GardenControllerConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*NetworkPolicyControllerConfiguration)(nil), (*config.NetworkPolicyControllerConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_NetworkPolicyControllerConfiguration_To_config_NetworkPolicyControllerConfiguration(a.(*NetworkPolicyControllerConfiguration), b.(*config.NetworkPolicyControllerConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*config.NetworkPolicyControllerConfiguration)(nil), (*NetworkPolicyControllerConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_config_NetworkPolicyControllerConfiguration_To_v1alpha1_NetworkPolicyControllerConfiguration(a.(*config.NetworkPolicyControllerConfiguration), b.(*NetworkPolicyControllerConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*NodeTolerationConfiguration)(nil), (*config.NodeTolerationConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_NodeTolerationConfiguration_To_config_NodeTolerationConfiguration(a.(*NodeTolerationConfiguration), b.(*config.NodeTolerationConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*config.NodeTolerationConfiguration)(nil), (*NodeTolerationConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_config_NodeTolerationConfiguration_To_v1alpha1_NodeTolerationConfiguration(a.(*config.NodeTolerationConfiguration), b.(*NodeTolerationConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*OperatorConfiguration)(nil), (*config.OperatorConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_OperatorConfiguration_To_config_OperatorConfiguration(a.(*OperatorConfiguration), b.(*config.OperatorConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*config.OperatorConfiguration)(nil), (*OperatorConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_config_OperatorConfiguration_To_v1alpha1_OperatorConfiguration(a.(*config.OperatorConfiguration), b.(*OperatorConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*Server)(nil), (*config.Server)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_Server_To_config_Server(a.(*Server), b.(*config.Server), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*config.Server)(nil), (*Server)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_config_Server_To_v1alpha1_Server(a.(*config.Server), b.(*Server), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*ServerConfiguration)(nil), (*config.ServerConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_ServerConfiguration_To_config_ServerConfiguration(a.(*ServerConfiguration), b.(*config.ServerConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*config.ServerConfiguration)(nil), (*ServerConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_config_ServerConfiguration_To_v1alpha1_ServerConfiguration(a.(*config.ServerConfiguration), b.(*ServerConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*VPAEvictionRequirementsControllerConfiguration)(nil), (*config.VPAEvictionRequirementsControllerConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_VPAEvictionRequirementsControllerConfiguration_To_config_VPAEvictionRequirementsControllerConfiguration(a.(*VPAEvictionRequirementsControllerConfiguration), b.(*config.VPAEvictionRequirementsControllerConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*config.VPAEvictionRequirementsControllerConfiguration)(nil), (*VPAEvictionRequirementsControllerConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_config_VPAEvictionRequirementsControllerConfiguration_To_v1alpha1_VPAEvictionRequirementsControllerConfiguration(a.(*config.VPAEvictionRequirementsControllerConfiguration), b.(*VPAEvictionRequirementsControllerConfiguration), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1alpha1_ConditionThreshold_To_config_ConditionThreshold(in *ConditionThreshold, out *config.ConditionThreshold, s conversion.Scope) error {
	out.Type = in.Type
	out.Duration = in.Duration
	return nil
}

// Convert_v1alpha1_ConditionThreshold_To_config_ConditionThreshold is an autogenerated conversion function.
func Convert_v1alpha1_ConditionThreshold_To_config_ConditionThreshold(in *ConditionThreshold, out *config.ConditionThreshold, s conversion.Scope) error {
	return autoConvert_v1alpha1_ConditionThreshold_To_config_ConditionThreshold(in, out, s)
}

func autoConvert_config_ConditionThreshold_To_v1alpha1_ConditionThreshold(in *config.ConditionThreshold, out *ConditionThreshold, s conversion.Scope) error {
	out.Type = in.Type
	out.Duration = in.Duration
	return nil
}

// Convert_config_ConditionThreshold_To_v1alpha1_ConditionThreshold is an autogenerated conversion function.
func Convert_config_ConditionThreshold_To_v1alpha1_ConditionThreshold(in *config.ConditionThreshold, out *ConditionThreshold, s conversion.Scope) error {
	return autoConvert_config_ConditionThreshold_To_v1alpha1_ConditionThreshold(in, out, s)
}

func autoConvert_v1alpha1_ControllerConfiguration_To_config_ControllerConfiguration(in *ControllerConfiguration, out *config.ControllerConfiguration, s conversion.Scope) error {
	if err := Convert_v1alpha1_GardenControllerConfig_To_config_GardenControllerConfig(&in.Garden, &out.Garden, s); err != nil {
		return err
	}
	if err := Convert_v1alpha1_GardenCareControllerConfiguration_To_config_GardenCareControllerConfiguration(&in.GardenCare, &out.GardenCare, s); err != nil {
		return err
	}
	if err := Convert_v1alpha1_NetworkPolicyControllerConfiguration_To_config_NetworkPolicyControllerConfiguration(&in.NetworkPolicy, &out.NetworkPolicy, s); err != nil {
		return err
	}
	if err := Convert_v1alpha1_VPAEvictionRequirementsControllerConfiguration_To_config_VPAEvictionRequirementsControllerConfiguration(&in.VPAEvictionRequirements, &out.VPAEvictionRequirements, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha1_ControllerConfiguration_To_config_ControllerConfiguration is an autogenerated conversion function.
func Convert_v1alpha1_ControllerConfiguration_To_config_ControllerConfiguration(in *ControllerConfiguration, out *config.ControllerConfiguration, s conversion.Scope) error {
	return autoConvert_v1alpha1_ControllerConfiguration_To_config_ControllerConfiguration(in, out, s)
}

func autoConvert_config_ControllerConfiguration_To_v1alpha1_ControllerConfiguration(in *config.ControllerConfiguration, out *ControllerConfiguration, s conversion.Scope) error {
	if err := Convert_config_GardenControllerConfig_To_v1alpha1_GardenControllerConfig(&in.Garden, &out.Garden, s); err != nil {
		return err
	}
	if err := Convert_config_GardenCareControllerConfiguration_To_v1alpha1_GardenCareControllerConfiguration(&in.GardenCare, &out.GardenCare, s); err != nil {
		return err
	}
	if err := Convert_config_NetworkPolicyControllerConfiguration_To_v1alpha1_NetworkPolicyControllerConfiguration(&in.NetworkPolicy, &out.NetworkPolicy, s); err != nil {
		return err
	}
	if err := Convert_config_VPAEvictionRequirementsControllerConfiguration_To_v1alpha1_VPAEvictionRequirementsControllerConfiguration(&in.VPAEvictionRequirements, &out.VPAEvictionRequirements, s); err != nil {
		return err
	}
	return nil
}

// Convert_config_ControllerConfiguration_To_v1alpha1_ControllerConfiguration is an autogenerated conversion function.
func Convert_config_ControllerConfiguration_To_v1alpha1_ControllerConfiguration(in *config.ControllerConfiguration, out *ControllerConfiguration, s conversion.Scope) error {
	return autoConvert_config_ControllerConfiguration_To_v1alpha1_ControllerConfiguration(in, out, s)
}

func autoConvert_v1alpha1_GardenCareControllerConfiguration_To_config_GardenCareControllerConfiguration(in *GardenCareControllerConfiguration, out *config.GardenCareControllerConfiguration, s conversion.Scope) error {
	out.SyncPeriod = (*v1.Duration)(unsafe.Pointer(in.SyncPeriod))
	out.ConditionThresholds = *(*[]config.ConditionThreshold)(unsafe.Pointer(&in.ConditionThresholds))
	return nil
}

// Convert_v1alpha1_GardenCareControllerConfiguration_To_config_GardenCareControllerConfiguration is an autogenerated conversion function.
func Convert_v1alpha1_GardenCareControllerConfiguration_To_config_GardenCareControllerConfiguration(in *GardenCareControllerConfiguration, out *config.GardenCareControllerConfiguration, s conversion.Scope) error {
	return autoConvert_v1alpha1_GardenCareControllerConfiguration_To_config_GardenCareControllerConfiguration(in, out, s)
}

func autoConvert_config_GardenCareControllerConfiguration_To_v1alpha1_GardenCareControllerConfiguration(in *config.GardenCareControllerConfiguration, out *GardenCareControllerConfiguration, s conversion.Scope) error {
	out.SyncPeriod = (*v1.Duration)(unsafe.Pointer(in.SyncPeriod))
	out.ConditionThresholds = *(*[]ConditionThreshold)(unsafe.Pointer(&in.ConditionThresholds))
	return nil
}

// Convert_config_GardenCareControllerConfiguration_To_v1alpha1_GardenCareControllerConfiguration is an autogenerated conversion function.
func Convert_config_GardenCareControllerConfiguration_To_v1alpha1_GardenCareControllerConfiguration(in *config.GardenCareControllerConfiguration, out *GardenCareControllerConfiguration, s conversion.Scope) error {
	return autoConvert_config_GardenCareControllerConfiguration_To_v1alpha1_GardenCareControllerConfiguration(in, out, s)
}

func autoConvert_v1alpha1_GardenControllerConfig_To_config_GardenControllerConfig(in *GardenControllerConfig, out *config.GardenControllerConfig, s conversion.Scope) error {
	out.ConcurrentSyncs = (*int)(unsafe.Pointer(in.ConcurrentSyncs))
	out.SyncPeriod = (*v1.Duration)(unsafe.Pointer(in.SyncPeriod))
	out.ETCDConfig = (*apisconfig.ETCDConfig)(unsafe.Pointer(in.ETCDConfig))
	return nil
}

// Convert_v1alpha1_GardenControllerConfig_To_config_GardenControllerConfig is an autogenerated conversion function.
func Convert_v1alpha1_GardenControllerConfig_To_config_GardenControllerConfig(in *GardenControllerConfig, out *config.GardenControllerConfig, s conversion.Scope) error {
	return autoConvert_v1alpha1_GardenControllerConfig_To_config_GardenControllerConfig(in, out, s)
}

func autoConvert_config_GardenControllerConfig_To_v1alpha1_GardenControllerConfig(in *config.GardenControllerConfig, out *GardenControllerConfig, s conversion.Scope) error {
	out.ConcurrentSyncs = (*int)(unsafe.Pointer(in.ConcurrentSyncs))
	out.SyncPeriod = (*v1.Duration)(unsafe.Pointer(in.SyncPeriod))
	out.ETCDConfig = (*configv1alpha1.ETCDConfig)(unsafe.Pointer(in.ETCDConfig))
	return nil
}

// Convert_config_GardenControllerConfig_To_v1alpha1_GardenControllerConfig is an autogenerated conversion function.
func Convert_config_GardenControllerConfig_To_v1alpha1_GardenControllerConfig(in *config.GardenControllerConfig, out *GardenControllerConfig, s conversion.Scope) error {
	return autoConvert_config_GardenControllerConfig_To_v1alpha1_GardenControllerConfig(in, out, s)
}

func autoConvert_v1alpha1_NetworkPolicyControllerConfiguration_To_config_NetworkPolicyControllerConfiguration(in *NetworkPolicyControllerConfiguration, out *config.NetworkPolicyControllerConfiguration, s conversion.Scope) error {
	out.ConcurrentSyncs = (*int)(unsafe.Pointer(in.ConcurrentSyncs))
	out.AdditionalNamespaceSelectors = *(*[]v1.LabelSelector)(unsafe.Pointer(&in.AdditionalNamespaceSelectors))
	return nil
}

// Convert_v1alpha1_NetworkPolicyControllerConfiguration_To_config_NetworkPolicyControllerConfiguration is an autogenerated conversion function.
func Convert_v1alpha1_NetworkPolicyControllerConfiguration_To_config_NetworkPolicyControllerConfiguration(in *NetworkPolicyControllerConfiguration, out *config.NetworkPolicyControllerConfiguration, s conversion.Scope) error {
	return autoConvert_v1alpha1_NetworkPolicyControllerConfiguration_To_config_NetworkPolicyControllerConfiguration(in, out, s)
}

func autoConvert_config_NetworkPolicyControllerConfiguration_To_v1alpha1_NetworkPolicyControllerConfiguration(in *config.NetworkPolicyControllerConfiguration, out *NetworkPolicyControllerConfiguration, s conversion.Scope) error {
	out.ConcurrentSyncs = (*int)(unsafe.Pointer(in.ConcurrentSyncs))
	out.AdditionalNamespaceSelectors = *(*[]v1.LabelSelector)(unsafe.Pointer(&in.AdditionalNamespaceSelectors))
	return nil
}

// Convert_config_NetworkPolicyControllerConfiguration_To_v1alpha1_NetworkPolicyControllerConfiguration is an autogenerated conversion function.
func Convert_config_NetworkPolicyControllerConfiguration_To_v1alpha1_NetworkPolicyControllerConfiguration(in *config.NetworkPolicyControllerConfiguration, out *NetworkPolicyControllerConfiguration, s conversion.Scope) error {
	return autoConvert_config_NetworkPolicyControllerConfiguration_To_v1alpha1_NetworkPolicyControllerConfiguration(in, out, s)
}

func autoConvert_v1alpha1_NodeTolerationConfiguration_To_config_NodeTolerationConfiguration(in *NodeTolerationConfiguration, out *config.NodeTolerationConfiguration, s conversion.Scope) error {
	out.DefaultNotReadyTolerationSeconds = (*int64)(unsafe.Pointer(in.DefaultNotReadyTolerationSeconds))
	out.DefaultUnreachableTolerationSeconds = (*int64)(unsafe.Pointer(in.DefaultUnreachableTolerationSeconds))
	return nil
}

// Convert_v1alpha1_NodeTolerationConfiguration_To_config_NodeTolerationConfiguration is an autogenerated conversion function.
func Convert_v1alpha1_NodeTolerationConfiguration_To_config_NodeTolerationConfiguration(in *NodeTolerationConfiguration, out *config.NodeTolerationConfiguration, s conversion.Scope) error {
	return autoConvert_v1alpha1_NodeTolerationConfiguration_To_config_NodeTolerationConfiguration(in, out, s)
}

func autoConvert_config_NodeTolerationConfiguration_To_v1alpha1_NodeTolerationConfiguration(in *config.NodeTolerationConfiguration, out *NodeTolerationConfiguration, s conversion.Scope) error {
	out.DefaultNotReadyTolerationSeconds = (*int64)(unsafe.Pointer(in.DefaultNotReadyTolerationSeconds))
	out.DefaultUnreachableTolerationSeconds = (*int64)(unsafe.Pointer(in.DefaultUnreachableTolerationSeconds))
	return nil
}

// Convert_config_NodeTolerationConfiguration_To_v1alpha1_NodeTolerationConfiguration is an autogenerated conversion function.
func Convert_config_NodeTolerationConfiguration_To_v1alpha1_NodeTolerationConfiguration(in *config.NodeTolerationConfiguration, out *NodeTolerationConfiguration, s conversion.Scope) error {
	return autoConvert_config_NodeTolerationConfiguration_To_v1alpha1_NodeTolerationConfiguration(in, out, s)
}

func autoConvert_v1alpha1_OperatorConfiguration_To_config_OperatorConfiguration(in *OperatorConfiguration, out *config.OperatorConfiguration, s conversion.Scope) error {
	if err := componentbaseconfigv1alpha1.Convert_v1alpha1_ClientConnectionConfiguration_To_config_ClientConnectionConfiguration(&in.RuntimeClientConnection, &out.RuntimeClientConnection, s); err != nil {
		return err
	}
	if err := componentbaseconfigv1alpha1.Convert_v1alpha1_ClientConnectionConfiguration_To_config_ClientConnectionConfiguration(&in.VirtualClientConnection, &out.VirtualClientConnection, s); err != nil {
		return err
	}
	if err := componentbaseconfigv1alpha1.Convert_v1alpha1_LeaderElectionConfiguration_To_config_LeaderElectionConfiguration(&in.LeaderElection, &out.LeaderElection, s); err != nil {
		return err
	}
	out.LogLevel = in.LogLevel
	out.LogFormat = in.LogFormat
	if err := Convert_v1alpha1_ServerConfiguration_To_config_ServerConfiguration(&in.Server, &out.Server, s); err != nil {
		return err
	}
	if in.Debugging != nil {
		in, out := &in.Debugging, &out.Debugging
		*out = new(componentbaseconfig.DebuggingConfiguration)
		if err := componentbaseconfigv1alpha1.Convert_v1alpha1_DebuggingConfiguration_To_config_DebuggingConfiguration(*in, *out, s); err != nil {
			return err
		}
	} else {
		out.Debugging = nil
	}
	out.FeatureGates = *(*map[string]bool)(unsafe.Pointer(&in.FeatureGates))
	if err := Convert_v1alpha1_ControllerConfiguration_To_config_ControllerConfiguration(&in.Controllers, &out.Controllers, s); err != nil {
		return err
	}
	out.NodeToleration = (*config.NodeTolerationConfiguration)(unsafe.Pointer(in.NodeToleration))
	return nil
}

// Convert_v1alpha1_OperatorConfiguration_To_config_OperatorConfiguration is an autogenerated conversion function.
func Convert_v1alpha1_OperatorConfiguration_To_config_OperatorConfiguration(in *OperatorConfiguration, out *config.OperatorConfiguration, s conversion.Scope) error {
	return autoConvert_v1alpha1_OperatorConfiguration_To_config_OperatorConfiguration(in, out, s)
}

func autoConvert_config_OperatorConfiguration_To_v1alpha1_OperatorConfiguration(in *config.OperatorConfiguration, out *OperatorConfiguration, s conversion.Scope) error {
	if err := componentbaseconfigv1alpha1.Convert_config_ClientConnectionConfiguration_To_v1alpha1_ClientConnectionConfiguration(&in.RuntimeClientConnection, &out.RuntimeClientConnection, s); err != nil {
		return err
	}
	if err := componentbaseconfigv1alpha1.Convert_config_ClientConnectionConfiguration_To_v1alpha1_ClientConnectionConfiguration(&in.VirtualClientConnection, &out.VirtualClientConnection, s); err != nil {
		return err
	}
	if err := componentbaseconfigv1alpha1.Convert_config_LeaderElectionConfiguration_To_v1alpha1_LeaderElectionConfiguration(&in.LeaderElection, &out.LeaderElection, s); err != nil {
		return err
	}
	out.LogLevel = in.LogLevel
	out.LogFormat = in.LogFormat
	if err := Convert_config_ServerConfiguration_To_v1alpha1_ServerConfiguration(&in.Server, &out.Server, s); err != nil {
		return err
	}
	if in.Debugging != nil {
		in, out := &in.Debugging, &out.Debugging
		*out = new(componentbaseconfigv1alpha1.DebuggingConfiguration)
		if err := componentbaseconfigv1alpha1.Convert_config_DebuggingConfiguration_To_v1alpha1_DebuggingConfiguration(*in, *out, s); err != nil {
			return err
		}
	} else {
		out.Debugging = nil
	}
	out.FeatureGates = *(*map[string]bool)(unsafe.Pointer(&in.FeatureGates))
	if err := Convert_config_ControllerConfiguration_To_v1alpha1_ControllerConfiguration(&in.Controllers, &out.Controllers, s); err != nil {
		return err
	}
	out.NodeToleration = (*NodeTolerationConfiguration)(unsafe.Pointer(in.NodeToleration))
	return nil
}

// Convert_config_OperatorConfiguration_To_v1alpha1_OperatorConfiguration is an autogenerated conversion function.
func Convert_config_OperatorConfiguration_To_v1alpha1_OperatorConfiguration(in *config.OperatorConfiguration, out *OperatorConfiguration, s conversion.Scope) error {
	return autoConvert_config_OperatorConfiguration_To_v1alpha1_OperatorConfiguration(in, out, s)
}

func autoConvert_v1alpha1_Server_To_config_Server(in *Server, out *config.Server, s conversion.Scope) error {
	out.BindAddress = in.BindAddress
	out.Port = in.Port
	return nil
}

// Convert_v1alpha1_Server_To_config_Server is an autogenerated conversion function.
func Convert_v1alpha1_Server_To_config_Server(in *Server, out *config.Server, s conversion.Scope) error {
	return autoConvert_v1alpha1_Server_To_config_Server(in, out, s)
}

func autoConvert_config_Server_To_v1alpha1_Server(in *config.Server, out *Server, s conversion.Scope) error {
	out.BindAddress = in.BindAddress
	out.Port = in.Port
	return nil
}

// Convert_config_Server_To_v1alpha1_Server is an autogenerated conversion function.
func Convert_config_Server_To_v1alpha1_Server(in *config.Server, out *Server, s conversion.Scope) error {
	return autoConvert_config_Server_To_v1alpha1_Server(in, out, s)
}

func autoConvert_v1alpha1_ServerConfiguration_To_config_ServerConfiguration(in *ServerConfiguration, out *config.ServerConfiguration, s conversion.Scope) error {
	if err := Convert_v1alpha1_Server_To_config_Server(&in.Webhooks, &out.Webhooks, s); err != nil {
		return err
	}
	out.HealthProbes = (*config.Server)(unsafe.Pointer(in.HealthProbes))
	out.Metrics = (*config.Server)(unsafe.Pointer(in.Metrics))
	return nil
}

// Convert_v1alpha1_ServerConfiguration_To_config_ServerConfiguration is an autogenerated conversion function.
func Convert_v1alpha1_ServerConfiguration_To_config_ServerConfiguration(in *ServerConfiguration, out *config.ServerConfiguration, s conversion.Scope) error {
	return autoConvert_v1alpha1_ServerConfiguration_To_config_ServerConfiguration(in, out, s)
}

func autoConvert_config_ServerConfiguration_To_v1alpha1_ServerConfiguration(in *config.ServerConfiguration, out *ServerConfiguration, s conversion.Scope) error {
	if err := Convert_config_Server_To_v1alpha1_Server(&in.Webhooks, &out.Webhooks, s); err != nil {
		return err
	}
	out.HealthProbes = (*Server)(unsafe.Pointer(in.HealthProbes))
	out.Metrics = (*Server)(unsafe.Pointer(in.Metrics))
	return nil
}

// Convert_config_ServerConfiguration_To_v1alpha1_ServerConfiguration is an autogenerated conversion function.
func Convert_config_ServerConfiguration_To_v1alpha1_ServerConfiguration(in *config.ServerConfiguration, out *ServerConfiguration, s conversion.Scope) error {
	return autoConvert_config_ServerConfiguration_To_v1alpha1_ServerConfiguration(in, out, s)
}

func autoConvert_v1alpha1_VPAEvictionRequirementsControllerConfiguration_To_config_VPAEvictionRequirementsControllerConfiguration(in *VPAEvictionRequirementsControllerConfiguration, out *config.VPAEvictionRequirementsControllerConfiguration, s conversion.Scope) error {
	out.ConcurrentSyncs = (*int)(unsafe.Pointer(in.ConcurrentSyncs))
	return nil
}

// Convert_v1alpha1_VPAEvictionRequirementsControllerConfiguration_To_config_VPAEvictionRequirementsControllerConfiguration is an autogenerated conversion function.
func Convert_v1alpha1_VPAEvictionRequirementsControllerConfiguration_To_config_VPAEvictionRequirementsControllerConfiguration(in *VPAEvictionRequirementsControllerConfiguration, out *config.VPAEvictionRequirementsControllerConfiguration, s conversion.Scope) error {
	return autoConvert_v1alpha1_VPAEvictionRequirementsControllerConfiguration_To_config_VPAEvictionRequirementsControllerConfiguration(in, out, s)
}

func autoConvert_config_VPAEvictionRequirementsControllerConfiguration_To_v1alpha1_VPAEvictionRequirementsControllerConfiguration(in *config.VPAEvictionRequirementsControllerConfiguration, out *VPAEvictionRequirementsControllerConfiguration, s conversion.Scope) error {
	out.ConcurrentSyncs = (*int)(unsafe.Pointer(in.ConcurrentSyncs))
	return nil
}

// Convert_config_VPAEvictionRequirementsControllerConfiguration_To_v1alpha1_VPAEvictionRequirementsControllerConfiguration is an autogenerated conversion function.
func Convert_config_VPAEvictionRequirementsControllerConfiguration_To_v1alpha1_VPAEvictionRequirementsControllerConfiguration(in *config.VPAEvictionRequirementsControllerConfiguration, out *VPAEvictionRequirementsControllerConfiguration, s conversion.Scope) error {
	return autoConvert_config_VPAEvictionRequirementsControllerConfiguration_To_v1alpha1_VPAEvictionRequirementsControllerConfiguration(in, out, s)
}
