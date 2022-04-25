/*
Copyright 2022 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"context"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/webhook/resourcesemantics"

	"knative.dev/pkg/apis"
)

var (
	source, _ = apis.ParseURL("https://horizon.api.dev")

	fullSpec HorizonSourceSpec = HorizonSourceSpec{
		SourceSpec: duckv1.SourceSpec{
			Sink: duckv1.Destination{
				Ref: &duckv1.KReference{
					APIVersion: "v1",
					Kind:       "Broker",
					Namespace:  "default",
					Name:       "default",
				},
			},
		},
		ServiceAccountName: "default",
		HorizonAuthSpec: HorizonAuthSpec{
			Address:       *source,
			SkipTLSVerify: false,
			SecretRef: corev1.LocalObjectReference{
				Name: "horizon-secret",
			},
		},
	}
)

func TestHorizonSourceImmutableFields(t *testing.T) {
	testCases := map[string]struct {
		orig    *HorizonSourceSpec
		updated HorizonSourceSpec
		allowed bool
	}{
		"nil orig": {
			updated: fullSpec,
			allowed: true,
		},
		"Sink.Namespace changed": {
			orig: &fullSpec,
			updated: HorizonSourceSpec{
				SourceSpec: duckv1.SourceSpec{
					Sink: duckv1.Destination{
						Ref: &duckv1.KReference{
							APIVersion: fullSpec.Sink.Ref.APIVersion,
							Kind:       fullSpec.Sink.Ref.Kind,
							Namespace:  "changed",
							Name:       fullSpec.Sink.Ref.Name,
						},
					},
				},
				ServiceAccountName: fullSpec.ServiceAccountName,
				HorizonAuthSpec:    fullSpec.HorizonAuthSpec,
			},
			allowed: false,
		},
		"Sink.Name changed": {
			orig: &fullSpec,
			updated: HorizonSourceSpec{
				SourceSpec: duckv1.SourceSpec{
					Sink: duckv1.Destination{
						Ref: &duckv1.KReference{
							APIVersion: fullSpec.Sink.Ref.APIVersion,
							Kind:       fullSpec.Sink.Ref.Kind,
							Namespace:  fullSpec.Sink.Ref.Namespace,
							Name:       "changed",
						},
					},
				},
				ServiceAccountName: fullSpec.ServiceAccountName,
				HorizonAuthSpec:    fullSpec.HorizonAuthSpec,
			},
			allowed: false,
		},
		"Sink.ApiVersion changed": {
			orig: &fullSpec,
			updated: HorizonSourceSpec{
				SourceSpec: duckv1.SourceSpec{
					Sink: duckv1.Destination{
						Ref: &duckv1.KReference{
							APIVersion: "v1alpha1",
							Kind:       fullSpec.Sink.Ref.Kind,
							Namespace:  fullSpec.Sink.Ref.Namespace,
							Name:       fullSpec.Sink.Ref.Name,
						},
					},
				},
				ServiceAccountName: fullSpec.ServiceAccountName,
				HorizonAuthSpec:    fullSpec.HorizonAuthSpec,
			},
			allowed: false,
		},
		"ServiceAccount changed": {
			orig: &fullSpec,
			updated: HorizonSourceSpec{
				SourceSpec:         fullSpec.SourceSpec,
				ServiceAccountName: "changed",
				HorizonAuthSpec:    fullSpec.HorizonAuthSpec,
			},
			allowed: false,
		},
		"Auth.Address changed": {
			orig: &fullSpec,
			updated: HorizonSourceSpec{
				SourceSpec:         fullSpec.SourceSpec,
				ServiceAccountName: fullSpec.ServiceAccountName,
				HorizonAuthSpec: HorizonAuthSpec{
					Address: apis.URL{
						Scheme: "http",
						Host:   "changed.example.com",
					},
					SkipTLSVerify: fullSpec.SkipTLSVerify,
					SecretRef:     fullSpec.SecretRef,
				},
			},
			allowed: false,
		},
		"Auth.SkipTLSVerify changed": {
			orig: &fullSpec,
			updated: HorizonSourceSpec{
				SourceSpec:         fullSpec.SourceSpec,
				ServiceAccountName: fullSpec.ServiceAccountName,
				HorizonAuthSpec: HorizonAuthSpec{
					Address:       fullSpec.Address,
					SkipTLSVerify: true,
					SecretRef:     fullSpec.SecretRef,
				},
			},
			allowed: false,
		},
		"Auth.SecretRef changed": {
			orig: &fullSpec,
			updated: HorizonSourceSpec{
				SourceSpec:         fullSpec.SourceSpec,
				ServiceAccountName: fullSpec.ServiceAccountName,
				HorizonAuthSpec: HorizonAuthSpec{
					Address:       fullSpec.Address,
					SkipTLSVerify: fullSpec.SkipTLSVerify,
					SecretRef:     corev1.LocalObjectReference{Name: "changed"},
				},
			},
			allowed: false,
		},
	}

	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			ctx := context.TODO()
			if tc.orig != nil {
				orig := &HorizonSource{
					Spec: *tc.orig,
				}

				ctx = apis.WithinUpdate(ctx, orig)
			}
			updated := &HorizonSource{
				Spec: tc.updated,
			}
			err := updated.Validate(ctx)
			if tc.allowed != (err == nil) {
				t.Fatalf("Unexpected immutable field check. Expected %v. Actual %v", tc.allowed, err)
			}
		})
	}
}

func TestHorizonSourceValidation(t *testing.T) {
	testCases := map[string]struct {
		cr   resourcesemantics.GenericCRD
		want *apis.FieldError
	}{
		"invalid nil spec": {
			cr: &HorizonSource{
				Spec: HorizonSourceSpec{},
			},
			want: func() *apis.FieldError {
				var errs *apis.FieldError

				feSink := apis.ErrGeneric("expected at least one, got none", "ref", "uri")
				feSink = feSink.ViaField("sink").ViaField("spec")
				errs = errs.Also(feSink)

				feAddress := apis.ErrMissingField("address.host")
				feAddress = feAddress.ViaField("spec")
				errs = errs.Also(feAddress)

				secretRef := apis.ErrMissingField("secretRef.name")
				secretRef = secretRef.ViaField("spec")
				errs = errs.Also(secretRef)

				feServiceAccountName := apis.ErrMissingField("serviceAccountName")
				feServiceAccountName = feServiceAccountName.ViaField("spec")
				errs = errs.Also(feServiceAccountName)

				return errs
			}(),
		},
		"secret missing": {
			cr: &HorizonSource{
				Spec: HorizonSourceSpec{
					SourceSpec: duckv1.SourceSpec{
						Sink: newDestination(),
					},
					ServiceAccountName: "default",
					HorizonAuthSpec: HorizonAuthSpec{
						Address: newHorizonAddress(),
					},
				},
			},
			want: func() *apis.FieldError {
				var errs *apis.FieldError

				secretRef := apis.ErrMissingField("secretRef.name")
				secretRef = secretRef.ViaField("spec")
				errs = errs.Also(secretRef)

				return errs
			}(),
		},
		"horizon source address missing": {
			cr: &HorizonSource{
				Spec: HorizonSourceSpec{
					SourceSpec: duckv1.SourceSpec{
						Sink: newDestination(),
					},
					ServiceAccountName: "default",
					HorizonAuthSpec: HorizonAuthSpec{
						SecretRef: newSecretRef(),
					},
				},
			},
			want: func() *apis.FieldError {
				var errs *apis.FieldError

				secretRef := apis.ErrMissingField("address.host")
				secretRef = secretRef.ViaField("spec")
				errs = errs.Also(secretRef)

				return errs
			}(),
		},
		"valid spec": {
			cr: &HorizonSource{
				Spec: HorizonSourceSpec{
					SourceSpec: duckv1.SourceSpec{
						Sink: newDestination(),
					},
					ServiceAccountName: "default",
					HorizonAuthSpec: HorizonAuthSpec{
						Address:   newHorizonAddress(),
						SecretRef: newSecretRef(),
					},
				},
			},
			want: func() *apis.FieldError {
				return nil
			}(),
		},
	}

	for n, test := range testCases {
		t.Run(n, func(t *testing.T) {
			got := test.cr.Validate(context.Background())
			if diff := cmp.Diff(test.want.Error(), got.Error()); diff != "" {
				t.Errorf("%s: validate (-want, +got) = %v", n, diff)
			}
		})
	}
}

func newSecretRef() corev1.LocalObjectReference {
	return corev1.LocalObjectReference{
		Name: "horizon-creds",
	}
}

func newDestination() duckv1.Destination {
	return duckv1.Destination{
		Ref: &duckv1.KReference{
			Kind:       "Deployment",
			Namespace:  "default",
			Name:       "receiver",
			APIVersion: "v1",
		},
	}
}

func newHorizonAddress() apis.URL {
	u, _ := url.Parse("http://api.horizon.corp.local")
	return apis.URL(*u)
}
