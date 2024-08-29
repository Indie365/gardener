// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package garden_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	blackboxexporterconfig "github.com/prometheus/blackbox_exporter/config"
	prometheuscommonconfig "github.com/prometheus/common/config"

	. "github.com/gardener/gardener/pkg/component/observability/monitoring/blackboxexporter/garden"
)

var _ = Describe("Config", func() {
	Describe("#Config", func() {
		It("should return the expected config for the garden's blackbox-exporter", func() {
			Expect(Config(false, true)).To(Equal(blackboxexporterconfig.Config{Modules: map[string]blackboxexporterconfig.Module{
				"http_gardener_apiserver": {
					Prober:  "http",
					Timeout: 10 * time.Second,
					HTTP: blackboxexporterconfig.HTTPProbe{
						Headers: map[string]string{
							"Accept":          "*/*",
							"Accept-Language": "en-US",
						},
						HTTPClientConfig: prometheuscommonconfig.HTTPClientConfig{
							TLSConfig: prometheuscommonconfig.TLSConfig{CAFile: "/var/run/secrets/blackbox_exporter/gardener-ca/bundle.crt"},
						},
						IPProtocol: "ipv4",
					},
				},
				"http_kube_apiserver": {
					Prober:  "http",
					Timeout: 10 * time.Second,
					HTTP: blackboxexporterconfig.HTTPProbe{
						Headers: map[string]string{
							"Accept":          "*/*",
							"Accept-Language": "en-US",
						},
						HTTPClientConfig: prometheuscommonconfig.HTTPClientConfig{
							TLSConfig:       prometheuscommonconfig.TLSConfig{CAFile: "/var/run/secrets/blackbox_exporter/cluster-access/bundle.crt"},
							BearerTokenFile: "/var/run/secrets/blackbox_exporter/cluster-access/token",
						},
						IPProtocol: "ipv4",
					},
				},
				"http_kube_apiserver_root_cas": {
					Prober:  "http",
					Timeout: 10 * time.Second,
					HTTP: blackboxexporterconfig.HTTPProbe{
						Headers: map[string]string{
							"Accept":          "*/*",
							"Accept-Language": "en-US",
						},
						HTTPClientConfig: prometheuscommonconfig.HTTPClientConfig{
							BearerTokenFile: "/var/run/secrets/blackbox_exporter/cluster-access/token",
						},
						IPProtocol: "ipv4",
					},
				},
				"http_gardener_dashboard": {
					Prober:  "http",
					Timeout: 10 * time.Second,
					HTTP: blackboxexporterconfig.HTTPProbe{
						Headers: map[string]string{
							"Accept":          "*/*",
							"Accept-Language": "en-US",
						},
						IPProtocol: "ipv4",
					},
				},
				"http_gardener_discovery_server": {
					Prober:  "http",
					Timeout: 10 * time.Second,
					HTTP: blackboxexporterconfig.HTTPProbe{
						Headers: map[string]string{
							"Accept":          "*/*",
							"Accept-Language": "en-US",
						},
						IPProtocol: "ipv4",
					},
				},
				HttpWebhookServerModuleName: {
					Prober:  "http",
					Timeout: 10 * time.Second,
					HTTP: blackboxexporterconfig.HTTPProbe{
						Headers: map[string]string{
							"Accept":          "*/*",
							"Accept-Language": "en-US",
						},
						HTTPClientConfig: prometheuscommonconfig.HTTPClientConfig{
							TLSConfig: prometheuscommonconfig.TLSConfig{CAFile: "/var/run/secrets/blackbox_exporter/gardener-ca/bundle.crt"},
						},
						ValidStatusCodes: []int{
							200, 201, 202, 203, 204, 205, 206, 207, 208, 226,
							300, 301, 302, 303, 304, 305, 307, 308,
							400, 401, 402, 403, 404, 405, 406, 407, 408, 409, 410, 411, 412, 413, 414, 415, 416, 417, 418, 421, 422, 423, 424, 426, 428, 429, 431, 451,
						},
						IPProtocol: "ipv4",
					},
				},
			}}))
		})

		When("isDashboardCertificateIssuedByGardener is true", func() {
			It("should configure the Gardener CA for the http_gardener_dashboard module", func() {
				Expect(Config(true, true).Modules["http_gardener_dashboard"].HTTP.HTTPClientConfig.TLSConfig).To(Equal(
					prometheuscommonconfig.TLSConfig{
						CAFile: "/var/run/secrets/blackbox_exporter/gardener-ca/bundle.crt"},
				))
			})
		})

		When("isGardenerDiscoveryServerEnabled is false", func() {
			It("should remove configuration for http_gardener_discovery_server module", func() {
				Expect(Config(false, false).Modules).ToNot(HaveKey("http_gardener_discovery_server"))
			})
		})
	})
})
