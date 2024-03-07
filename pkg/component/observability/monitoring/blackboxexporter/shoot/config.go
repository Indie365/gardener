// Copyright 2024 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
	"time"

	blackboxexporterconfig "github.com/prometheus/blackbox_exporter/config"
	prometheuscommonconfig "github.com/prometheus/common/config"
)

// Config returns the blackbox-exporter config for the shoot use-case.
func Config() blackboxexporterconfig.Config {
	return blackboxexporterconfig.Config{Modules: map[string]blackboxexporterconfig.Module{
		"http_kubernetes_service": {
			Prober:  "http",
			Timeout: 10 * time.Second,
			HTTP: blackboxexporterconfig.HTTPProbe{
				Headers: map[string]string{
					"Accept":          "*/*",
					"Accept-Language": "en-US",
				},
				HTTPClientConfig: prometheuscommonconfig.HTTPClientConfig{
					TLSConfig:       prometheuscommonconfig.TLSConfig{CAFile: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"},
					BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
				},
				IPProtocol: "ipv4",
			},
		},
	}}
}
