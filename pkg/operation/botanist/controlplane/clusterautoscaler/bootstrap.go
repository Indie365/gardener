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

package clusterautoscaler

import (
	"context"
	"time"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/operation/common"
	"github.com/gardener/gardener/pkg/utils/managedresources"

	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterRoleControlName     = "system:cluster-autoscaler-seed"
	managedResourceControlName = "cluster-autoscaler"
)

// TimeoutWaitForManagedResource is the timeout used while waiting for the ManagedResources to become healthy
// or deleted.
var TimeoutWaitForManagedResource = 2 * time.Minute

// BootstrapSeed deploys the RBAC configuration for the control cluster.
func BootstrapSeed(ctx context.Context, c client.Client, namespace, _ string) error {
	var (
		versions = schema.GroupVersions([]schema.GroupVersion{rbacv1.SchemeGroupVersion})
		codec    = kubernetes.SeedCodec.CodecForVersions(kubernetes.SeedSerializer, kubernetes.SeedSerializer, versions, versions)

		clusterRole = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterRoleControlName,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{machinev1alpha1.GroupName},
					Resources: []string{"*"},
					Verbs:     []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"},
				},
			},
		}
		clusterRoleYAML, _ = runtime.Encode(codec, clusterRole)
	)

	if err := common.DeployManagedResourceForSeed(ctx, c, managedResourceControlName, namespace, false, map[string][]byte{"clusterrole.yaml": clusterRoleYAML}); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, TimeoutWaitForManagedResource)
	defer cancel()

	return managedresources.WaitUntilManagedResourceHealthy(timeoutCtx, c, namespace, managedResourceControlName)
}

// DebootstrapSeed deletes all the resources deployed during the seed bootstrapping.
func DebootstrapSeed(ctx context.Context, c client.Client, namespace string) error {
	if err := common.DeleteManagedResourceForSeed(ctx, c, managedResourceControlName, namespace); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, TimeoutWaitForManagedResource)
	defer cancel()

	return managedresources.WaitUntilManagedResourceDeleted(timeoutCtx, c, namespace, managedResourceControlName)
}
