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
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/controllermanager/apis/config"
	"github.com/gardener/gardener/pkg/controllerutils"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ControllerName is the name of this controller.
const ControllerName = "shoot"

// Controller controls Shoots.
type Controller struct {
	config *config.ControllerManagerConfiguration
	log    logr.Logger

	shootQuotaReconciler       reconcile.Reconciler
	shootRefReconciler         reconcile.Reconciler
	shootRetryReconciler       reconcile.Reconciler
	shootStatusLabelReconciler reconcile.Reconciler
	hasSyncedFuncs             []cache.InformerSynced

	shootQuotaQueue        workqueue.RateLimitingInterface
	shootReferenceQueue    workqueue.RateLimitingInterface
	shootRetryQueue        workqueue.RateLimitingInterface
	shootStatusLabelQueue  workqueue.RateLimitingInterface
	numberOfRunningWorkers int
	workerCh               chan int
}

// NewShootController takes a ClientMap, a GardenerInformerFactory, a KubernetesInformerFactory, a
// ControllerManagerConfig struct and an EventRecorder to create a new Shoot controller.
func NewShootController(
	ctx context.Context,
	log logr.Logger,
	mgr manager.Manager,
	config *config.ControllerManagerConfiguration,
) (
	*Controller,
	error,
) {
	log = log.WithName(ControllerName)

	gardenClient := mgr.GetClient()
	gardenCache := mgr.GetCache()

	shootInformer, err := gardenCache.GetInformer(ctx, &gardencorev1beta1.Shoot{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Shoot Informer: %w", err)
	}

	shootController := &Controller{
		config: config,
		log:    log,

		shootQuotaReconciler:       NewShootQuotaReconciler(gardenClient, config.Controllers.ShootQuota),
		shootRetryReconciler:       NewShootRetryReconciler(gardenClient, config.Controllers.ShootRetry),
		shootStatusLabelReconciler: NewShootStatusLabelReconciler(gardenClient),
		shootRefReconciler:         NewShootReferenceReconciler(gardenClient),

		shootQuotaQueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "shoot-quota"),
		shootReferenceQueue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "shoot-references"),
		shootRetryQueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "shoot-retry"),
		shootStatusLabelQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "shoot-status-label"),

		workerCh: make(chan int),
	}

	shootInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    shootController.shootQuotaAdd,
		DeleteFunc: shootController.shootQuotaDelete,
	})

	shootInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    shootController.shootReferenceAdd,
		UpdateFunc: shootController.shootReferenceUpdate,
	})

	shootInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    shootController.shootRetryAdd,
		UpdateFunc: shootController.shootRetryUpdate,
	})

	shootInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    shootController.shootStatusLabelAdd,
		UpdateFunc: shootController.shootStatusLabelUpdate,
	})

	shootController.hasSyncedFuncs = []cache.InformerSynced{
		shootInformer.HasSynced,
	}

	return shootController, nil
}

// Run runs the Controller until the given stop channel can be read from.
func (c *Controller) Run(
	ctx context.Context,
	shootQuotaWorkers, shootReferenceWorkers, shootRetryWorkers, shootStatusLabelWorkers int,
) {
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

	c.log.Info("Shoot controller initialized")

	for i := 0; i < shootQuotaWorkers; i++ {
		controllerutils.CreateWorker(ctx, c.shootQuotaQueue, "Shoot Quota", c.shootQuotaReconciler, &waitGroup, c.workerCh, controllerutils.WithLogger(c.log.WithName(quotaReconcilerName)))
	}
	for i := 0; i < shootReferenceWorkers; i++ {
		controllerutils.CreateWorker(ctx, c.shootReferenceQueue, "Shoot Reference", c.shootRefReconciler, &waitGroup, c.workerCh, controllerutils.WithLogger(c.log.WithName(referenceReconcilerName)))
	}
	for i := 0; i < shootRetryWorkers; i++ {
		controllerutils.CreateWorker(ctx, c.shootRetryQueue, "Shoot Retry", c.shootRetryReconciler, &waitGroup, c.workerCh, controllerutils.WithLogger(c.log.WithName(retryReconcilerName)))
	}
	for i := 0; i < shootStatusLabelWorkers; i++ {
		controllerutils.CreateWorker(ctx, c.shootStatusLabelQueue, "Shoot Status Label", c.shootStatusLabelReconciler, &waitGroup, c.workerCh, controllerutils.WithLogger(c.log.WithName(statusLabelReconcilerName)))
	}

	// Shutdown handling
	<-ctx.Done()
	c.shootQuotaQueue.ShutDown()
	c.shootReferenceQueue.ShutDown()
	c.shootRetryQueue.ShutDown()
	c.shootStatusLabelQueue.ShutDown()

	for {
		var (
			shootQuotaQueueLength       = c.shootQuotaQueue.Len()
			referenceQueueLength        = c.shootReferenceQueue.Len()
			shootRetryQueueLength       = c.shootRetryQueue.Len()
			shootStatusLabelQueueLength = c.shootStatusLabelQueue.Len()
			queueLengths                = shootQuotaQueueLength + referenceQueueLength + shootRetryQueueLength + shootStatusLabelQueueLength
		)
		if queueLengths == 0 && c.numberOfRunningWorkers == 0 {
			c.log.V(1).Info("No running Shoot worker and no items left in the queues. Terminating Shoot controller")
			break
		}

		c.log.V(1).Info("Waiting for Shoot workers to finish", "numberOfRunningWorkers", c.numberOfRunningWorkers, "queueLength", queueLengths)
		time.Sleep(5 * time.Second)
	}

	waitGroup.Wait()
}
