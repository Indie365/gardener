// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package health

import (
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

// CheckAPIService checks whether the given APIService is healthy.
// An APIService is considered healthy if it has the `Available` condition and its status is `True`.
func CheckAPIService(apiService *apiregistrationv1.APIService) error {
	const (
		requiredCondition       = apiregistrationv1.Available
		requiredConditionStatus = apiregistrationv1.ConditionTrue
	)

	for _, condition := range apiService.Status.Conditions {
		if condition.Type == requiredCondition {
			return checkConditionState(
				string(requiredConditionStatus),
				string(condition.Status),
				condition.Reason,
				condition.Message,
			)
		}
	}
	return requiredConditionMissing(string(requiredCondition))
}
