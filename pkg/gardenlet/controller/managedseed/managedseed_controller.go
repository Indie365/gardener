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

package managedseed

import (
	"k8s.io/client-go/tools/cache"

	seedmanagementv1alpha1 "github.com/gardener/gardener/pkg/apis/seedmanagement/v1alpha1"
	"github.com/gardener/gardener/pkg/utils"
)

func (c *Controller) managedSeedAdd(obj interface{}) {
	managedSeed, ok := obj.(*seedmanagementv1alpha1.ManagedSeed)
	if !ok {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}

	generationChanged := managedSeed.Generation != managedSeed.Status.ObservedGeneration
	jitterUpdates := (c.config.Controllers.ManagedSeed.JitterUpdates != nil) && *(c.config.Controllers.ManagedSeed.JitterUpdates)

	// Managed seed with deletion timestamp and newly created managed seed will be enqueued immediately.
	// Generation is 1 for newly created objects.
	if managedSeed.DeletionTimestamp != nil || managedSeed.Generation == 1 {
		c.logger.Debugf("Added ManagedSeed %s without delay to the queue", key)
		c.managedSeedQueue.Add(key)
		return
	}

	if generationChanged {
		if jitterUpdates {
			enqueueWithJitterDelay(c, key)
		} else {
			c.logger.Debugf("Added ManagedSeed %s without delay to the queue", key)
			c.managedSeedQueue.Add(key)
		}
	} else {
		// Spread reconciliation of managed seeds (including gardenlet updates/rollouts) across the configured sync jitter
		// period to avoid overloading the gardener-apiserver if all gardenlets in all managed seeds are (re)starting
		// roughly at the same time
		enqueueWithJitterDelay(c, key)
	}
}

func (c *Controller) managedSeedUpdate(_, newObj interface{}) {
	managedSeed, ok := newObj.(*seedmanagementv1alpha1.ManagedSeed)
	if !ok {
		return
	}

	if managedSeed.Generation == managedSeed.Status.ObservedGeneration {
		return
	}

	c.managedSeedAdd(newObj)
}

func (c *Controller) managedSeedDelete(obj interface{}) {
	if _, ok := obj.(*seedmanagementv1alpha1.ManagedSeed); !ok {
		if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); !ok {
			return
		} else if _, ok := tombstone.Obj.(*seedmanagementv1alpha1.ManagedSeed); !ok {
			return
		}
	}
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}

	c.managedSeedQueue.Add(key)
}

func enqueueWithJitterDelay(c *Controller, key string) {
	duration := utils.RandomDurationWithMetaDuration(c.config.Controllers.ManagedSeed.SyncJitterPeriod)
	c.logger.Infof("Added ManagedSeed %s with delay %s to the queue", key, duration)
	c.managedSeedQueue.AddAfter(key, duration)
}
