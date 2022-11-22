// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package backupentry

import (
	"fmt"

	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener/pkg/features"
	"github.com/gardener/gardener/pkg/gardenlet/apis/config"
	confighelper "github.com/gardener/gardener/pkg/gardenlet/apis/config/helper"
	"github.com/gardener/gardener/pkg/gardenlet/controller/backupentry/backupentry"
	"github.com/gardener/gardener/pkg/gardenlet/controller/backupentry/migration"
	gardenletfeatures "github.com/gardener/gardener/pkg/gardenlet/features"
)

// AddToManager adds all ControllerInstallation controllers to the given manager.
func AddToManager(
	mgr manager.Manager,
	gardenCluster cluster.Cluster,
	seedCluster cluster.Cluster,
	cfg config.GardenletConfiguration,
) error {
	if err := (&backupentry.Reconciler{
		Clock:    clock.RealClock{},
		Config:   *cfg.Controllers.BackupEntry,
		SeedName: cfg.SeedConfig.Name,
	}).AddToManager(mgr, gardenCluster, seedCluster); err != nil {
		return fmt.Errorf("failed adding main reconciler: %w", err)
	}

	if gardenletfeatures.FeatureGate.Enabled(features.ForceRestore) && confighelper.OwnerChecksEnabledInSeedConfig(cfg.SeedConfig) {
		if err := (&migration.Reconciler{
			Clock:  clock.RealClock{},
			Config: cfg,
		}).AddToManager(mgr, gardenCluster); err != nil {
			return fmt.Errorf("failed adding main reconciler: %w", err)
		}
	}

	return nil
}
