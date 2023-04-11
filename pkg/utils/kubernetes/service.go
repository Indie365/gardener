// Copyright 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package kubernetes

import (
	"fmt"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
)

// FQDNForService returns the fully qualified domain name of a service with the given name and namespace
func FQDNForService(name, namespace string) string {
	return fmt.Sprintf("%s.%s.svc.%s", name, namespace, v1beta1.DefaultDomain)
}

// DNSNamesForService returns the possible DNS names for a service with the given name and namespace.
func DNSNamesForService(name, namespace string) []string {
	return []string{
		name,
		fmt.Sprintf("%s.%s", name, namespace),
		fmt.Sprintf("%s.%s.svc", name, namespace),
		FQDNForService(name, namespace),
	}
}
