// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package kubecontrollermanager_test

import (
	"path/filepath"

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"

	. "github.com/gardener/gardener/pkg/component/kubecontrollermanager"
	"github.com/gardener/gardener/pkg/component/test"
)

var _ = Describe("Monitoring", func() {
	It("should successfully test the scrape config", func() {
		kubeControllerManager := New(logr.Discard(), nil, "", nil, Values{RuntimeVersion: semver.MustParse("1.25.0"), TargetVersion: semver.MustParse("1.26.4")})
		test.ScrapeConfigs(kubeControllerManager, expectedScrapeConfig)
	})

	It("should successfully test the alerting rules", func() {
		kubeControllerManager := New(logr.Discard(), nil, "", nil, Values{RuntimeVersion: semver.MustParse("1.25.0"), TargetVersion: semver.MustParse("1.26.4")})

		test.AlertingRulesWithPromtool(
			kubeControllerManager,
			map[string]string{"kube-controller-manager.rules.yaml": expectedAlertingRule},
			filepath.Join("testdata", "monitoring_alertingrules.yaml"),
		)
	})
})

const (
	expectedScrapeConfig = `job_name: kube-controller-manager
scheme: https
tls_config:
  insecure_skip_verify: true
authorization:
  type: Bearer
  credentials_file: /var/run/secrets/gardener.cloud/shoot/token/token
honor_labels: false
scrape_timeout: 15s
kubernetes_sd_configs:
- role: endpoints
  namespaces:
    names: []
relabel_configs:
- source_labels:
  - __meta_kubernetes_service_name
  - __meta_kubernetes_endpoint_port_name
  action: keep
  regex: kube-controller-manager;metrics
- action: labelmap
  regex: __meta_kubernetes_service_label_(.+)
- source_labels: [ __meta_kubernetes_pod_name ]
  target_label: pod
metric_relabel_configs:
- source_labels: [ __name__ ]
  regex: ^(rest_client_requests_total|process_max_fds|process_open_fds)$
  action: keep
`

	expectedAlertingRule = `groups:
- name: kube-controller-manager.rules
  rules:
  - alert: KubeControllerManagerDown
    expr: absent(up{job="kube-controller-manager"} == 1)
    for: 15m
    labels:
      service: kube-controller-manager
      severity: critical
      type: seed
      visibility: all
    annotations:
      description: Deployments and replication controllers are not making progress.
      summary: Kube Controller Manager is down.`
)
