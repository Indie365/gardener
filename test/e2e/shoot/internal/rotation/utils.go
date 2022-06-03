// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package rotation

import (
	"sort"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var managedByGardenletSecretsManager = client.MatchingLabels{
	"managed-by":       "secrets-manager",
	"manager-identity": "gardenlet",
}

type secretConfigNamesToSecrets map[string][]corev1.Secret

func groupByName(allSecrets []corev1.Secret) secretConfigNamesToSecrets {
	grouped := make(secretConfigNamesToSecrets)
	for _, secret := range allSecrets {
		grouped[secret.Labels["name"]] = append(grouped[secret.Labels["name"]], secret)
	}

	// sort by age
	for _, secrets := range grouped {
		sort.Slice(secrets, func(i, j int) bool {
			return secrets[i].CreationTimestamp.Before(&secrets[j].CreationTimestamp)
		})
	}
	return grouped
}
