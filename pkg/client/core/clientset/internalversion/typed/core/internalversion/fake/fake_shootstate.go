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

// FakeShootStates implements ShootStateInterface
type FakeShootStates struct {
	Fake *FakeCore
	ns   string
}

var shootstatesResource = schema.GroupVersionResource{Group: "core.gardener.cloud", Version: "", Resource: "shootstates"}

var shootstatesKind = schema.GroupVersionKind{Group: "core.gardener.cloud", Version: "", Kind: "ShootState"}

// Get takes name of the shootState, and returns the corresponding shootState object, and an error if there is any.
func (c *FakeShootStates) Get(ctx context.Context, name string, options v1.GetOptions) (result *core.ShootState, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(shootstatesResource, c.ns, name), &core.ShootState{})

	if obj == nil {
		return nil, err
	}
	return obj.(*core.ShootState), err
}

// List takes label and field selectors, and returns the list of ShootStates that match those selectors.
func (c *FakeShootStates) List(ctx context.Context, opts v1.ListOptions) (result *core.ShootStateList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(shootstatesResource, shootstatesKind, c.ns, opts), &core.ShootStateList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &core.ShootStateList{ListMeta: obj.(*core.ShootStateList).ListMeta}
	for _, item := range obj.(*core.ShootStateList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested shootStates.
func (c *FakeShootStates) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(shootstatesResource, c.ns, opts))

}

// Create takes the representation of a shootState and creates it.  Returns the server's representation of the shootState, and an error, if there is any.
func (c *FakeShootStates) Create(ctx context.Context, shootState *core.ShootState, opts v1.CreateOptions) (result *core.ShootState, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(shootstatesResource, c.ns, shootState), &core.ShootState{})

	if obj == nil {
		return nil, err
	}
	return obj.(*core.ShootState), err
}

// Update takes the representation of a shootState and updates it. Returns the server's representation of the shootState, and an error, if there is any.
func (c *FakeShootStates) Update(ctx context.Context, shootState *core.ShootState, opts v1.UpdateOptions) (result *core.ShootState, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(shootstatesResource, c.ns, shootState), &core.ShootState{})

	if obj == nil {
		return nil, err
	}
	return obj.(*core.ShootState), err
}

// Delete takes name of the shootState and deletes it. Returns an error if one occurs.
func (c *FakeShootStates) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(shootstatesResource, c.ns, name, opts), &core.ShootState{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeShootStates) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(shootstatesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &core.ShootStateList{})
	return err
}

// Patch applies the patch and returns the patched shootState.
func (c *FakeShootStates) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *core.ShootState, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(shootstatesResource, c.ns, name, pt, data, subresources...), &core.ShootState{})

	if obj == nil {
		return nil, err
	}
	return obj.(*core.ShootState), err
}
