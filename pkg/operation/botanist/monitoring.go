// SPDX-FileCopyrightText: 2018 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package botanist

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"github.com/gardener/gardener/imagevector"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/component"
	"github.com/gardener/gardener/pkg/component/monitoring"
	gardenlethelper "github.com/gardener/gardener/pkg/gardenlet/apis/config/helper"
	gardenerutils "github.com/gardener/gardener/pkg/utils/gardener"
)

// DefaultMonitoring creates a new monitoring component.
func (b *Botanist) DefaultMonitoring() (monitoring.Interface, error) {
	imageAlertmanager, err := imagevector.ImageVector().FindImage(imagevector.ImageNameAlertmanager)
	if err != nil {
		return nil, err
	}
	imageBlackboxExporter, err := imagevector.ImageVector().FindImage(imagevector.ImageNameBlackboxExporter)
	if err != nil {
		return nil, err
	}
	imageConfigmapReloader, err := imagevector.ImageVector().FindImage(imagevector.ImageNameConfigmapReloader)
	if err != nil {
		return nil, err
	}
	imagePrometheus, err := imagevector.ImageVector().FindImage(imagevector.ImageNamePrometheus)
	if err != nil {
		return nil, err
	}

	var alertingSecrets []*corev1.Secret
	for _, key := range b.GetSecretKeysOfRole(v1beta1constants.GardenRoleAlerting) {
		alertingSecrets = append(alertingSecrets, b.LoadSecret(key))
	}

	values := monitoring.Values{
		AlertingSecrets:              alertingSecrets,
		AlertmanagerEnabled:          b.Shoot.WantsAlertmanager,
		APIServerDomain:              gardenerutils.GetAPIServerDomain(b.Shoot.InternalClusterDomain),
		APIServerHost:                b.SeedClientSet.RESTConfig().Host,
		Config:                       b.Config.Monitoring,
		GlobalShootRemoteWriteSecret: b.LoadSecret(v1beta1constants.GardenRoleGlobalShootRemoteWriteMonitoring),
		IgnoreAlerts:                 b.Shoot.IgnoreAlerts,
		ImageAlertmanager:            imageAlertmanager.String(),
		ImageBlackboxExporter:        imageBlackboxExporter.String(),
		ImageConfigmapReloader:       imageConfigmapReloader.String(),
		ImagePrometheus:              imagePrometheus.String(),
		IngressHostAlertmanager:      b.ComputeAlertManagerHost(),
		IngressHostPrometheus:        b.ComputePrometheusHost(),
		IsWorkerless:                 b.Shoot.IsWorkerless,
		KubernetesVersion:            b.Shoot.GetInfo().Spec.Kubernetes.Version,
		MonitoringConfig:             b.Shoot.GetInfo().Spec.Monitoring,
		NodeLocalDNSEnabled:          b.Shoot.NodeLocalDNSEnabled,
		ProjectName:                  b.Garden.Project.Name,
		Replicas:                     b.Shoot.GetReplicas(1),
		RuntimeProviderType:          b.Seed.GetInfo().Spec.Provider.Type,
		RuntimeRegion:                b.Seed.GetInfo().Spec.Provider.Region,
		StorageCapacityAlertmanager:  b.Seed.GetValidVolumeSize("1Gi"),
		TargetName:                   b.Shoot.GetInfo().Name,
		TargetProviderType:           b.Shoot.GetInfo().Spec.Provider.Type,
		WildcardCertName:             nil,
	}

	if b.Shoot.Networks != nil {
		if services := b.Shoot.Networks.Services; services != nil {
			values.ServiceNetworkCIDR = pointer.String(services.String())
		}
		if pods := b.Shoot.Networks.Pods; pods != nil {
			values.PodNetworkCIDR = pointer.String(pods.String())
		}
		if apiServer := b.Shoot.Networks.APIServer; apiServer != nil {
			values.APIServerServiceIP = pointer.String(apiServer.String())
		}
	}

	if b.Shoot.GetInfo().Spec.Networking != nil {
		values.NodeNetworkCIDR = b.Shoot.GetInfo().Spec.Networking.Nodes
	}

	return monitoring.New(
		b.SeedClientSet.Client(),
		b.SeedClientSet.ChartApplier(),
		b.SecretsManager,
		b.Shoot.SeedNamespace,
		values,
	), nil
}

// DeployMonitoring installs the Helm release "seed-monitoring" in the Seed clusters. It comprises components
// to monitor the Shoot cluster whose control plane runs in the Seed cluster.
func (b *Botanist) DeployMonitoring(ctx context.Context) error {
	if !b.IsShootMonitoringEnabled() {
		return b.Shoot.Components.Monitoring.Monitoring.Destroy(ctx)
	}

	if b.ControlPlaneWildcardCert != nil {
		b.Operation.Shoot.Components.Monitoring.Monitoring.SetWildcardCertName(pointer.String(b.ControlPlaneWildcardCert.GetName()))
	}
	b.Shoot.Components.Monitoring.Monitoring.SetNamespaceUID(b.SeedNamespaceObject.UID)
	b.Shoot.Components.Monitoring.Monitoring.SetComponents(b.getMonitoringComponents())
	return b.Shoot.Components.Monitoring.Monitoring.Deploy(ctx)
}

func (b *Botanist) getMonitoringComponents() []component.MonitoringComponent {
	// Fetch component-specific monitoring configuration
	monitoringComponents := []component.MonitoringComponent{
		b.Shoot.Components.ControlPlane.EtcdMain,
		b.Shoot.Components.ControlPlane.EtcdEvents,
		b.Shoot.Components.ControlPlane.KubeAPIServer,
		b.Shoot.Components.ControlPlane.KubeControllerManager,
		b.Shoot.Components.ControlPlane.KubeStateMetrics,
		b.Shoot.Components.ControlPlane.ResourceManager,
	}

	if b.Shoot.IsShootControlPlaneLoggingEnabled(b.Config) && gardenlethelper.IsValiEnabled(b.Config) {
		monitoringComponents = append(monitoringComponents, b.Shoot.Components.Logging.Vali)
	}

	if !b.Shoot.IsWorkerless {
		monitoringComponents = append(monitoringComponents,
			b.Shoot.Components.ControlPlane.KubeScheduler,
			b.Shoot.Components.ControlPlane.MachineControllerManager,
			b.Shoot.Components.ControlPlane.VPNSeedServer,
			b.Shoot.Components.SystemComponents.BlackboxExporter,
			b.Shoot.Components.SystemComponents.CoreDNS,
			b.Shoot.Components.SystemComponents.KubeProxy,
			b.Shoot.Components.SystemComponents.NodeExporter,
			b.Shoot.Components.SystemComponents.VPNShoot,
		)

		if b.ShootUsesDNS() {
			monitoringComponents = append(monitoringComponents, b.Shoot.Components.SystemComponents.APIServerProxy)
		}

		if b.Shoot.NodeLocalDNSEnabled {
			monitoringComponents = append(monitoringComponents, b.Shoot.Components.SystemComponents.NodeLocalDNS)
		}

		if b.Shoot.WantsClusterAutoscaler {
			monitoringComponents = append(monitoringComponents, b.Shoot.Components.ControlPlane.ClusterAutoscaler)
		}
	}

	return monitoringComponents
}
