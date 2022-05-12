/*
Copyright 2021-2022 hliangzhao.

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

// QueueLister helps list Queues.
// All objects returned here must be treated as read-only.
type QueueLister interface {
	// List lists all Queues in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Queue, err error)
	// Get retrieves the Queue from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.Queue, error)
	QueueListerExpansion
}

// queueLister implements the QueueLister interface.
type queueLister struct {
	indexer cache.Indexer
}

// NewQueueLister returns a new QueueLister.
func NewQueueLister(indexer cache.Indexer) QueueLister {
	return &queueLister{indexer: indexer}
}

// List lists all Queues in the indexer.
func (s *queueLister) List(selector labels.Selector) (ret []*v1alpha1.Queue, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Queue))
	})
	return ret, err
}

// Get retrieves the Queue from the index for a given name.
func (s *queueLister) Get(name string) (*v1alpha1.Queue, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("queue"), name)
	}
	return obj.(*v1alpha1.Queue), nil
}
