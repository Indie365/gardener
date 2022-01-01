/*
Copyright (c) SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	core "github.com/gardener/gardener/pkg/apis/core"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeControllerDeployments implements ControllerDeploymentInterface
type FakeControllerDeployments struct {
	Fake *FakeCore
}

var controllerdeploymentsResource = schema.GroupVersionResource{Group: "core.gardener.cloud", Version: "", Resource: "controllerdeployments"}

var controllerdeploymentsKind = schema.GroupVersionKind{Group: "core.gardener.cloud", Version: "", Kind: "ControllerDeployment"}

// Get takes name of the controllerDeployment, and returns the corresponding controllerDeployment object, and an error if there is any.
func (c *FakeControllerDeployments) Get(ctx context.Context, name string, options v1.GetOptions) (result *core.ControllerDeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(controllerdeploymentsResource, name), &core.ControllerDeployment{})
	if obj == nil {
		return nil, err
	}
	return obj.(*core.ControllerDeployment), err
}

// List takes label and field selectors, and returns the list of ControllerDeployments that match those selectors.
func (c *FakeControllerDeployments) List(ctx context.Context, opts v1.ListOptions) (result *core.ControllerDeploymentList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(controllerdeploymentsResource, controllerdeploymentsKind, opts), &core.ControllerDeploymentList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &core.ControllerDeploymentList{ListMeta: obj.(*core.ControllerDeploymentList).ListMeta}
	for _, item := range obj.(*core.ControllerDeploymentList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested controllerDeployments.
func (c *FakeControllerDeployments) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(controllerdeploymentsResource, opts))
}

// Create takes the representation of a controllerDeployment and creates it.  Returns the server's representation of the controllerDeployment, and an error, if there is any.
func (c *FakeControllerDeployments) Create(ctx context.Context, controllerDeployment *core.ControllerDeployment, opts v1.CreateOptions) (result *core.ControllerDeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(controllerdeploymentsResource, controllerDeployment), &core.ControllerDeployment{})
	if obj == nil {
		return nil, err
	}
	return obj.(*core.ControllerDeployment), err
}

// Update takes the representation of a controllerDeployment and updates it. Returns the server's representation of the controllerDeployment, and an error, if there is any.
func (c *FakeControllerDeployments) Update(ctx context.Context, controllerDeployment *core.ControllerDeployment, opts v1.UpdateOptions) (result *core.ControllerDeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(controllerdeploymentsResource, controllerDeployment), &core.ControllerDeployment{})
	if obj == nil {
		return nil, err
	}
	return obj.(*core.ControllerDeployment), err
}

// Delete takes name of the controllerDeployment and deletes it. Returns an error if one occurs.
func (c *FakeControllerDeployments) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(controllerdeploymentsResource, name), &core.ControllerDeployment{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeControllerDeployments) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(controllerdeploymentsResource, listOpts)

	_, err := c.Fake.Invokes(action, &core.ControllerDeploymentList{})
	return err
}

// Patch applies the patch and returns the patched controllerDeployment.
func (c *FakeControllerDeployments) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *core.ControllerDeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(controllerdeploymentsResource, name, pt, data, subresources...), &core.ControllerDeployment{})
	if obj == nil {
		return nil, err
	}
	return obj.(*core.ControllerDeployment), err
}
