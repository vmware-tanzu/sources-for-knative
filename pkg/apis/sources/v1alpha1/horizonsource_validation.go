/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"context"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmp"
)

// Validate validates HorizonSource.
func (src *HorizonSource) Validate(ctx context.Context) *apis.FieldError {
	var errs *apis.FieldError

	if apis.IsInUpdate(ctx) {
		original := apis.GetBaseline(ctx).(*HorizonSource)

		// all fields immutable
		if diff, err := kmp.ShortDiff(original.Spec, src.Spec); err != nil {
			return &apis.FieldError{
				Message: "Failed to diff HorizonSource",
				Paths:   []string{"spec"},
				Details: err.Error(),
			}
		} else if diff != "" {
			return &apis.FieldError{
				Message: "Immutable fields changed (-old +new)",
				Paths:   []string{"spec"},
				Details: diff,
			}
		}
	}

	errs = errs.Also(src.Spec.Validate(ctx).ViaField("spec"))
	return errs
}

// Validate validates HorizonSourceSpec.
func (spec *HorizonSourceSpec) Validate(ctx context.Context) *apis.FieldError {
	var errs *apis.FieldError

	errs = spec.Sink.Validate(ctx).ViaField("sink").
		Also(spec.HorizonAuthSpec.Validate(ctx))

	if spec.ServiceAccountName == "" {
		errs = errs.Also(apis.ErrMissingField("serviceAccountName"))
	}

	return errs
}

// Validate implements apis.Validatable
func (auth *HorizonAuthSpec) Validate(ctx context.Context) (err *apis.FieldError) {
	if auth.Address.Host == "" {
		err = err.Also(apis.ErrMissingField("address.host"))
	}
	if auth.SecretRef.Name == "" {
		err = err.Also(apis.ErrMissingField("secretRef.name"))
	}
	return err
}
