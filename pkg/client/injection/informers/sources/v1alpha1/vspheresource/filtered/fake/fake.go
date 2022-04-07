/*
Copyright 2022 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by injection-gen. DO NOT EDIT.

package fake

import (
	context "context"

	factoryfiltered "github.com/vmware-tanzu/sources-for-knative/pkg/client/injection/informers/factory/filtered"
	filtered "github.com/vmware-tanzu/sources-for-knative/pkg/client/injection/informers/sources/v1alpha1/vspheresource/filtered"
	controller "knative.dev/pkg/controller"
	injection "knative.dev/pkg/injection"
	logging "knative.dev/pkg/logging"
)

var Get = filtered.Get

func init() {
	injection.Fake.RegisterFilteredInformers(withInformer)
}

func withInformer(ctx context.Context) (context.Context, []controller.Informer) {
	untyped := ctx.Value(factoryfiltered.LabelKey{})
	if untyped == nil {
		logging.FromContext(ctx).Panic(
			"Unable to fetch labelkey from context.")
	}
	labelSelectors := untyped.([]string)
	infs := []controller.Informer{}
	for _, selector := range labelSelectors {
		f := factoryfiltered.Get(ctx, selector)
		inf := f.Sources().V1alpha1().VSphereSources()
		ctx = context.WithValue(ctx, filtered.Key{Selector: selector}, inf)
		infs = append(infs, inf.Informer())
	}
	return ctx, infs
}
