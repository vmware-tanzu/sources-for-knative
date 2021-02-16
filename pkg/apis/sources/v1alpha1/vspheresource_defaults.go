/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"context"

	"knative.dev/pkg/apis"

	"github.com/vmware-tanzu/sources-for-knative/pkg/vsphere"
)

// SetDefaults implements apis.Defaultable
func (vs *VSphereSource) SetDefaults(ctx context.Context) {
	withNS := apis.WithinParent(ctx, vs.ObjectMeta)
	vs.Spec.Sink.SetDefaults(withNS)

	// only checking period, setting maxAge to 0 will disable event replay
	// to get at-most-once semantics
	if vs.Spec.CheckpointConfig.PeriodSeconds == 0 {
		vs.Spec.CheckpointConfig.PeriodSeconds = int64(vsphere.CheckpointDefaultPeriod.Seconds())
	}
}
