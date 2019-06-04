// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package main

import (
	"context"
	"fmt"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/operation/common"
	"github.com/gardener/gardener/test/integration/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// updateWorkerZone updates the zone of the workers.
// Azure shoots are ignored.
func updateWorkerZone(shoot *gardenv1beta1.Shoot, cloudprovider gardenv1beta1.CloudProvider, zone string) {
	switch cloudprovider {
	case gardenv1beta1.CloudProviderAWS:
		shoot.Spec.Cloud.AWS.Zones = []string{zone}
	case gardenv1beta1.CloudProviderGCP:
		shoot.Spec.Cloud.GCP.Zones = []string{zone}
	case gardenv1beta1.CloudProviderAzure:
		return
	case gardenv1beta1.CloudProviderOpenStack:
		shoot.Spec.Cloud.OpenStack.Zones = []string{zone}
	case gardenv1beta1.CloudProviderAlicloud:
		shoot.Spec.Cloud.Alicloud.Zones = []string{zone}
	default:
		testLogger.Warnf("unsupported cloudprovider %s", cloudprovider)
	}
}

// updateMachineType updates the machine type of a shoot if a machinetype is provided.
func updateMachineType(shoot *gardenv1beta1.Shoot, cloudprovider gardenv1beta1.CloudProvider, machinetype string) {
	if machinetype != "" {
		switch cloudprovider {
		case gardenv1beta1.CloudProviderAWS:
			shoot.Spec.Cloud.AWS.Workers[0].MachineType = machinetype
		case gardenv1beta1.CloudProviderGCP:
			shoot.Spec.Cloud.GCP.Workers[0].MachineType = machinetype
		case gardenv1beta1.CloudProviderAzure:
			shoot.Spec.Cloud.Azure.Workers[0].MachineType = machinetype
		case gardenv1beta1.CloudProviderOpenStack:
			shoot.Spec.Cloud.OpenStack.Workers[0].MachineType = machinetype
		case gardenv1beta1.CloudProviderAlicloud:
			shoot.Spec.Cloud.Alicloud.Workers[0].MachineType = machinetype
		default:
			testLogger.Warnf("unsupported cloudprovider %s", cloudprovider)
		}
	}
}

// updateAutoscalerMinMax updates autoscalerMin and autoscalerMax of the worker if they are defined.
func updateAutoscalerMinMax(shoot *gardenv1beta1.Shoot, cloudprovider gardenv1beta1.CloudProvider, min, max *int) {
	if min != nil {
		switch cloudprovider {
		case gardenv1beta1.CloudProviderAWS:
			shoot.Spec.Cloud.AWS.Workers[0].AutoScalerMin = *min
		case gardenv1beta1.CloudProviderGCP:
			shoot.Spec.Cloud.GCP.Workers[0].AutoScalerMin = *min
		case gardenv1beta1.CloudProviderAzure:
			shoot.Spec.Cloud.Azure.Workers[0].AutoScalerMin = *min
		case gardenv1beta1.CloudProviderOpenStack:
			shoot.Spec.Cloud.OpenStack.Workers[0].AutoScalerMin = *min
		case gardenv1beta1.CloudProviderAlicloud:
			shoot.Spec.Cloud.Alicloud.Workers[0].AutoScalerMin = *min
		default:
			testLogger.Warnf("unsupported cloudprovider %s", cloudprovider)
		}
	}
	if max != nil {
		switch cloudprovider {
		case gardenv1beta1.CloudProviderAWS:
			shoot.Spec.Cloud.AWS.Workers[0].AutoScalerMax = *max
		case gardenv1beta1.CloudProviderGCP:
			shoot.Spec.Cloud.GCP.Workers[0].AutoScalerMax = *max
		case gardenv1beta1.CloudProviderAzure:
			shoot.Spec.Cloud.Azure.Workers[0].AutoScalerMax = *max
		case gardenv1beta1.CloudProviderOpenStack:
			shoot.Spec.Cloud.OpenStack.Workers[0].AutoScalerMax = *max
		case gardenv1beta1.CloudProviderAlicloud:
			shoot.Spec.Cloud.Alicloud.Workers[0].AutoScalerMax = *max
		default:
			testLogger.Warnf("unsupported cloudprovider %s", cloudprovider)
		}
	}
}

// updateFloatingPoolName updates the floatingPoolName if an openstack cluster is created.
func updateFloatingPoolName(shoot *gardenv1beta1.Shoot, floatingPoolName string, cloudprovider gardenv1beta1.CloudProvider) {
	if cloudprovider == gardenv1beta1.CloudProviderOpenStack {
		shoot.Spec.Cloud.OpenStack.FloatingPoolName = floatingPoolName
	}
}

// updateLoadBalancerProvider updates the loadBalancerProvider if an openstack cluster is created.
func updateLoadBalancerProvider(shoot *gardenv1beta1.Shoot, loadBalancerProvider string, cloudprovider gardenv1beta1.CloudProvider) {
	if cloudprovider == gardenv1beta1.CloudProviderOpenStack && loadBalancerProvider != "" {
		shoot.Spec.Cloud.OpenStack.LoadBalancerProvider = loadBalancerProvider
	}
}

// updateAnnotations adds default annotations that should be present on any shoot created.
func updateAnnotations(shoot *gardenv1beta1.Shoot) {
	if shoot.Annotations == nil {
		shoot.Annotations = map[string]string{}
	}
	shoot.Annotations[common.GardenIgnoreAlerts] = "true"
}

// updateBackupInfrastructureAnnotations adds default annotations that should be present on any backupinfrastructure created.
func updateBackupInfrastructureAnnotations(backup *gardenv1beta1.BackupInfrastructure) {
	if backup.Annotations == nil {
		backup.Annotations = map[string]string{}
	}
	backup.Annotations[common.BackupInfrastructureForceDeletion] = "true"
}

// getBackupInfrastructureOfShoot returns the BackupInfrastructure object of the shoot
func getBackupInfrastructureOfShoot(ctx  context.Context, shootGardenerTest *framework.ShootGardenerTest, shootObject *gardenv1beta1.Shoot) (*gardenv1beta1.BackupInfrastructure, error) {
	backups := &gardenv1beta1.BackupInfrastructureList{}
	err := shootGardenerTest.GardenClient.Client().List(ctx, &client.ListOptions{Namespace: shootObject.Namespace}, backups)
	if err != nil {
		return nil, err
	}
	for _, backup := range backups.Items {
		if backup.Spec.ShootUID == shootObject.GetUID() {
			return &backup, nil
		}
	}
	return nil, fmt.Errorf("cannot find backup infrastructure for shoot with uid %s", shootObject.GetUID())
}