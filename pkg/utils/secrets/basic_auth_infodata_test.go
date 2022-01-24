// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package secrets_test

import (
	. "github.com/gardener/gardener/pkg/utils/secrets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BasicAuth InfoData", func() {
	var (
		basicAuthJSON     = []byte(`{"password":"foo"}`)
		basicAuthInfoData = &BasicAuthInfoData{
			Password: "foo",
		}
	)

	Describe("#UnmarshalBasicAuth", func() {
		It("should properly unmarshal BasicAuthJSONData into BasicAuthInfoData", func() {
			infoData, err := UnmarshalBasicAuth(basicAuthJSON)
			Expect(err).NotTo(HaveOccurred())
			Expect(infoData).To(Equal(basicAuthInfoData))
		})
	})

	Describe("#Marshal", func() {
		It("should properly marshal BaiscAuthInfoData into BasicAuthJSONData", func() {
			data, err := basicAuthInfoData.Marshal()
			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(Equal(basicAuthJSON))
		})
	})

	Describe("#TypeVersion", func() {
		It("should return the correct TypeVersion", func() {
			typeVersion := basicAuthInfoData.TypeVersion()
			Expect(typeVersion).To(Equal(BasicAuthDataType))
		})
	})

	Describe("#NewBasicAuthInfoData", func() {
		It("should return new BasicAuthInfoData from the passed password", func() {
			newBasicAuthInfoData := NewBasicAuthInfoData("foo")
			Expect(newBasicAuthInfoData).To(Equal(basicAuthInfoData))
		})
	})
})
