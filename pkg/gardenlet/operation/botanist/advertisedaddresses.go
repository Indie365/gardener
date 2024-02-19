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

package botanist

import (
	"context"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenerutils "github.com/gardener/gardener/pkg/utils/gardener"
)

// UpdateAdvertisedAddresses updates the shoot.status.advertisedAddresses with the list of
// addresses on which the API server of the shoot is accessible.
func (b *Botanist) UpdateAdvertisedAddresses(ctx context.Context) error {
	return b.Shoot.UpdateInfoStatus(ctx, b.GardenClient, false, func(shoot *gardencorev1beta1.Shoot) error {
		shoot.Status.AdvertisedAddresses = b.ToAdvertisedAddresses()
		return nil
	})
}

// ToAdvertisedAddresses returns list of advertised addresses on a Shoot cluster.
func (b *Botanist) ToAdvertisedAddresses() []gardencorev1beta1.ShootAdvertisedAddress {
	var addresses []gardencorev1beta1.ShootAdvertisedAddress

	if b.Shoot == nil {
		return addresses
	}

	if b.Shoot.ExternalClusterDomain != nil && len(*b.Shoot.ExternalClusterDomain) > 0 {
		addresses = append(addresses, gardencorev1beta1.ShootAdvertisedAddress{
			Name: "external",
			URL:  "https://" + gardenerutils.GetAPIServerDomain(*b.Shoot.ExternalClusterDomain),
		})
	}

	if len(b.Shoot.InternalClusterDomain) > 0 {
		addresses = append(addresses, gardencorev1beta1.ShootAdvertisedAddress{
			Name: "internal",
			URL:  "https://" + gardenerutils.GetAPIServerDomain(b.Shoot.InternalClusterDomain),
		})
	}

	if len(b.APIServerAddress) > 0 && len(addresses) == 0 {
		addresses = append(addresses, gardencorev1beta1.ShootAdvertisedAddress{
			Name: "unmanaged",
			URL:  "https://" + b.APIServerAddress,
		})
	}

	return addresses
}
