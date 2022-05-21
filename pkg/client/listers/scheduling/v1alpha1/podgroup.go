/*
Copyright 2021-2022 The Volcano Authors.

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

package v1alpha1

import (
	v1alpha1 "github.com/hliangzhao/volcano/pkg/apis/scheduling/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// PodGroupLister helps list PodGroups.
// All objects returned here must be treated as read-only.
type PodGroupLister interface {
	// List lists all PodGroups in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.PodGroup, err error)
	// PodGroups returns an object that can list and get PodGroups.
	PodGroups(namespace string) PodGroupNamespaceLister
	PodGroupListerExpansion
}

// podGroupLister implements the PodGroupLister interface.
type podGroupLister struct {
	indexer cache.Indexer
}

// NewPodGroupLister returns a new PodGroupLister.
func NewPodGroupLister(indexer cache.Indexer) PodGroupLister {
	return &podGroupLister{indexer: indexer}
}

// List lists all PodGroups in the indexer.
func (s *podGroupLister) List(selector labels.Selector) (ret []*v1alpha1.PodGroup, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.PodGroup))
	})
	return ret, err
}

// PodGroups returns an object that can list and get PodGroups.
func (s *podGroupLister) PodGroups(namespace string) PodGroupNamespaceLister {
	return podGroupNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// PodGroupNamespaceLister helps list and get PodGroups.
// All objects returned here must be treated as read-only.
type PodGroupNamespaceLister interface {
	// List lists all PodGroups in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.PodGroup, err error)
	// Get retrieves the PodGroup from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.PodGroup, error)
	PodGroupNamespaceListerExpansion
}

// podGroupNamespaceLister implements the PodGroupNamespaceLister
// interface.
type podGroupNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all PodGroups in the indexer for a given namespace.
func (s podGroupNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.PodGroup, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.PodGroup))
	})
	return ret, err
}

// Get retrieves the PodGroup from the indexer for a given namespace and name.
func (s podGroupNamespaceLister) Get(name string) (*v1alpha1.PodGroup, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("podgroup"), name)
	}
	return obj.(*v1alpha1.PodGroup), nil
}
