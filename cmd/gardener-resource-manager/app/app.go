// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package app

import (
	"context"
	"fmt"
	goruntime "runtime"
	"sync"

	"github.com/gardener/gardener/pkg/controllerutils/routes"
	gardenerhealthz "github.com/gardener/gardener/pkg/healthz"
	resourcemanagercmd "github.com/gardener/gardener/pkg/resourcemanager/cmd"
	csrapprovercontroller "github.com/gardener/gardener/pkg/resourcemanager/controller/csrapprover"
	garbagecollectorcontroller "github.com/gardener/gardener/pkg/resourcemanager/controller/garbagecollector"
	healthcontroller "github.com/gardener/gardener/pkg/resourcemanager/controller/health"
	resourcecontroller "github.com/gardener/gardener/pkg/resourcemanager/controller/managedresource"
	rootcacontroller "github.com/gardener/gardener/pkg/resourcemanager/controller/rootcapublisher"
	secretcontroller "github.com/gardener/gardener/pkg/resourcemanager/controller/secret"
	tokeninvalidatorcontroller "github.com/gardener/gardener/pkg/resourcemanager/controller/tokeninvalidator"
	tokenrequestorcontroller "github.com/gardener/gardener/pkg/resourcemanager/controller/tokenrequestor"
	resourcemanagerhealthz "github.com/gardener/gardener/pkg/resourcemanager/healthz"
	podschedulernamewebhook "github.com/gardener/gardener/pkg/resourcemanager/webhook/podschedulername"
	"github.com/gardener/gardener/pkg/resourcemanager/webhook/podtopologyspreadconstraints"
	"github.com/gardener/gardener/pkg/resourcemanager/webhook/podzoneaffinity"
	projectedtokenmountwebhook "github.com/gardener/gardener/pkg/resourcemanager/webhook/projectedtokenmount"
	seccompprofilewebhook "github.com/gardener/gardener/pkg/resourcemanager/webhook/seccompprofile"
	tokeninvalidatorwebhook "github.com/gardener/gardener/pkg/resourcemanager/webhook/tokeninvalidator"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/component-base/version"
	"k8s.io/component-base/version/verflag"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var log = logf.Log

// NewResourceManagerCommand creates a new command for running gardener resource manager controllers.
func NewResourceManagerCommand() *cobra.Command {
	var (
		managerOpts       = &resourcemanagercmd.ManagerOptions{}
		profilingOpts     = &resourcemanagercmd.ProfilingOptions{}
		sourceClientOpts  = &resourcemanagercmd.SourceClientOptions{}
		targetClusterOpts = &resourcemanagercmd.TargetClusterOptions{}

		resourceControllerOpts                  = &resourcecontroller.ControllerOptions{}
		secretControllerOpts                    = &secretcontroller.ControllerOptions{}
		healthControllerOpts                    = &healthcontroller.ControllerOptions{}
		gcControllerOpts                        = &garbagecollectorcontroller.ControllerOptions{}
		tokenInvalidatorControllerOpts          = &tokeninvalidatorcontroller.ControllerOptions{}
		tokenRequestorControllerOpts            = &tokenrequestorcontroller.ControllerOptions{}
		rootCAControllerOpts                    = &rootcacontroller.ControllerOptions{}
		csrApproverControllerOpts               = &csrapprovercontroller.ControllerOptions{}
		projectedTokenMountWebhookOpts          = &projectedtokenmountwebhook.WebhookOptions{}
		podSchedulerNameWebhookOpts             = &podschedulernamewebhook.WebhookOptions{}
		podZoneAffinityWebhookOpts              = &podzoneaffinity.WebhookOptions{}
		podTopologySpreadConstraintsWebhookOpts = &podtopologyspreadconstraints.WebhookOptions{}
		seccompProfileWebhookOpts               = &seccompprofilewebhook.WebhookOptions{}

		cmd = &cobra.Command{
			Use: "gardener-resource-manager",

			RunE: func(cmd *cobra.Command, args []string) error {
				verflag.PrintAndExitIfRequested()

				ctx, cancel := context.WithCancel(cmd.Context())
				defer cancel()

				log.Info("Starting gardener-resource-manager", "version", version.Get().GitVersion)
				cmd.Flags().VisitAll(func(flag *pflag.Flag) {
					log.Info(fmt.Sprintf("FLAG: --%s=%s", flag.Name, flag.Value)) //nolint:logcheck
				})

				if err := resourcemanagercmd.CompleteAll(
					managerOpts,
					sourceClientOpts,
					targetClusterOpts,
					resourceControllerOpts,
					secretControllerOpts,
					healthControllerOpts,
					gcControllerOpts,
					tokenInvalidatorControllerOpts,
					tokenRequestorControllerOpts,
					rootCAControllerOpts,
					csrApproverControllerOpts,
					projectedTokenMountWebhookOpts,
					podSchedulerNameWebhookOpts,
					podTopologySpreadConstraintsWebhookOpts,
					podZoneAffinityWebhookOpts,
					seccompProfileWebhookOpts,
				); err != nil {
					return err
				}

				managerOptions := manager.Options{Logger: log}
				resourcemanagerhealthz.DefaultAddOptions.Ctx = ctx

				managerOpts.Completed().Apply(&managerOptions)
				sourceClientOpts.Completed().ApplyManagerOptions(&managerOptions)
				sourceClientOpts.Completed().ApplyClientSet(&resourcemanagerhealthz.DefaultAddOptions.ClientSet)
				resourceControllerOpts.Completed().TargetCluster = targetClusterOpts.Completed().Cluster
				secretControllerOpts.Completed().ClassFilter = *resourceControllerOpts.Completed().ClassFilter
				healthControllerOpts.Completed().ClassFilter = *resourceControllerOpts.Completed().ClassFilter
				resourceControllerOpts.Completed().GarbageCollectorActivated = gcControllerOpts.Completed().SyncPeriod > 0
				if err := resourceControllerOpts.Completed().ApplyDefaultClusterId(ctx, log, sourceClientOpts.Completed().RESTConfig); err != nil {
					return err
				}
				healthControllerOpts.Completed().ClusterID = resourceControllerOpts.Completed().ClusterID
				healthControllerOpts.Completed().TargetCluster = targetClusterOpts.Completed().Cluster
				healthControllerOpts.Completed().TargetCacheDisabled = targetClusterOpts.Completed().DisableCachedClient
				gcControllerOpts.Completed().TargetCluster = targetClusterOpts.Completed().Cluster
				tokenInvalidatorControllerOpts.Completed().TargetCluster = targetClusterOpts.Completed().Cluster
				tokenRequestorControllerOpts.Completed().TargetCluster = targetClusterOpts.Completed().Cluster
				rootCAControllerOpts.Completed().TargetCluster = targetClusterOpts.Completed().Cluster
				csrApproverControllerOpts.Completed().TargetCluster = targetClusterOpts.Completed().Cluster
				csrApproverControllerOpts.Completed().Namespace = managerOptions.Namespace
				projectedTokenMountWebhookOpts.Completed().TargetCluster = targetClusterOpts.Completed().Cluster

				// setup manager
				mgr, err := manager.New(sourceClientOpts.Completed().RESTConfig, managerOptions)
				if err != nil {
					return fmt.Errorf("could not instantiate manager: %w", err)
				}

				if err := mgr.Add(targetClusterOpts.Completed().Cluster); err != nil {
					return fmt.Errorf("could not add target cluster to manager: %w", err)
				}

				log.Info("Setting up healthcheck endpoints")
				if err := mgr.AddReadyzCheck("informer-sync", gardenerhealthz.NewCacheSyncHealthz(mgr.GetCache())); err != nil {
					return err
				}
				if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
					return err
				}

				log.Info("Setting up readycheck for webhook server")
				if err := mgr.AddReadyzCheck("webhook-server", mgr.GetWebhookServer().StartedChecker()); err != nil {
					return err
				}

				if profilingOpts.EnableProfiling {
					if err := (routes.Profiling{}).AddToManager(mgr); err != nil {
						return fmt.Errorf("failed adding profiling handlers to manager: %w", err)
					}
					if profilingOpts.EnableContentionProfiling {
						goruntime.SetBlockProfileRate(1)
					}
				}

				// add controllers, health endpoint and webhooks to manager
				if err := resourcemanagercmd.AddAllToManager(mgr,
					// controllers
					resourcecontroller.AddToManager,
					secretcontroller.AddToManager,
					healthcontroller.AddToManager,
					garbagecollectorcontroller.AddToManager,
					tokeninvalidatorcontroller.AddToManager,
					tokenrequestorcontroller.AddToManager,
					rootcacontroller.AddToManager,
					csrapprovercontroller.AddToManager,
					// health endpoints
					resourcemanagerhealthz.AddToManager,
					// webhooks
					tokeninvalidatorwebhook.AddToManager,
					projectedtokenmountwebhook.AddToManager,
				); err != nil {
					return err
				}

				if podSchedulerNameWebhookOpts.Completed().Enabled {
					if err := podschedulernamewebhook.AddToManager(mgr); err != nil {
						return err
					}
				}

				if podTopologySpreadConstraintsWebhookOpts.Completed().Enabled {
					if err := podtopologyspreadconstraints.AddToManager(mgr); err != nil {
						return nil
					}
				}

				if podZoneAffinityWebhookOpts.Completed().Enabled {
					if err := podzoneaffinity.AddToManager(mgr); err != nil {
						return err
					}
				}

				if seccompProfileWebhookOpts.Completed().Enabled {
					if err := seccompprofilewebhook.AddToManager(mgr); err != nil {
						return err
					}
				}

				// start manager and exit if there was an error
				var wg sync.WaitGroup
				errChan := make(chan error)

				go func() {
					defer wg.Done()
					wg.Add(1)

					if err := mgr.Start(ctx); err != nil {
						errChan <- fmt.Errorf("error running manager: %w", err)
					}
				}()

				select {
				case err := <-errChan:
					cancel()
					wg.Wait()
					return err

				case <-cmd.Context().Done():
					log.Info("Stop signal received, shutting down")
					wg.Wait()
					return nil
				}
			},
			SilenceUsage: true,
		}
	)

	resourcemanagercmd.AddAllFlags(
		cmd.Flags(),
		managerOpts,
		profilingOpts,
		sourceClientOpts,
		targetClusterOpts,
		resourceControllerOpts,
		secretControllerOpts,
		healthControllerOpts,
		gcControllerOpts,
		tokenInvalidatorControllerOpts,
		tokenRequestorControllerOpts,
		rootCAControllerOpts,
		csrApproverControllerOpts,
		projectedTokenMountWebhookOpts,
		podSchedulerNameWebhookOpts,
		podTopologySpreadConstraintsWebhookOpts,
		podZoneAffinityWebhookOpts,
		seccompProfileWebhookOpts,
	)
	verflag.AddFlags(cmd.Flags())

	return cmd
}
