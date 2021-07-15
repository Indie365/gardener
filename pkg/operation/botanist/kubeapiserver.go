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

package botanist

import (
	"context"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/client/kubernetes/clientmap/keys"
	"github.com/gardener/gardener/pkg/features"
	gardenletfeatures "github.com/gardener/gardener/pkg/gardenlet/features"
	"github.com/gardener/gardener/pkg/operation/botanist/component/kubeapiserver"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// DefaultKubeAPIServer returns a deployer for the kube-apiserver.
func (b *Botanist) DefaultKubeAPIServer() (kubeapiserver.Interface, error) {
	return kubeapiserver.New(
		b.K8sSeedClient,
		b.Shoot.SeedNamespace,
		kubeapiserver.Values{
			Autoscaling: b.computeKubeAPIServerAutoscalingConfig(),
			SNI: kubeapiserver.SNIConfig{
				PodMutatorEnabled: b.APIServerSNIPodMutatorEnabled(),
			},
			Version: b.Shoot.KubernetesVersion,
		},
	), nil
}

func (b *Botanist) computeKubeAPIServerAutoscalingConfig() kubeapiserver.AutoscalingConfig {
	var (
		hvpaEnabled               = gardenletfeatures.FeatureGate.Enabled(features.HVPA)
		useMemoryMetricForHvpaHPA = false
		scaleDownDisabledForHvpa  = false
		defaultReplicas           *int32
		minReplicas               int32 = 1
		maxReplicas               int32 = 4
	)

	if b.Shoot.Purpose == gardencorev1beta1.ShootPurposeProduction {
		minReplicas = 2
	}

	if metav1.HasAnnotation(b.Shoot.Info.ObjectMeta, v1beta1constants.ShootAlphaControlPlaneScaleDownDisabled) {
		minReplicas = 4
		scaleDownDisabledForHvpa = true
	}

	if b.ManagedSeed != nil {
		hvpaEnabled = gardenletfeatures.FeatureGate.Enabled(features.HVPAForShootedSeed)
		useMemoryMetricForHvpaHPA = true

		if b.ManagedSeedAPIServer != nil {
			minReplicas = *b.ManagedSeedAPIServer.Autoscaler.MinReplicas
			maxReplicas = b.ManagedSeedAPIServer.Autoscaler.MaxReplicas

			if !hvpaEnabled {
				defaultReplicas = b.ManagedSeedAPIServer.Replicas
			}
		}
	}

	return kubeapiserver.AutoscalingConfig{
		HVPAEnabled:               hvpaEnabled,
		Replicas:                  defaultReplicas,
		MinReplicas:               minReplicas,
		MaxReplicas:               maxReplicas,
		UseMemoryMetricForHvpaHPA: useMemoryMetricForHvpaHPA,
		ScaleDownDisabledForHvpa:  scaleDownDisabledForHvpa,
	}
}

func (b *Botanist) computeKubeAPIServerReplicas(deployment *appsv1.Deployment) *int32 {
	autoscalingConfig := b.Shoot.Components.ControlPlane.KubeAPIServer.GetValues().Autoscaling

	switch {
	case autoscalingConfig.Replicas != nil:
		// If the replicas were already set then don't change them.
		return autoscalingConfig.Replicas
	case deployment == nil:
		// If the Deployment does not yet exist then set the desired replicas to the minimum replicas.
		return &autoscalingConfig.MinReplicas
	case deployment.Spec.Replicas != nil && *deployment.Spec.Replicas > 0:
		// If the Deployment exists then don't interfere with the replicas because they are controlled via HVPA or HPA.
		return deployment.Spec.Replicas
	case b.Shoot.HibernationEnabled && (deployment.Spec.Replicas == nil || *deployment.Spec.Replicas == 0):
		// If the Shoot is hibernated and the deployment has already been scaled down then we want to keep it scaled
		// down. If it has not yet been scaled down then above case applies (replicas are kept) - the scale-down will
		// happen at a later point in the flow.
		return pointer.Int32(0)
	default:
		// If none of the above cases applies then a default value has to be returned.
		return pointer.Int32(1)
	}
}

// GetLegacyDeployKubeAPIServerFunc is exposed for testing.
// TODO: Remove this once the kube-apiserver component refactoring has been completed.
var GetLegacyDeployKubeAPIServerFunc = func(botanist *Botanist) func(context.Context) error {
	return botanist.deployKubeAPIServer
}

// DeployKubeAPIServer deploys the Kubernetes API server.
func (b *Botanist) DeployKubeAPIServer(ctx context.Context) error {
	if err := GetLegacyDeployKubeAPIServerFunc(b)(ctx); err != nil {
		return err
	}

	deployment := &appsv1.Deployment{}
	if err := b.K8sSeedClient.Client().Get(ctx, kutil.Key(b.Shoot.SeedNamespace, v1beta1constants.DeploymentNameKubeAPIServer), deployment); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		deployment = nil
	}

	b.Shoot.Components.ControlPlane.KubeAPIServer.SetAutoscalingReplicas(b.computeKubeAPIServerReplicas(deployment))

	return b.Shoot.Components.ControlPlane.KubeAPIServer.Deploy(ctx)
}

// DeleteKubeAPIServer deletes the kube-apiserver deployment in the Seed cluster which holds the Shoot's control plane.
func (b *Botanist) DeleteKubeAPIServer(ctx context.Context) error {
	// invalidate shoot client here before deleting API server
	if err := b.ClientMap.InvalidateClient(keys.ForShoot(b.Shoot.Info)); err != nil {
		return err
	}
	b.K8sShootClient = nil

	return b.Shoot.Components.ControlPlane.KubeAPIServer.Destroy(ctx)
}

// WakeUpKubeAPIServer creates a service and ensures API Server is scaled up
func (b *Botanist) WakeUpKubeAPIServer(ctx context.Context) error {
	sniPhase := b.Shoot.Components.ControlPlane.KubeAPIServerSNIPhase.Done()

	if err := b.DeployKubeAPIService(ctx, sniPhase); err != nil {
		return err
	}
	if err := b.Shoot.Components.ControlPlane.KubeAPIServerService.Wait(ctx); err != nil {
		return err
	}
	if b.APIServerSNIEnabled() {
		if err := b.DeployKubeAPIServerSNI(ctx); err != nil {
			return err
		}
	}
	if err := b.DeployKubeAPIServer(ctx); err != nil {
		return err
	}
	if err := kubernetes.ScaleDeployment(ctx, b.K8sSeedClient.Client(), kutil.Key(b.Shoot.SeedNamespace, v1beta1constants.DeploymentNameKubeAPIServer), 1); err != nil {
		return err
	}
	if err := b.Shoot.Components.ControlPlane.KubeAPIServer.Wait(ctx); err != nil {
		return err
	}

	return nil
}

// ScaleKubeAPIServerToOne scales kube-apiserver replicas to one.
func (b *Botanist) ScaleKubeAPIServerToOne(ctx context.Context) error {
	return kubernetes.ScaleDeployment(ctx, b.K8sSeedClient.Client(), kutil.Key(b.Shoot.SeedNamespace, v1beta1constants.DeploymentNameKubeAPIServer), 1)
}
