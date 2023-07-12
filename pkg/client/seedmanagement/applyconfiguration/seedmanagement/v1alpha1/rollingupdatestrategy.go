/*
Copyright SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

// RollingUpdateStrategyApplyConfiguration represents an declarative configuration of the RollingUpdateStrategy type for use
// with apply.
type RollingUpdateStrategyApplyConfiguration struct {
	Partition *int32 `json:"partition,omitempty"`
}

// RollingUpdateStrategyApplyConfiguration constructs an declarative configuration of the RollingUpdateStrategy type for use with
// apply.
func RollingUpdateStrategy() *RollingUpdateStrategyApplyConfiguration {
	return &RollingUpdateStrategyApplyConfiguration{}
}

// WithPartition sets the Partition field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Partition field is set to the value of the last call.
func (b *RollingUpdateStrategyApplyConfiguration) WithPartition(value int32) *RollingUpdateStrategyApplyConfiguration {
	b.Partition = &value
	return b
}
