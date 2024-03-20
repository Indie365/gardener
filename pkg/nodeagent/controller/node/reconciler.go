// Copyright 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package node

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/gardener/pkg/controllerutils"
	nodeagentv1alpha1 "github.com/gardener/gardener/pkg/nodeagent/apis/config/v1alpha1"
	"github.com/gardener/gardener/pkg/nodeagent/dbus"
)

const annotationRestartSystemdServices = "worker.gardener.cloud/restart-systemd-services"

// Reconciler checks for node annotation changes and restarts the specified systemd services.
type Reconciler struct {
	Client   client.Client
	Recorder record.EventRecorder
	DBus     dbus.DBus
}

// Reconcile checks for node annotation changes and restarts the specified systemd services.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := logf.FromContext(ctx)

	ctx, cancel := controllerutils.GetMainReconciliationContext(ctx, controllerutils.DefaultReconciliationTimeout)
	defer cancel()

	node := &metav1.PartialObjectMetadata{}
	node.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Node"))
	if err := r.Client.Get(ctx, request.NamespacedName, node); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("Object is gone, stop reconciling")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("error retrieving object from store: %w", err)
	}

	services, ok := node.Annotations[annotationRestartSystemdServices]
	if !ok {
		return reconcile.Result{}, nil
	}

	var restartGardenerNodeAgent bool
	for _, serviceName := range strings.Split(services, ",") {
		if !strings.HasSuffix(serviceName, ".service") {
			serviceName = serviceName + ".service"
		}
		// If the gardener-node-agent itself should be restarted, we have to first remove the annotation from the node.
		// Otherwise, the annotation is never removed and it restarts itself indefinitely.
		if serviceName == nodeagentv1alpha1.UnitName {
			restartGardenerNodeAgent = true
			continue
		}
		r.restartService(ctx, log, node, serviceName)
	}

	log.Info("Removing annotation from node", "annotation", annotationRestartSystemdServices)
	patch := client.MergeFrom(node.DeepCopy())
	delete(node.Annotations, annotationRestartSystemdServices)
	if err := r.Client.Patch(ctx, node, patch); err != nil {
		return reconcile.Result{}, err
	}

	if restartGardenerNodeAgent {
		r.restartService(ctx, log, node, nodeagentv1alpha1.UnitName)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) restartService(ctx context.Context, log logr.Logger, node client.Object, serviceName string) {
	log.Info("Restarting systemd service", "serviceName", serviceName)
	if err := r.DBus.Restart(ctx, r.Recorder, node, serviceName); err != nil {
		// We don't return the error here since we don't want to repeatedly try to restart services again and again.
		// In both cases (success or failure), an event will be recorded on the Node so that users can check whether
		// the restart worked.
		log.Error(err, "Failed restarting systemd service", "serviceName", serviceName)
	}
}
