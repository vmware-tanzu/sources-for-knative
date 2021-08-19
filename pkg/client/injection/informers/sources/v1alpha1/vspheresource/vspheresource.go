/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by injection-gen. DO NOT EDIT.

package vspheresource

import (
	context "context"

	apissourcesv1alpha1 "github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	versioned "github.com/vmware-tanzu/sources-for-knative/pkg/client/clientset/versioned"
	v1alpha1 "github.com/vmware-tanzu/sources-for-knative/pkg/client/informers/externalversions/sources/v1alpha1"
	client "github.com/vmware-tanzu/sources-for-knative/pkg/client/injection/client"
	factory "github.com/vmware-tanzu/sources-for-knative/pkg/client/injection/informers/factory"
	sourcesv1alpha1 "github.com/vmware-tanzu/sources-for-knative/pkg/client/listers/sources/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	cache "k8s.io/client-go/tools/cache"
	controller "knative.dev/pkg/controller"
	injection "knative.dev/pkg/injection"
	logging "knative.dev/pkg/logging"
)

func init() {
	injection.Default.RegisterInformer(withInformer)
	injection.Dynamic.RegisterDynamicInformer(withDynamicInformer)
}

// Key is used for associating the Informer inside the context.Context.
type Key struct{}

func withInformer(ctx context.Context) (context.Context, controller.Informer) {
	f := factory.Get(ctx)
	inf := f.Sources().V1alpha1().VSphereSources()
	return context.WithValue(ctx, Key{}, inf), inf.Informer()
}

func withDynamicInformer(ctx context.Context) context.Context {
	inf := &wrapper{client: client.Get(ctx)}
	return context.WithValue(ctx, Key{}, inf)
}

// Get extracts the typed informer from the context.
func Get(ctx context.Context) v1alpha1.VSphereSourceInformer {
	untyped := ctx.Value(Key{})
	if untyped == nil {
		logging.FromContext(ctx).Panic(
			"Unable to fetch github.com/vmware-tanzu/sources-for-knative/pkg/client/informers/externalversions/sources/v1alpha1.VSphereSourceInformer from context.")
	}
	return untyped.(v1alpha1.VSphereSourceInformer)
}

type wrapper struct {
	client versioned.Interface

	namespace string
}

var _ v1alpha1.VSphereSourceInformer = (*wrapper)(nil)
var _ sourcesv1alpha1.VSphereSourceLister = (*wrapper)(nil)

func (w *wrapper) Informer() cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(nil, &apissourcesv1alpha1.VSphereSource{}, 0, nil)
}

func (w *wrapper) Lister() sourcesv1alpha1.VSphereSourceLister {
	return w
}

func (w *wrapper) VSphereSources(namespace string) sourcesv1alpha1.VSphereSourceNamespaceLister {
	return &wrapper{client: w.client, namespace: namespace}
}

func (w *wrapper) List(selector labels.Selector) (ret []*apissourcesv1alpha1.VSphereSource, err error) {
	lo, err := w.client.SourcesV1alpha1().VSphereSources(w.namespace).List(context.TODO(), v1.ListOptions{
		LabelSelector: selector.String(),
		// TODO(mattmoor): Incorporate resourceVersion bounds based on staleness criteria.
	})
	if err != nil {
		return nil, err
	}
	for idx := range lo.Items {
		ret = append(ret, &lo.Items[idx])
	}
	return ret, nil
}

func (w *wrapper) Get(name string) (*apissourcesv1alpha1.VSphereSource, error) {
	return w.client.SourcesV1alpha1().VSphereSources(w.namespace).Get(context.TODO(), name, v1.GetOptions{
		// TODO(mattmoor): Incorporate resourceVersion bounds based on staleness criteria.
	})
}
