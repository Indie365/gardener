// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package certificatesigningrequest

import (
	"fmt"

	"github.com/gardener/gardener/pkg/client/kubernetes"

	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// ControllerName is the name of this controller.
	ControllerName = "certificatesigningrequest"
)

// AddToManager adds a new CSR controller to the given manager.
func AddToManager(mgr manager.Manager, k8sClient kubernetes.Interface) error {
	reconciler := &reconciler{
		gardenClient: mgr.GetClient(),
		k8sClient:    k8sClient,
	}

	ctrlOptions := controller.Options{
		Reconciler: reconciler,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	reconciler.logger = c.GetLogger()

	csr := &certificatesv1beta1.CertificateSigningRequest{}
	if err := c.Watch(&source.Kind{Type: csr}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %w", csr, err)
	}

	return nil
}
