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

package controllerinstallation

import (
	"context"
	"sync"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/gardenlet/apis/config"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

const (
	// ControllerName is the name of this controller.
	ControllerName = "controllerinstallation"
)

// Controller controls ControllerInstallation.
type Controller struct {
	log logr.Logger

	hasSyncedFuncs         []cache.InformerSynced
	workerCh               chan int
	numberOfRunningWorkers int
}

// NewController instantiates a new ControllerInstallation controller.
func NewController(
	ctx context.Context,
	log logr.Logger,
	gardenCluster cluster.Cluster,
	seedClientSet kubernetes.Interface,
	config *config.GardenletConfiguration,
	identity *gardencorev1beta1.Gardener,
	gardenNamespace *corev1.Namespace,
	gardenClusterIdentity string,
) (
	*Controller,
	error,
) {
	log = log.WithName(ControllerName)

	controller := &Controller{
		log: log,

		workerCh: make(chan int),
	}

	controller.hasSyncedFuncs = []cache.InformerSynced{}

	return controller, nil
}

// Run runs the Controller until the given stop channel can be read from.
func (c *Controller) Run(ctx context.Context) {
	var waitGroup sync.WaitGroup

	if !cache.WaitForCacheSync(ctx.Done(), c.hasSyncedFuncs...) {
		c.log.Error(wait.ErrWaitTimeout, "Timed out waiting for caches to sync")
		return
	}

	go func() {
		for res := range c.workerCh {
			c.numberOfRunningWorkers += res
		}
	}()

	c.log.Info("ControllerInstallation controller initialized")

	// Shutdown handling
	<-ctx.Done()

	for {
		if c.numberOfRunningWorkers == 0 {
			c.log.V(1).Info("No running ControllerInstallation worker and no items left in the queues. Terminated ControllerInstallation controller")

			break
		}
		c.log.V(1).Info("Waiting for ControllerInstallation workers to finish", "numberOfRunningWorkers", c.numberOfRunningWorkers)
		time.Sleep(5 * time.Second)
	}

	waitGroup.Wait()
}
