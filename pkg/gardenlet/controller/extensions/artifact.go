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

package extensions

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	runtimecache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

// controllerArtifacts bundles a list of artifacts for extension kinds
// which are required for state and ControllerInstallation processing.
type controllerArtifacts struct {
	controllerInstallationArtifacts map[string]*artifact
	hasSyncedFuncs                  []cache.InformerSynced
	shutDownFuncs                   []func()
}

type predicateFn func(newObj, oldObj interface{}) bool

// artifact is specified for extension kinds.
// It servers as a helper to setup the corresponding reconciliation function.
type artifact struct {
	initialized bool

	gvk               schema.GroupVersionKind
	newObjFunc        func() client.Object
	newListFunc       func() client.ObjectList
	informer          runtimecache.Informer
	queue             workqueue.RateLimitingInterface
	predicate         predicateFn
	addEventHandlerFn func()
}

// newControllerArtifacts creates a new controllerArtifacts instance with the necessary artifacts
// for state and ControllerInstallation processing.
func newControllerArtifacts() controllerArtifacts {
	a := controllerArtifacts{
		controllerInstallationArtifacts: make(map[string]*artifact),
	}

	gvk := extensionsv1alpha1.SchemeGroupVersion.WithKind(extensionsv1alpha1.BackupBucketResource)
	a.registerExtensionControllerArtifacts(
		newControllerInstallationArtifact(gvk, func() client.ObjectList { return &extensionsv1alpha1.BackupBucketList{} }, extensionTypeChanged),
	)

	gvk = extensionsv1alpha1.SchemeGroupVersion.WithKind(extensionsv1alpha1.BackupEntryResource)
	a.registerExtensionControllerArtifacts(
		newControllerInstallationArtifact(gvk, func() client.ObjectList { return &extensionsv1alpha1.BackupEntryList{} }, extensionTypeChanged),
	)

	gvk = extensionsv1alpha1.SchemeGroupVersion.WithKind(extensionsv1alpha1.BastionResource)
	a.registerExtensionControllerArtifacts(
		newControllerInstallationArtifact(gvk, func() client.ObjectList { return &extensionsv1alpha1.BastionList{} }, extensionTypeChanged),
	)

	gvk = extensionsv1alpha1.SchemeGroupVersion.WithKind(extensionsv1alpha1.ContainerRuntimeResource)
	a.registerExtensionControllerArtifacts(
		newControllerInstallationArtifact(gvk, func() client.ObjectList { return &extensionsv1alpha1.ContainerRuntimeList{} }, extensionTypeChanged),
	)

	gvk = extensionsv1alpha1.SchemeGroupVersion.WithKind(extensionsv1alpha1.ControlPlaneResource)
	a.registerExtensionControllerArtifacts(newControllerInstallationArtifact(gvk, func() client.ObjectList { return &extensionsv1alpha1.ControlPlaneList{} }, extensionTypeChanged))

	gvk = extensionsv1alpha1.SchemeGroupVersion.WithKind(extensionsv1alpha1.ExtensionResource)
	a.registerExtensionControllerArtifacts(
		newControllerInstallationArtifact(gvk, func() client.ObjectList { return &extensionsv1alpha1.ExtensionList{} }, extensionTypeChanged),
	)

	gvk = extensionsv1alpha1.SchemeGroupVersion.WithKind(extensionsv1alpha1.InfrastructureResource)
	a.registerExtensionControllerArtifacts(
		newControllerInstallationArtifact(gvk, func() client.ObjectList { return &extensionsv1alpha1.InfrastructureList{} }, extensionTypeChanged),
	)

	gvk = extensionsv1alpha1.SchemeGroupVersion.WithKind(extensionsv1alpha1.NetworkResource)
	a.registerExtensionControllerArtifacts(
		newControllerInstallationArtifact(gvk, func() client.ObjectList { return &extensionsv1alpha1.NetworkList{} }, extensionTypeChanged),
	)

	gvk = extensionsv1alpha1.SchemeGroupVersion.WithKind(extensionsv1alpha1.OperatingSystemConfigResource)
	a.registerExtensionControllerArtifacts(
		newControllerInstallationArtifact(gvk, func() client.ObjectList { return &extensionsv1alpha1.OperatingSystemConfigList{} }, extensionTypeChanged),
	)

	gvk = extensionsv1alpha1.SchemeGroupVersion.WithKind(extensionsv1alpha1.WorkerResource)
	a.registerExtensionControllerArtifacts(
		newControllerInstallationArtifact(gvk, func() client.ObjectList { return &extensionsv1alpha1.WorkerList{} }, extensionTypeChanged),
	)

	gvk = extensionsv1alpha1.SchemeGroupVersion.WithKind(extensionsv1alpha1.DNSRecordResource)
	a.registerExtensionControllerArtifacts(
		newControllerInstallationArtifact(gvk, func() client.ObjectList { return &extensionsv1alpha1.DNSRecordList{} }, extensionTypeChanged),
	)

	return a
}

func (c *controllerArtifacts) registerExtensionControllerArtifacts(controllerInstallation *artifact) {
	if controllerInstallation != nil {
		c.controllerInstallationArtifacts[controllerInstallation.gvk.Kind] = controllerInstallation
	}
}

// initialize obtains the informers for the enclosing artifacts.
func (c *controllerArtifacts) initialize(ctx context.Context, seedCluster cluster.Cluster) error {
	initialize := func(a *artifact) error {
		if a.initialized {
			return nil
		}
		informer, err := seedCluster.GetCache().GetInformerForKind(ctx, a.gvk)
		if err != nil {
			return err
		}
		a.informer = informer
		c.hasSyncedFuncs = append(c.hasSyncedFuncs, informer.HasSynced)
		c.shutDownFuncs = append(c.shutDownFuncs, a.queue.ShutDown)
		a.addEventHandlerFn()
		a.initialized = true
		return nil
	}

	for _, artifact := range c.controllerInstallationArtifacts {
		if err := initialize(artifact); err != nil {
			return err
		}
	}

	return nil
}

func (c *controllerArtifacts) shutdownQueues() {
	for _, shutdown := range c.shutDownFuncs {
		shutdown()
	}
}

func newControllerInstallationArtifact(gvk schema.GroupVersionKind, newObjFunc func() client.ObjectList, fn predicateFn) *artifact {
	a := &artifact{
		gvk:         gvk,
		newListFunc: newObjFunc,
		queue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), fmt.Sprintf("controllerinstallation-extension-%s", gvk.Kind)),
		predicate:   fn,
	}

	a.addEventHandlerFn = func() {
		a.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    createEnqueueEmptyRequestFunc(a.queue),
			UpdateFunc: createEnqueueEmptyRequestOnUpdateFunc(a.queue, a.predicate),
			DeleteFunc: createEnqueueEmptyRequestFunc(a.queue),
		})
	}

	return a
}
