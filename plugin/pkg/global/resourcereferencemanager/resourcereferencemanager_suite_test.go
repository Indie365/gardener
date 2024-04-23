// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package resourcereferencemanager_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestResourceReferenceManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AdmissionPlugin Global ResourceReferenceManager Suite")
}
