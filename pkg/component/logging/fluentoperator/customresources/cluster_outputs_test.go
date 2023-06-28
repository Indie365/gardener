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

package customresources_test

import (
	fluentbitv1alpha2 "github.com/fluent/fluent-operator/v2/apis/fluentbit/v1alpha2"
	"github.com/fluent/fluent-operator/v2/apis/fluentbit/v1alpha2/plugins/custom"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/gardener/gardener/pkg/component/logging/fluentoperator/customresources"
)

var _ = Describe("Logging", func() {
	Describe("#GetDefaultClusterOutputs", func() {
		var (
			labels = map[string]string{"some-key": "some-value"}
		)

		It("should return the expected DefaultClusterOutput custom resources", func() {
			fluentBitClusterOutputs := GetDefaultClusterOutput(labels)

			Expect(fluentBitClusterOutputs).To(Equal(
				&fluentbitv1alpha2.ClusterOutput{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "journald",
						Labels: labels,
					},
					Spec: fluentbitv1alpha2.OutputSpec{
						CustomPlugin: &custom.CustomPlugin{
							Config: `Name gardenervali
Match journald.*
Labels {origin="seed-journald"}
RemoveKeys kubernetes,stream,hostname,unit
LabelMapPath {"hostname":"host_name","unit":"systemd_component"}
QueueDir /fluent-bit/buffers
QueueName seed-journald
LogLevel info
Url http://logging.garden.svc:3100/vali/api/v1/push
BatchWait 60s
BatchSize 30720
LineFormat json
SortByTimestamp true
DropSingleKey false
AutoKubernetesLabels false
HostnameKeyValue nodename ${NODE_NAME}
MaxRetries 3
Timeout 10s
MinBackoff 30s
Buffer true
BufferType dque
QueueSegmentSize 300
QueueSync normal
NumberOfBatchIDs 5
`,
						},
					},
				},
			))
		})
	})

	Describe("#GetDynamicClusterOutput", func() {
		var (
			labels = map[string]string{"some-key": "some-value"}
		)

		It("should return the expected DynamicClusterOutput custom resources", func() {
			fluentBitClusterOutputs := GetDynamicClusterOutput(labels)

			Expect(fluentBitClusterOutputs).To(Equal(
				&fluentbitv1alpha2.ClusterOutput{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "dynamic-vali",
						Labels: labels,
					},
					Spec: fluentbitv1alpha2.OutputSpec{
						CustomPlugin: &custom.CustomPlugin{
							Config: `Name gardenervali
Match kubernetes.*
Labels {origin="seed"}
DropSingleKey false
LabelSelector gardener.cloud/role:shoot
DynamicHostPath {"kubernetes": {"namespace_name": "namespace"}}
DynamicHostPrefix http://logging.
DynamicHostSuffix .svc:3100/vali/api/v1/push
DynamicHostRegex ^shoot-
QueueDir /fluent-bit/buffers/seed
QueueName seed-dynamic
SendDeletedClustersLogsToDefaultClient true
CleanExpiredClientsPeriod 1h
ControllerSyncTimeout 120s
PreservedLabels origin,namespace_name,pod_name
TenantID operator
LogLevel info
Url http://logging.garden.svc:3100/vali/api/v1/push
BatchWait 60s
BatchSize 30720
LineFormat json
SortByTimestamp true
DropSingleKey false
AutoKubernetesLabels false
HostnameKeyValue nodename ${NODE_NAME}
MaxRetries 3
Timeout 10s
MinBackoff 30s
Buffer true
BufferType dque
QueueSegmentSize 300
QueueSync normal
NumberOfBatchIDs 5
RemoveKeys kubernetes,stream,time,tag,gardenuser,job
LabelMapPath {"kubernetes": {"container_name":"container_name","container_id":"container_id","namespace_name":"namespace_name","pod_name":"pod_name"},"severity": "severity","job": "job"}
FallbackToTagWhenMetadataIsMissing true
TagKey tag
DropLogEntryWithoutK8sMetadata true
`,
						},
					},
				},
			))
		})
	})

	Describe("#GetStaticClusterOutput", func() {
		var (
			labels = map[string]string{"some-key": "some-value"}
		)

		It("should return the expected DynamicClusterOutput custom resources", func() {
			fluentBitClusterOutputs := GetStaticClusterOutput(labels)

			Expect(fluentBitClusterOutputs).To(Equal(
				&fluentbitv1alpha2.ClusterOutput{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "static-vali",
						Labels: labels,
					},
					Spec: fluentbitv1alpha2.OutputSpec{
						CustomPlugin: &custom.CustomPlugin{
							Config: `Name gardenervali
Match kubernetes.*
Labels {origin="garden"}
QueueDir /fluent-bit/buffers/garden
QueueName gardener-operator-static
LogLevel info
Url http://logging.garden.svc:3100/vali/api/v1/push
BatchWait 60s
BatchSize 30720
LineFormat json
SortByTimestamp true
DropSingleKey false
AutoKubernetesLabels false
HostnameKeyValue nodename ${NODE_NAME}
MaxRetries 3
Timeout 10s
MinBackoff 30s
Buffer true
BufferType dque
QueueSegmentSize 300
QueueSync normal
NumberOfBatchIDs 5
RemoveKeys kubernetes,stream,time,tag,gardenuser,job
LabelMapPath {"kubernetes": {"container_name":"container_name","container_id":"container_id","namespace_name":"namespace_name","pod_name":"pod_name"},"severity": "severity","job": "job"}
FallbackToTagWhenMetadataIsMissing true
TagKey tag
DropLogEntryWithoutK8sMetadata true
`,
						},
					},
				},
			))
		})
	})
})
