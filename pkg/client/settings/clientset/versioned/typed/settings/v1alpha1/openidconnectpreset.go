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

package v1alpha1

import (
	"context"
	json "encoding/json"
	"fmt"
	"time"

	v1alpha1 "github.com/gardener/gardener/pkg/apis/settings/v1alpha1"
	settingsv1alpha1 "github.com/gardener/gardener/pkg/client/settings/applyconfiguration/settings/v1alpha1"
	scheme "github.com/gardener/gardener/pkg/client/settings/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// OpenIDConnectPresetsGetter has a method to return a OpenIDConnectPresetInterface.
// A group's client should implement this interface.
type OpenIDConnectPresetsGetter interface {
	OpenIDConnectPresets(namespace string) OpenIDConnectPresetInterface
}

// OpenIDConnectPresetInterface has methods to work with OpenIDConnectPreset resources.
type OpenIDConnectPresetInterface interface {
	Create(ctx context.Context, openIDConnectPreset *v1alpha1.OpenIDConnectPreset, opts v1.CreateOptions) (*v1alpha1.OpenIDConnectPreset, error)
	Update(ctx context.Context, openIDConnectPreset *v1alpha1.OpenIDConnectPreset, opts v1.UpdateOptions) (*v1alpha1.OpenIDConnectPreset, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.OpenIDConnectPreset, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.OpenIDConnectPresetList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.OpenIDConnectPreset, err error)
	Apply(ctx context.Context, openIDConnectPreset *settingsv1alpha1.OpenIDConnectPresetApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.OpenIDConnectPreset, err error)
	OpenIDConnectPresetExpansion
}

// openIDConnectPresets implements OpenIDConnectPresetInterface
type openIDConnectPresets struct {
	client rest.Interface
	ns     string
}

// newOpenIDConnectPresets returns a OpenIDConnectPresets
func newOpenIDConnectPresets(c *SettingsV1alpha1Client, namespace string) *openIDConnectPresets {
	return &openIDConnectPresets{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the openIDConnectPreset, and returns the corresponding openIDConnectPreset object, and an error if there is any.
func (c *openIDConnectPresets) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.OpenIDConnectPreset, err error) {
	result = &v1alpha1.OpenIDConnectPreset{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("openidconnectpresets").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of OpenIDConnectPresets that match those selectors.
func (c *openIDConnectPresets) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.OpenIDConnectPresetList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.OpenIDConnectPresetList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("openidconnectpresets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested openIDConnectPresets.
func (c *openIDConnectPresets) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("openidconnectpresets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a openIDConnectPreset and creates it.  Returns the server's representation of the openIDConnectPreset, and an error, if there is any.
func (c *openIDConnectPresets) Create(ctx context.Context, openIDConnectPreset *v1alpha1.OpenIDConnectPreset, opts v1.CreateOptions) (result *v1alpha1.OpenIDConnectPreset, err error) {
	result = &v1alpha1.OpenIDConnectPreset{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("openidconnectpresets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(openIDConnectPreset).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a openIDConnectPreset and updates it. Returns the server's representation of the openIDConnectPreset, and an error, if there is any.
func (c *openIDConnectPresets) Update(ctx context.Context, openIDConnectPreset *v1alpha1.OpenIDConnectPreset, opts v1.UpdateOptions) (result *v1alpha1.OpenIDConnectPreset, err error) {
	result = &v1alpha1.OpenIDConnectPreset{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("openidconnectpresets").
		Name(openIDConnectPreset.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(openIDConnectPreset).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the openIDConnectPreset and deletes it. Returns an error if one occurs.
func (c *openIDConnectPresets) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("openidconnectpresets").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *openIDConnectPresets) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("openidconnectpresets").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched openIDConnectPreset.
func (c *openIDConnectPresets) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.OpenIDConnectPreset, err error) {
	result = &v1alpha1.OpenIDConnectPreset{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("openidconnectpresets").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied openIDConnectPreset.
func (c *openIDConnectPresets) Apply(ctx context.Context, openIDConnectPreset *settingsv1alpha1.OpenIDConnectPresetApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.OpenIDConnectPreset, err error) {
	if openIDConnectPreset == nil {
		return nil, fmt.Errorf("openIDConnectPreset provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(openIDConnectPreset)
	if err != nil {
		return nil, err
	}
	name := openIDConnectPreset.Name
	if name == nil {
		return nil, fmt.Errorf("openIDConnectPreset.Name must be provided to Apply")
	}
	result = &v1alpha1.OpenIDConnectPreset{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("openidconnectpresets").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
