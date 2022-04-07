/*
Copyright 2022 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"context"

	"knative.dev/pkg/apis"
)

// SetDefaults mutates HorizonSource.
func (hs *HorizonSource) SetDefaults(ctx context.Context) {
	if hs != nil && hs.Spec.ServiceAccountName == "" {
		hs.Spec.ServiceAccountName = "default"
	}

	// call SetDefaults against duckv1.Destination with a context of ObjectMeta of HorizonSource.
	withNS := apis.WithinParent(ctx, hs.ObjectMeta)
	hs.Spec.Sink.SetDefaults(withNS)
}
