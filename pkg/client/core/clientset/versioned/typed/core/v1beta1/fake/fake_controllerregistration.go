// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeControllerRegistrations implements ControllerRegistrationInterface
type FakeControllerRegistrations struct {
	Fake *FakeCoreV1beta1
}

var controllerregistrationsResource = v1beta1.SchemeGroupVersion.WithResource("controllerregistrations")

var controllerregistrationsKind = v1beta1.SchemeGroupVersion.WithKind("ControllerRegistration")

// Get takes name of the controllerRegistration, and returns the corresponding controllerRegistration object, and an error if there is any.
func (c *FakeControllerRegistrations) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta1.ControllerRegistration, err error) {
	emptyResult := &v1beta1.ControllerRegistration{}
	obj, err := c.Fake.
		Invokes(testing.NewRootGetActionWithOptions(controllerregistrationsResource, name, options), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1beta1.ControllerRegistration), err
}

// List takes label and field selectors, and returns the list of ControllerRegistrations that match those selectors.
func (c *FakeControllerRegistrations) List(ctx context.Context, opts v1.ListOptions) (result *v1beta1.ControllerRegistrationList, err error) {
	emptyResult := &v1beta1.ControllerRegistrationList{}
	obj, err := c.Fake.
		Invokes(testing.NewRootListActionWithOptions(controllerregistrationsResource, controllerregistrationsKind, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.ControllerRegistrationList{ListMeta: obj.(*v1beta1.ControllerRegistrationList).ListMeta}
	for _, item := range obj.(*v1beta1.ControllerRegistrationList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested controllerRegistrations.
func (c *FakeControllerRegistrations) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchActionWithOptions(controllerregistrationsResource, opts))
}

// Create takes the representation of a controllerRegistration and creates it.  Returns the server's representation of the controllerRegistration, and an error, if there is any.
func (c *FakeControllerRegistrations) Create(ctx context.Context, controllerRegistration *v1beta1.ControllerRegistration, opts v1.CreateOptions) (result *v1beta1.ControllerRegistration, err error) {
	emptyResult := &v1beta1.ControllerRegistration{}
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateActionWithOptions(controllerregistrationsResource, controllerRegistration, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1beta1.ControllerRegistration), err
}

// Update takes the representation of a controllerRegistration and updates it. Returns the server's representation of the controllerRegistration, and an error, if there is any.
func (c *FakeControllerRegistrations) Update(ctx context.Context, controllerRegistration *v1beta1.ControllerRegistration, opts v1.UpdateOptions) (result *v1beta1.ControllerRegistration, err error) {
	emptyResult := &v1beta1.ControllerRegistration{}
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateActionWithOptions(controllerregistrationsResource, controllerRegistration, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1beta1.ControllerRegistration), err
}

// Delete takes name of the controllerRegistration and deletes it. Returns an error if one occurs.
func (c *FakeControllerRegistrations) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(controllerregistrationsResource, name, opts), &v1beta1.ControllerRegistration{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeControllerRegistrations) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionActionWithOptions(controllerregistrationsResource, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v1beta1.ControllerRegistrationList{})
	return err
}

// Patch applies the patch and returns the patched controllerRegistration.
func (c *FakeControllerRegistrations) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.ControllerRegistration, err error) {
	emptyResult := &v1beta1.ControllerRegistration{}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceActionWithOptions(controllerregistrationsResource, name, pt, data, opts, subresources...), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1beta1.ControllerRegistration), err
}
