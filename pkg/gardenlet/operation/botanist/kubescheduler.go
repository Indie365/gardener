// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package botanist

import (
	"github.com/gardener/gardener/imagevector"
	kubescheduler "github.com/gardener/gardener/pkg/component/kubernetes/scheduler"
	imagevectorutils "github.com/gardener/gardener/pkg/utils/imagevector"
)

// DefaultKubeScheduler returns a deployer for the kube-scheduler.
func (b *Botanist) DefaultKubeScheduler() (kubescheduler.Interface, error) {
	image, err := imagevector.ImageVector().FindImage(imagevector.ImageNameKubeScheduler, imagevectorutils.RuntimeVersion(b.SeedVersion()), imagevectorutils.TargetVersion(b.ShootVersion()))
	if err != nil {
		return nil, err
	}

	return kubescheduler.New(
		b.SeedClientSet.Client(),
		b.Shoot.SeedNamespace,
		b.SecretsManager,
		b.Seed.KubernetesVersion,
		b.Shoot.KubernetesVersion,
		image.String(),
		b.Shoot.GetReplicas(1),
		b.Shoot.GetInfo().Spec.Kubernetes.KubeScheduler,
	), nil
}