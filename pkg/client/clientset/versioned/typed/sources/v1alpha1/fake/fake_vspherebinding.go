/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeVSphereBindings implements VSphereBindingInterface
type FakeVSphereBindings struct {
	Fake *FakeSourcesV1alpha1
	ns   string
}

var vspherebindingsResource = v1alpha1.SchemeGroupVersion.WithResource("vspherebindings")

var vspherebindingsKind = v1alpha1.SchemeGroupVersion.WithKind("VSphereBinding")

// Get takes name of the vSphereBinding, and returns the corresponding vSphereBinding object, and an error if there is any.
func (c *FakeVSphereBindings) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.VSphereBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(vspherebindingsResource, c.ns, name), &v1alpha1.VSphereBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VSphereBinding), err
}

// List takes label and field selectors, and returns the list of VSphereBindings that match those selectors.
func (c *FakeVSphereBindings) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.VSphereBindingList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(vspherebindingsResource, vspherebindingsKind, c.ns, opts), &v1alpha1.VSphereBindingList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.VSphereBindingList{ListMeta: obj.(*v1alpha1.VSphereBindingList).ListMeta}
	for _, item := range obj.(*v1alpha1.VSphereBindingList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested vSphereBindings.
func (c *FakeVSphereBindings) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(vspherebindingsResource, c.ns, opts))

}

// Create takes the representation of a vSphereBinding and creates it.  Returns the server's representation of the vSphereBinding, and an error, if there is any.
func (c *FakeVSphereBindings) Create(ctx context.Context, vSphereBinding *v1alpha1.VSphereBinding, opts v1.CreateOptions) (result *v1alpha1.VSphereBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(vspherebindingsResource, c.ns, vSphereBinding), &v1alpha1.VSphereBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VSphereBinding), err
}

// Update takes the representation of a vSphereBinding and updates it. Returns the server's representation of the vSphereBinding, and an error, if there is any.
func (c *FakeVSphereBindings) Update(ctx context.Context, vSphereBinding *v1alpha1.VSphereBinding, opts v1.UpdateOptions) (result *v1alpha1.VSphereBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(vspherebindingsResource, c.ns, vSphereBinding), &v1alpha1.VSphereBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VSphereBinding), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeVSphereBindings) UpdateStatus(ctx context.Context, vSphereBinding *v1alpha1.VSphereBinding, opts v1.UpdateOptions) (*v1alpha1.VSphereBinding, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(vspherebindingsResource, "status", c.ns, vSphereBinding), &v1alpha1.VSphereBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VSphereBinding), err
}

// Delete takes name of the vSphereBinding and deletes it. Returns an error if one occurs.
func (c *FakeVSphereBindings) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(vspherebindingsResource, c.ns, name, opts), &v1alpha1.VSphereBinding{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeVSphereBindings) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(vspherebindingsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.VSphereBindingList{})
	return err
}

// Patch applies the patch and returns the patched vSphereBinding.
func (c *FakeVSphereBindings) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.VSphereBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(vspherebindingsResource, c.ns, name, pt, data, subresources...), &v1alpha1.VSphereBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VSphereBinding), err
}
