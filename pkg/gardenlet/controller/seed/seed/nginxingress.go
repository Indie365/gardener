// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package seed

import (
	"encoding/json"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1helper "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1/helper"
	"github.com/gardener/gardener/pkg/component"
	"github.com/gardener/gardener/pkg/component/extensions/dnsrecord"
	"github.com/gardener/gardener/pkg/utils"
)

func getManagedIngressDNSRecord(
	log logr.Logger,
	seedClient client.Client,
	gardenNamespaceName string,
	dnsConfig gardencorev1beta1.SeedDNS,
	secretData map[string][]byte,
	seedFQDN string,
	loadBalancerAddress string,
) component.DeployMigrateWaiter {
	values := &dnsrecord.Values{
		Name:                         "seed-ingress",
		SecretName:                   "seed-ingress",
		Namespace:                    gardenNamespaceName,
		SecretData:                   secretData,
		DNSName:                      seedFQDN,
		RecordType:                   extensionsv1alpha1helper.GetDNSRecordType(loadBalancerAddress),
		ReconcileOnlyOnChangeOrError: true,
	}

	if dnsConfig.Provider != nil {
		values.Type = dnsConfig.Provider.Type
	}

	if loadBalancerAddress != "" {
		values.Values = []string{loadBalancerAddress}
	}

	return dnsrecord.New(
		log,
		seedClient,
		values,
		dnsrecord.DefaultInterval,
		dnsrecord.DefaultSevereThreshold,
		dnsrecord.DefaultTimeout,
	)
}

func getConfig(seed *gardencorev1beta1.Seed) (map[string]string, error) {
	var (
		defaultConfig = map[string]interface{}{
			"server-name-hash-bucket-size": "256",
			"use-proxy-protocol":           "false",
			"worker-processes":             "2",
			"allow-snippet-annotations":    "true",
		}
		providerConfig = map[string]interface{}{}
	)
	if seed.Spec.Ingress != nil && seed.Spec.Ingress.Controller.ProviderConfig != nil {
		if err := json.Unmarshal(seed.Spec.Ingress.Controller.ProviderConfig.Raw, &providerConfig); err != nil {
			return nil, err
		}
	}

	return utils.InterfaceMapToStringMap(utils.MergeMaps(defaultConfig, providerConfig)), nil
}
