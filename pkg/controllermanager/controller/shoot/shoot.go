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

package shoot

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/pkg/controllermanager/apis/config"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManager adds new shoot controllers to the given manager.
func AddToManager(
	ctx context.Context,
	mgr manager.Manager,
	config *config.ControllerManagerControllerConfiguration,
) error {
	if err := addShootHibernationController(ctx, mgr, &config.ShootHibernation); err != nil {
		return fmt.Errorf("failed to add shoot-hibernation controller: %w", err)
	}

	if err := addShootMaintenanceController(ctx, mgr, &config.ShootMaintenance); err != nil {
		return fmt.Errorf("failed to add shoot-maintenance controller: %w", err)
	}

	if err := addShootQuotaController(ctx, mgr, &config.ShootQuota); err != nil {
		return fmt.Errorf("failed to add shoot-quota controller: %w", err)
	}

	if err := addShootReferenceController(ctx, mgr, config.ShootReference); err != nil {
		return fmt.Errorf("failed to add shoot-reference controller: %w", err)
	}

	if err := addShootRetryController(ctx, mgr, config.ShootRetry); err != nil {
		return fmt.Errorf("failed to add shoot-retry controller: %w", err)
	}

	if err := addConfigMapController(ctx, mgr, &config.ShootMaintenance); err != nil {
		return fmt.Errorf("failed to add shoot-configmap controller: %w", err)
	}

	return nil
}
