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
// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/hliangzhao/volcano/pkg/apis/nodeinfo/v1alpha1"
	scheme "github.com/hliangzhao/volcano/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// NumatopologiesGetter has a method to return a NumatopologyInterface.
// A group's client should implement this interface.
type NumatopologiesGetter interface {
	Numatopologies() NumatopologyInterface
}

// NumatopologyInterface has methods to work with Numatopology resources.
type NumatopologyInterface interface {
	Create(ctx context.Context, numatopology *v1alpha1.Numatopology, opts v1.CreateOptions) (*v1alpha1.Numatopology, error)
	Update(ctx context.Context, numatopology *v1alpha1.Numatopology, opts v1.UpdateOptions) (*v1alpha1.Numatopology, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.Numatopology, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.NumatopologyList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Numatopology, err error)
	NumatopologyExpansion
}

// numatopologies implements NumatopologyInterface
type numatopologies struct {
	client rest.Interface
}

// newNumatopologies returns a Numatopologies
func newNumatopologies(c *NodeinfoV1alpha1Client) *numatopologies {
	return &numatopologies{
		client: c.RESTClient(),
	}
}

// Get takes name of the numatopology, and returns the corresponding numatopology object, and an error if there is any.
func (c *numatopologies) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Numatopology, err error) {
	result = &v1alpha1.Numatopology{}
	err = c.client.Get().
		Resource("numatopologies").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Numatopologies that match those selectors.
func (c *numatopologies) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.NumatopologyList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.NumatopologyList{}
	err = c.client.Get().
		Resource("numatopologies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested numatopologies.
func (c *numatopologies) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("numatopologies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a numatopology and creates it.  Returns the server's representation of the numatopology, and an error, if there is any.
func (c *numatopologies) Create(ctx context.Context, numatopology *v1alpha1.Numatopology, opts v1.CreateOptions) (result *v1alpha1.Numatopology, err error) {
	result = &v1alpha1.Numatopology{}
	err = c.client.Post().
		Resource("numatopologies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(numatopology).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a numatopology and updates it. Returns the server's representation of the numatopology, and an error, if there is any.
func (c *numatopologies) Update(ctx context.Context, numatopology *v1alpha1.Numatopology, opts v1.UpdateOptions) (result *v1alpha1.Numatopology, err error) {
	result = &v1alpha1.Numatopology{}
	err = c.client.Put().
		Resource("numatopologies").
		Name(numatopology.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(numatopology).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the numatopology and deletes it. Returns an error if one occurs.
func (c *numatopologies) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("numatopologies").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *numatopologies) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("numatopologies").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched numatopology.
func (c *numatopologies) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Numatopology, err error) {
	result = &v1alpha1.Numatopology{}
	err = c.client.Patch(pt).
		Resource("numatopologies").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
