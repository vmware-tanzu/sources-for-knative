/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"time"

	v1alpha1 "github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	scheme "github.com/vmware-tanzu/sources-for-knative/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// VSphereBindingsGetter has a method to return a VSphereBindingInterface.
// A group's client should implement this interface.
type VSphereBindingsGetter interface {
	VSphereBindings(namespace string) VSphereBindingInterface
}

// VSphereBindingInterface has methods to work with VSphereBinding resources.
type VSphereBindingInterface interface {
	Create(*v1alpha1.VSphereBinding) (*v1alpha1.VSphereBinding, error)
	Update(*v1alpha1.VSphereBinding) (*v1alpha1.VSphereBinding, error)
	UpdateStatus(*v1alpha1.VSphereBinding) (*v1alpha1.VSphereBinding, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.VSphereBinding, error)
	List(opts v1.ListOptions) (*v1alpha1.VSphereBindingList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.VSphereBinding, err error)
	VSphereBindingExpansion
}

// vSphereBindings implements VSphereBindingInterface
type vSphereBindings struct {
	client rest.Interface
	ns     string
}

// newVSphereBindings returns a VSphereBindings
func newVSphereBindings(c *SourcesV1alpha1Client, namespace string) *vSphereBindings {
	return &vSphereBindings{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the vSphereBinding, and returns the corresponding vSphereBinding object, and an error if there is any.
func (c *vSphereBindings) Get(name string, options v1.GetOptions) (result *v1alpha1.VSphereBinding, err error) {
	result = &v1alpha1.VSphereBinding{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("vspherebindings").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of VSphereBindings that match those selectors.
func (c *vSphereBindings) List(opts v1.ListOptions) (result *v1alpha1.VSphereBindingList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.VSphereBindingList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("vspherebindings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested vSphereBindings.
func (c *vSphereBindings) Watch(opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("vspherebindings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a vSphereBinding and creates it.  Returns the server's representation of the vSphereBinding, and an error, if there is any.
func (c *vSphereBindings) Create(vSphereBinding *v1alpha1.VSphereBinding) (result *v1alpha1.VSphereBinding, err error) {
	result = &v1alpha1.VSphereBinding{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("vspherebindings").
		Body(vSphereBinding).
		Do().
		Into(result)
	return
}

// Update takes the representation of a vSphereBinding and updates it. Returns the server's representation of the vSphereBinding, and an error, if there is any.
func (c *vSphereBindings) Update(vSphereBinding *v1alpha1.VSphereBinding) (result *v1alpha1.VSphereBinding, err error) {
	result = &v1alpha1.VSphereBinding{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("vspherebindings").
		Name(vSphereBinding.Name).
		Body(vSphereBinding).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *vSphereBindings) UpdateStatus(vSphereBinding *v1alpha1.VSphereBinding) (result *v1alpha1.VSphereBinding, err error) {
	result = &v1alpha1.VSphereBinding{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("vspherebindings").
		Name(vSphereBinding.Name).
		SubResource("status").
		Body(vSphereBinding).
		Do().
		Into(result)
	return
}

// Delete takes name of the vSphereBinding and deletes it. Returns an error if one occurs.
func (c *vSphereBindings) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("vspherebindings").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *vSphereBindings) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("vspherebindings").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched vSphereBinding.
func (c *vSphereBindings) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.VSphereBinding, err error) {
	result = &v1alpha1.VSphereBinding{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("vspherebindings").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
