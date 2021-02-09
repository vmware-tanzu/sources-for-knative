/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"context"

	"knative.dev/pkg/apis"
)

// Validate implements apis.Validatable
func (vs *VSphereSource) Validate(ctx context.Context) *apis.FieldError {
	return vs.Spec.Validate(ctx).ViaField("spec")
}

// Validate implements apis.Validatable
func (vsss *VSphereSourceSpec) Validate(ctx context.Context) *apis.FieldError {
	return vsss.Sink.Validate(ctx).ViaField("sink").Also(vsss.VAuthSpec.Validate(ctx)).Also(vsss.CheckpointConfig.
		Validate(ctx))
}

func (vcs VCheckpointSpec) Validate(ctx context.Context) (err *apis.FieldError) {
	if vcs.PeriodSeconds < 0 {
		err = err.Also(apis.ErrInvalidValue(vcs.PeriodSeconds, "checkpointConfig.periodSeconds"))
	}

	if vcs.MaxAgeSeconds < 0 {
		err = err.Also(apis.ErrInvalidValue(vcs.MaxAgeSeconds, "checkpointConfig.maxAgeSeconds"))
	}

	return err
}
