/*
Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

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

// Code generated by lister-gen. DO NOT EDIT.

package internalversion

import (
	core "github.com/gardener/gardener/pkg/apis/core"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ShootLeftoverLister helps list ShootLeftovers.
// All objects returned here must be treated as read-only.
type ShootLeftoverLister interface {
	// List lists all ShootLeftovers in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*core.ShootLeftover, err error)
	// ShootLeftovers returns an object that can list and get ShootLeftovers.
	ShootLeftovers(namespace string) ShootLeftoverNamespaceLister
	ShootLeftoverListerExpansion
}

// shootLeftoverLister implements the ShootLeftoverLister interface.
type shootLeftoverLister struct {
	indexer cache.Indexer
}

// NewShootLeftoverLister returns a new ShootLeftoverLister.
func NewShootLeftoverLister(indexer cache.Indexer) ShootLeftoverLister {
	return &shootLeftoverLister{indexer: indexer}
}

// List lists all ShootLeftovers in the indexer.
func (s *shootLeftoverLister) List(selector labels.Selector) (ret []*core.ShootLeftover, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*core.ShootLeftover))
	})
	return ret, err
}

// ShootLeftovers returns an object that can list and get ShootLeftovers.
func (s *shootLeftoverLister) ShootLeftovers(namespace string) ShootLeftoverNamespaceLister {
	return shootLeftoverNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ShootLeftoverNamespaceLister helps list and get ShootLeftovers.
// All objects returned here must be treated as read-only.
type ShootLeftoverNamespaceLister interface {
	// List lists all ShootLeftovers in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*core.ShootLeftover, err error)
	// Get retrieves the ShootLeftover from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*core.ShootLeftover, error)
	ShootLeftoverNamespaceListerExpansion
}

// shootLeftoverNamespaceLister implements the ShootLeftoverNamespaceLister
// interface.
type shootLeftoverNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ShootLeftovers in the indexer for a given namespace.
func (s shootLeftoverNamespaceLister) List(selector labels.Selector) (ret []*core.ShootLeftover, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*core.ShootLeftover))
	})
	return ret, err
}

// Get retrieves the ShootLeftover from the indexer for a given namespace and name.
func (s shootLeftoverNamespaceLister) Get(name string) (*core.ShootLeftover, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(core.Resource("shootleftover"), name)
	}
	return obj.(*core.ShootLeftover), nil
}
