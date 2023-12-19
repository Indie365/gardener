// SPDX-FileCopyrightText: 2018 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package botanist

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/gardener/imagevector"
	gardencore "github.com/gardener/gardener/pkg/apis/core"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/component"
	"github.com/gardener/gardener/pkg/component/logging/eventlogger"
	"github.com/gardener/gardener/pkg/component/logging/vali"
	"github.com/gardener/gardener/pkg/component/shared"
	"github.com/gardener/gardener/pkg/features"
	gardenlethelper "github.com/gardener/gardener/pkg/gardenlet/apis/config/helper"
	"github.com/gardener/gardener/pkg/operation/common"
	imagevectorutils "github.com/gardener/gardener/pkg/utils/imagevector"
	kubernetesutils "github.com/gardener/gardener/pkg/utils/kubernetes"
)

// DeployLogging will install the logging stack for the Shoot in the Seed clusters.
func (b *Botanist) DeployLogging(ctx context.Context) error {
	if !b.Shoot.IsShootControlPlaneLoggingEnabled(b.Config) {
		return b.DestroySeedLogging(ctx)
	}

	// TODO(rickardsjp, istvanballok): Remove in release v1.77 once the Loki to Vali migration is complete.
	if exists, err := b.lokiPvcExists(ctx); err != nil {
		return err
	} else if exists {
		if err := b.destroyLokiBasedShootLoggingStackRetainingPvc(ctx); err != nil {
			return err
		}
		// If a Loki PVC exists, rename it to Vali.
		if err := b.renameLokiPvcToVali(ctx); err != nil {
			return err
		}
	}

	if b.isShootEventLoggerEnabled() {
		if err := b.Shoot.Components.Logging.EventLogger.Deploy(ctx); err != nil {
			return err
		}
	} else {
		if err := b.Shoot.Components.Logging.EventLogger.Destroy(ctx); err != nil {
			return err
		}
	}

	// check if vali is enabled in gardenlet config, default is true
	if !gardenlethelper.IsValiEnabled(b.Config) {
		return b.Shoot.Components.Logging.Vali.Destroy(ctx)
	}

	return b.Shoot.Components.Logging.Vali.Deploy(ctx)
}

// DestroySeedLogging will uninstall the logging stack for the Shoot in the Seed clusters.
func (b *Botanist) DestroySeedLogging(ctx context.Context) error {
	if err := b.Shoot.Components.Logging.EventLogger.Destroy(ctx); err != nil {
		return err
	}

	return b.Shoot.Components.Logging.Vali.Destroy(ctx)
}

func (b *Botanist) lokiPvcExists(ctx context.Context) (bool, error) {
	return common.LokiPvcExists(ctx, b.SeedClientSet.Client(), b.Shoot.SeedNamespace, b.Logger)
}

func (b *Botanist) renameLokiPvcToVali(ctx context.Context) error {
	return common.RenameLokiPvcToValiPvc(ctx, b.SeedClientSet.Client(), b.Shoot.SeedNamespace, b.Logger)
}

func (b *Botanist) destroyLokiBasedShootLoggingStackRetainingPvc(ctx context.Context) error {
	if err := b.destroyLokiBasedShootNodeLogging(ctx); err != nil {
		return err
	}

	// The EventLogger is not dependent on Loki/Vali and therefore doesn't need to be deleted.
	// if err := b.Shoot.Components.Logging.EventLogger.Destroy(ctx); err != nil {
	// 	return err
	// }

	return common.DeleteLokiRetainPvc(ctx, b.SeedClientSet.Client(), b.Shoot.SeedNamespace, b.Logger)
}

func (b *Botanist) destroyLokiBasedShootNodeLogging(ctx context.Context) error {
	return kubernetesutils.DeleteObjects(ctx, b.SeedClientSet.Client(),
		&networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "loki", Namespace: b.Shoot.SeedNamespace}},
		&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: "allow-from-prometheus-to-loki-telegraf", Namespace: b.Shoot.SeedNamespace}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "telegraf-config", Namespace: b.Shoot.SeedNamespace}},
	)
}

func (b *Botanist) isShootNodeLoggingEnabled() bool {
	if b.Shoot != nil && !b.Shoot.IsWorkerless && b.Shoot.IsShootControlPlaneLoggingEnabled(b.Config) &&
		gardenlethelper.IsValiEnabled(b.Config) && b.Config != nil &&
		b.Config.Logging != nil && b.Config.Logging.ShootNodeLogging != nil {
		for _, purpose := range b.Config.Logging.ShootNodeLogging.ShootPurposes {
			if gardencore.ShootPurpose(b.Shoot.Purpose) == purpose {
				return true
			}
		}
	}
	return false
}

func (b *Botanist) isShootEventLoggerEnabled() bool {
	return b.Shoot != nil && b.Shoot.IsShootControlPlaneLoggingEnabled(b.Config) && gardenlethelper.IsEventLoggingEnabled(b.Config)
}

// DefaultEventLogger returns a deployer for the shoot-event-logger.
func (b *Botanist) DefaultEventLogger() (component.Deployer, error) {
	imageEventLogger, err := imagevector.ImageVector().FindImage(imagevector.ImageNameEventLogger, imagevectorutils.RuntimeVersion(b.SeedVersion()), imagevectorutils.TargetVersion(b.ShootVersion()))
	if err != nil {
		return nil, err
	}

	return eventlogger.New(
		b.SeedClientSet.Client(),
		b.Shoot.SeedNamespace,
		b.SecretsManager,
		eventlogger.Values{
			Image:    imageEventLogger.String(),
			Replicas: b.Shoot.GetReplicas(1),
		},
	)
}

// DefaultVali returns a deployer for Vali.
func (b *Botanist) DefaultVali() (vali.Interface, error) {
	hvpaEnabled := features.DefaultFeatureGate.Enabled(features.HVPA)
	if b.ManagedSeed != nil {
		hvpaEnabled = features.DefaultFeatureGate.Enabled(features.HVPAForShootedSeed)
	}

	return shared.NewVali(
		b.SeedClientSet.Client(),
		b.Shoot.SeedNamespace,
		b.SecretsManager,
		component.ClusterTypeShoot,
		b.Shoot.GetReplicas(1),
		b.isShootNodeLoggingEnabled(),
		v1beta1constants.PriorityClassNameShootControlPlane100,
		nil,
		b.ComputeValiHost(),
		hvpaEnabled,
		nil,
	)
}
