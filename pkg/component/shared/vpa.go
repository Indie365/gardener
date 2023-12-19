// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package shared

import (
	"time"

	"github.com/Masterminds/semver/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener/imagevector"
	"github.com/gardener/gardener/pkg/component"
	"github.com/gardener/gardener/pkg/component/vpa"
	imagevectorutils "github.com/gardener/gardener/pkg/utils/imagevector"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
)

// NewVerticalPodAutoscaler instantiates a new `vertical-pod-autoscaler` component.
func NewVerticalPodAutoscaler(
	c client.Client,
	gardenNamespaceName string,
	runtimeVersion *semver.Version,
	secretsManager secretsmanager.Interface,
	enabled bool,
	secretNameServerCA string,
	priorityClassNameAdmissionController string,
	priorityClassNameRecommender string,
	priorityClassNameUpdater string,
) (
	component.DeployWaiter,
	error,
) {
	imageAdmissionController, err := imagevector.ImageVector().FindImage(imagevector.ImageNameVpaAdmissionController, imagevectorutils.TargetVersion(runtimeVersion.String()))
	if err != nil {
		return nil, err
	}

	imageRecommender, err := imagevector.ImageVector().FindImage(imagevector.ImageNameVpaRecommender, imagevectorutils.TargetVersion(runtimeVersion.String()))
	if err != nil {
		return nil, err
	}

	imageUpdater, err := imagevector.ImageVector().FindImage(imagevector.ImageNameVpaUpdater, imagevectorutils.TargetVersion(runtimeVersion.String()))
	if err != nil {
		return nil, err
	}

	return vpa.New(
		c,
		gardenNamespaceName,
		secretsManager,
		vpa.Values{
			ClusterType:              component.ClusterTypeSeed,
			Enabled:                  enabled,
			SecretNameServerCA:       secretNameServerCA,
			RuntimeKubernetesVersion: runtimeVersion,
			AdmissionController: vpa.ValuesAdmissionController{
				Image:             imageAdmissionController.String(),
				PriorityClassName: priorityClassNameAdmissionController,
			},
			Recommender: vpa.ValuesRecommender{
				Image:                        imageRecommender.String(),
				PriorityClassName:            priorityClassNameRecommender,
				RecommendationMarginFraction: pointer.Float64(0.05),
			},
			Updater: vpa.ValuesUpdater{
				EvictionTolerance:      pointer.Float64(1.0),
				EvictAfterOOMThreshold: &metav1.Duration{Duration: 48 * time.Hour},
				Image:                  imageUpdater.String(),
				PriorityClassName:      priorityClassNameUpdater,
			},
		},
	), nil
}
