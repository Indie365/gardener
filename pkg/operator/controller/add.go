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

package controller

import (
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener/charts"
	"github.com/gardener/gardener/pkg/operator/apis/config"
	"github.com/gardener/gardener/pkg/operator/controller/garden"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	"github.com/gardener/gardener/pkg/utils/imagevector"
)

// AddToManager adds all controllers to the given manager.
func AddToManager(mgr manager.Manager, cfg *config.OperatorConfiguration) error {
	imageVector, err := imagevector.ReadGlobalImageVectorWithEnvOverride(filepath.Join(charts.Path, "images.yaml"))
	if err != nil {
		return fmt.Errorf("failed reading image vector override: %w", err)
	}

	var componentImageVectors imagevector.ComponentImageVectors
	if path := os.Getenv(imagevector.ComponentOverrideEnv); path != "" {
		componentImageVectors, err = imagevector.ReadComponentOverwriteFile(path)
		if err != nil {
			return fmt.Errorf("failed reading component-specific image vector override: %w", err)
		}
	}

	identity, err := gutil.DetermineIdentity()
	if err != nil {
		return err
	}

	if err := (&garden.Reconciler{
		Config:                *cfg,
		Identity:              identity,
		ImageVector:           imageVector,
		ComponentImageVectors: componentImageVectors,
	}).AddToManager(mgr); err != nil {
		return fmt.Errorf("failed adding Garden controller: %w", err)
	}

	return nil
}
