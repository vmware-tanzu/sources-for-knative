/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by injection-gen. DO NOT EDIT.

package horizonsource

import (
	context "context"

	v1alpha1 "github.com/vmware-tanzu/sources-for-knative/pkg/client/informers/externalversions/sources/v1alpha1"
	factory "github.com/vmware-tanzu/sources-for-knative/pkg/client/injection/informers/factory"
	controller "knative.dev/pkg/controller"
	injection "knative.dev/pkg/injection"
	logging "knative.dev/pkg/logging"
)

func init() {
	injection.Default.RegisterInformer(withInformer)
}

// Key is used for associating the Informer inside the context.Context.
type Key struct{}

func withInformer(ctx context.Context) (context.Context, controller.Informer) {
	f := factory.Get(ctx)
	inf := f.Sources().V1alpha1().HorizonSources()
	return context.WithValue(ctx, Key{}, inf), inf.Informer()
}

// Get extracts the typed informer from the context.
func Get(ctx context.Context) v1alpha1.HorizonSourceInformer {
	untyped := ctx.Value(Key{})
	if untyped == nil {
		logging.FromContext(ctx).Panic(
			"Unable to fetch github.com/vmware-tanzu/sources-for-knative/pkg/client/informers/externalversions/sources/v1alpha1.HorizonSourceInformer from context.")
	}
	return untyped.(v1alpha1.HorizonSourceInformer)
}
