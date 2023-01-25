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

package istio

import (
	"context"
	"embed"
	"path/filepath"
	"strings"
	"time"

	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/gardener/gardener/pkg/operation"
	"github.com/gardener/gardener/pkg/operation/botanist/component"
	"github.com/gardener/gardener/pkg/utils/managedresources"
)

const (
	// ManagedResourceControlName is the name of the ManagedResource containing the resource specifications.
	ManagedResourceControlName = "istio"

	istiodServiceName            = "istiod"
	istiodServicePortNameMetrics = "metrics"
	portWebhookServer            = 10250
)

var (
	//go:embed charts/istio/istio-istiod
	chartIstiod     embed.FS
	chartPathIstiod = filepath.Join("charts", "istio", "istio-istiod")
)

type istiod struct {
	client        client.Client
	chartRenderer chartrenderer.Interface
	values        Values
}

// Values contains configuration values for the Istiod component.
type IstiodValues struct {
	Namespace   string
	Image       string
	TrustDomain string
	Zones       []string
}

// Values contains configuration values for the Istio component.
type Values struct {
	Istiod         IstiodValues
	IngressGateway []IngressGatewayValues
}

// NewIstio can be used to deploy istio's istiod in a namespace.
// Destroy does nothing.
func NewIstio(
	client client.Client,
	chartRenderer chartrenderer.Interface,
	values Values,
) component.DeployWaiter {
	return &istiod{
		client:        client,
		chartRenderer: chartRenderer,
		values:        values,
	}
}

func (i *istiod) Deploy(ctx context.Context) error {
	istiodNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: i.values.Istiod.Namespace}}
	if _, err := controllerutils.CreateOrGetAndMergePatch(ctx, i.client, istiodNamespace, func() error {
		metav1.SetMetaDataLabel(&istiodNamespace.ObjectMeta, "istio-operator-managed", "Reconcile")
		metav1.SetMetaDataLabel(&istiodNamespace.ObjectMeta, "istio-injection", "disabled")
		metav1.SetMetaDataLabel(&istiodNamespace.ObjectMeta, resourcesv1alpha1.HighAvailabilityConfigConsider, "true")
		metav1.SetMetaDataLabel(&istiodNamespace.ObjectMeta, v1beta1constants.GardenRole, v1beta1constants.GardenRoleIstioSystem)
		metav1.SetMetaDataAnnotation(&istiodNamespace.ObjectMeta, resourcesv1alpha1.HighAvailabilityConfigZones, strings.Join(i.values.Istiod.Zones, ","))
		return nil
	}); err != nil {
		return err
	}

	// TODO(mvladev): Rotate this on every istio version upgrade.
	for _, ingressGateway := range i.values.IngressGateway {
		for _, filterName := range []string{"tcp-stats-filter-1.11", "stats-filter-1.11", "tcp-stats-filter-1.12", "stats-filter-1.12"} {
			if err := client.IgnoreNotFound(i.client.Delete(ctx, &networkingv1alpha3.EnvoyFilter{
				ObjectMeta: metav1.ObjectMeta{Name: filterName, Namespace: ingressGateway.Namespace},
			})); err != nil {
				return err
			}
		}
	}

	renderedIstiodChart, err := i.generateIstiodChart()
	if err != nil {
		return err
	}

	renderedIstioIngressGatewayChart, err := i.generateIstioIngressGatewayChart()
	if err != nil {
		return err
	}

	for _, istioIngressGateway := range i.values.IngressGateway {
		gatewayNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: istioIngressGateway.Namespace}}
		if _, err := controllerutils.CreateOrGetAndMergePatch(ctx, i.client, gatewayNamespace, func() error {
			metav1.SetMetaDataLabel(&gatewayNamespace.ObjectMeta, "istio-operator-managed", "Reconcile")
			metav1.SetMetaDataLabel(&gatewayNamespace.ObjectMeta, "istio-injection", "disabled")
			metav1.SetMetaDataLabel(&gatewayNamespace.ObjectMeta, v1beta1constants.GardenRole, v1beta1constants.GardenRoleIstioIngress)

			if value, ok := istioIngressGateway.Labels[v1beta1constants.GardenRole]; ok && strings.HasPrefix(value, v1beta1constants.GardenRoleExposureClassHandler) {
				metav1.SetMetaDataLabel(&gatewayNamespace.ObjectMeta, v1beta1constants.GardenRole, value)
			}
			if value, ok := istioIngressGateway.Labels[v1beta1constants.LabelExposureClassHandlerName]; ok {
				metav1.SetMetaDataLabel(&gatewayNamespace.ObjectMeta, v1beta1constants.LabelExposureClassHandlerName, value)
			}

			if value, ok := istioIngressGateway.Labels[operation.IstioDefaultZoneKey]; ok {
				metav1.SetMetaDataLabel(&gatewayNamespace.ObjectMeta, operation.IstioDefaultZoneKey, value)
			}

			metav1.SetMetaDataLabel(&gatewayNamespace.ObjectMeta, resourcesv1alpha1.HighAvailabilityConfigConsider, "true")
			zones := i.values.Istiod.Zones
			if len(istioIngressGateway.Zones) > 0 {
				zones = istioIngressGateway.Zones
			}
			metav1.SetMetaDataAnnotation(&gatewayNamespace.ObjectMeta, resourcesv1alpha1.HighAvailabilityConfigZones, strings.Join(zones, ","))
			if len(zones) == 1 {
				metav1.SetMetaDataAnnotation(&gatewayNamespace.ObjectMeta, resourcesv1alpha1.HighAvailabilityConfigZonePinning, "true")
			}
			return nil
		}); err != nil {
			return err
		}
	}

	renderedChart := renderedIstiodChart
	renderedChart.Manifests = append(renderedChart.Manifests, renderedIstioIngressGatewayChart.Manifests...)
	registry := managedresources.NewRegistry(kubernetes.SeedScheme, kubernetes.SeedCodec, kubernetes.SeedSerializer)

	for _, transformer := range getIstioSystemNetworkPolicyTransformers() {
		obj := &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      transformer.name,
				Namespace: i.values.Istiod.Namespace,
			},
		}

		if err := transformer.transform(obj)(); err != nil {
			return err
		}

		if err := registry.Add(obj); err != nil {
			return err
		}
	}

	for _, istioIngressGateway := range i.values.IngressGateway {
		for _, transformer := range getIstioIngressNetworkPolicyTransformers() {
			obj := &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      transformer.name,
					Namespace: istioIngressGateway.Namespace,
				},
			}

			if err := transformer.transform(obj)(); err != nil {
				return err
			}

			if err := registry.Add(obj); err != nil {
				return err
			}
		}
	}

	chartsMap := renderedChart.AsSecretData()
	objMap := registry.SerializedObjects()

	for key := range objMap {
		chartsMap[key] = objMap[key]
	}

	return managedresources.CreateForSeed(ctx, i.client, i.values.Istiod.Namespace, ManagedResourceControlName, false, chartsMap)
}

func (i *istiod) Destroy(ctx context.Context) error {
	if err := managedresources.DeleteForSeed(ctx, i.client, i.values.Istiod.Namespace, ManagedResourceControlName); err != nil {
		return err
	}

	if err := i.client.Delete(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: i.values.Istiod.Namespace,
		},
	}); client.IgnoreNotFound(err) != nil {
		return err
	}

	for _, istioIngressGateway := range i.values.IngressGateway {
		if err := i.client.Delete(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: istioIngressGateway.Namespace,
			},
		}); client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	return nil
}

// TimeoutWaitForManagedResource is the timeout used while waiting for the ManagedResources to become healthy
// or deleted.
var TimeoutWaitForManagedResource = 2 * time.Minute

func (i *istiod) Wait(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, TimeoutWaitForManagedResource)
	defer cancel()

	return managedresources.WaitUntilHealthy(timeoutCtx, i.client, i.values.Istiod.Namespace, ManagedResourceControlName)
}

func (i *istiod) WaitCleanup(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, TimeoutWaitForManagedResource)
	defer cancel()

	return managedresources.WaitUntilDeleted(timeoutCtx, i.client, i.values.Istiod.Namespace, ManagedResourceControlName)
}

func (i *istiod) generateIstiodChart() (*chartrenderer.RenderedChart, error) {
	return i.chartRenderer.RenderEmbeddedFS(chartIstiod, chartPathIstiod, ManagedResourceControlName, i.values.Istiod.Namespace, map[string]interface{}{
		"serviceName": istiodServiceName,
		"trustDomain": i.values.Istiod.TrustDomain,
		"labels": map[string]interface{}{
			"app":   "istiod",
			"istio": "pilot",
		},
		"deployNamespace":   false,
		"priorityClassName": "istiod",
		"ports": map[string]interface{}{
			"https": portWebhookServer,
		},
		"portsNames": map[string]interface{}{
			"metrics": istiodServicePortNameMetrics,
		},
		"image": i.values.Istiod.Image,
	})
}
