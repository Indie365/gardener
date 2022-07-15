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

package quota

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/controllerutils"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ControllerName is the name of this controller.
const ControllerName = "quota"

// Controller controls Quotas.
type Controller struct {
	reconciler     reconcile.Reconciler
	hasSyncedFuncs []cache.InformerSynced

	log                    logr.Logger
	quotaQueue             workqueue.RateLimitingInterface
	workerCh               chan int
	numberOfRunningWorkers int
}

// NewQuotaController takes a Kubernetes client for the Garden clusters <k8sGardenClient>, a struct
// holding information about the acting Gardener, a <quotaInformer>, and a <recorder> for
// event recording. It creates a new Gardener controller.
func NewQuotaController(
	ctx context.Context,
	log logr.Logger,
	mgr manager.Manager,
) (
	*Controller,
	error,
) {
	log = log.WithName(ControllerName)

	quotaInformer, err := mgr.GetCache().GetInformer(ctx, &gardencorev1beta1.Quota{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Quota Informer: %w", err)
	}

	quotaController := &Controller{
		reconciler: NewQuotaReconciler(mgr.GetClient(), mgr.GetEventRecorderFor(ControllerName+"-controller")),
		log:        log,
		quotaQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Quota"),
		workerCh:   make(chan int),
	}

	quotaInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    quotaController.quotaAdd,
		UpdateFunc: quotaController.quotaUpdate,
		DeleteFunc: quotaController.quotaDelete,
	})

	quotaController.hasSyncedFuncs = append(quotaController.hasSyncedFuncs, quotaInformer.HasSynced)

	return quotaController, nil
}

// Run runs the Controller until the given stop channel can be read from.
func (c *Controller) Run(ctx context.Context, workers int) {
	var waitGroup sync.WaitGroup

	if !cache.WaitForCacheSync(ctx.Done(), c.hasSyncedFuncs...) {
		c.log.Error(wait.ErrWaitTimeout, "Timed out waiting for caches to sync")
		return
	}

	// Count number of running workers.
	go func() {
		for res := range c.workerCh {
			c.numberOfRunningWorkers += res
		}
	}()

	c.log.Info("Quota controller initialized")

	for i := 0; i < workers; i++ {
		controllerutils.CreateWorker(ctx, c.quotaQueue, "Quota", c.reconciler, &waitGroup, c.workerCh, controllerutils.WithLogger(c.log))
	}

	// Shutdown handling
	<-ctx.Done()
	c.quotaQueue.ShutDown()

	for {
		if c.quotaQueue.Len() == 0 && c.numberOfRunningWorkers == 0 {
			c.log.V(1).Info("No running Quota worker and no items left in the queues. Terminating Quota controller")
			break
		}
		c.log.V(1).Info("Waiting for Quota workers to finish", "numberOfRunningWorkers", c.numberOfRunningWorkers, "queueLength", c.quotaQueue.Len())
		time.Sleep(5 * time.Second)
	}

	waitGroup.Wait()
}
