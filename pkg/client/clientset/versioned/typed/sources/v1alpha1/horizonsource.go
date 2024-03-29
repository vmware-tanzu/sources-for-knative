/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	scheme "github.com/vmware-tanzu/sources-for-knative/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// HorizonSourcesGetter has a method to return a HorizonSourceInterface.
// A group's client should implement this interface.
type HorizonSourcesGetter interface {
	HorizonSources(namespace string) HorizonSourceInterface
}

// HorizonSourceInterface has methods to work with HorizonSource resources.
type HorizonSourceInterface interface {
	Create(ctx context.Context, horizonSource *v1alpha1.HorizonSource, opts v1.CreateOptions) (*v1alpha1.HorizonSource, error)
	Update(ctx context.Context, horizonSource *v1alpha1.HorizonSource, opts v1.UpdateOptions) (*v1alpha1.HorizonSource, error)
	UpdateStatus(ctx context.Context, horizonSource *v1alpha1.HorizonSource, opts v1.UpdateOptions) (*v1alpha1.HorizonSource, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.HorizonSource, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.HorizonSourceList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.HorizonSource, err error)
	HorizonSourceExpansion
}

// horizonSources implements HorizonSourceInterface
type horizonSources struct {
	client rest.Interface
	ns     string
}

// newHorizonSources returns a HorizonSources
func newHorizonSources(c *SourcesV1alpha1Client, namespace string) *horizonSources {
	return &horizonSources{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the horizonSource, and returns the corresponding horizonSource object, and an error if there is any.
func (c *horizonSources) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.HorizonSource, err error) {
	result = &v1alpha1.HorizonSource{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("horizonsources").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of HorizonSources that match those selectors.
func (c *horizonSources) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.HorizonSourceList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.HorizonSourceList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("horizonsources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested horizonSources.
func (c *horizonSources) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("horizonsources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a horizonSource and creates it.  Returns the server's representation of the horizonSource, and an error, if there is any.
func (c *horizonSources) Create(ctx context.Context, horizonSource *v1alpha1.HorizonSource, opts v1.CreateOptions) (result *v1alpha1.HorizonSource, err error) {
	result = &v1alpha1.HorizonSource{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("horizonsources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(horizonSource).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a horizonSource and updates it. Returns the server's representation of the horizonSource, and an error, if there is any.
func (c *horizonSources) Update(ctx context.Context, horizonSource *v1alpha1.HorizonSource, opts v1.UpdateOptions) (result *v1alpha1.HorizonSource, err error) {
	result = &v1alpha1.HorizonSource{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("horizonsources").
		Name(horizonSource.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(horizonSource).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *horizonSources) UpdateStatus(ctx context.Context, horizonSource *v1alpha1.HorizonSource, opts v1.UpdateOptions) (result *v1alpha1.HorizonSource, err error) {
	result = &v1alpha1.HorizonSource{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("horizonsources").
		Name(horizonSource.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(horizonSource).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the horizonSource and deletes it. Returns an error if one occurs.
func (c *horizonSources) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("horizonsources").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *horizonSources) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("horizonsources").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched horizonSource.
func (c *horizonSources) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.HorizonSource, err error) {
	result = &v1alpha1.HorizonSource{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("horizonsources").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
