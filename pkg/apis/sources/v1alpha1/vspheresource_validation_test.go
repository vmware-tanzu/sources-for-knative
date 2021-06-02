/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"context"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	validSourceSpec = duckv1.SourceSpec{
		Sink: duckv1.Destination{
			URI: &apis.URL{
				Scheme: "https",
				Host:   "knative.dev",
			},
		},
	}
)

func TestVSphereSourceValidation(t *testing.T) {
	tests := []struct {
		name string
		c    *VSphereSource
		want *apis.FieldError
	}{{
		name: "valid",
		c: &VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: VSphereSourceSpec{
				SourceSpec:      validSourceSpec,
				VAuthSpec:       validVAuthSpec,
				PayloadEncoding: cloudevents.ApplicationXML,
			},
		},
		want: nil,
	}, {
		name: "valid with JSON payloadEncoding",
		c: &VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: VSphereSourceSpec{
				SourceSpec:      validSourceSpec,
				VAuthSpec:       validVAuthSpec,
				PayloadEncoding: cloudevents.ApplicationJSON,
			},
		},
		want: nil,
	}, {
		name: "invalid payloadEncoding",
		c: &VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: VSphereSourceSpec{
				SourceSpec:      validSourceSpec,
				VAuthSpec:       validVAuthSpec,
				PayloadEncoding: "application/text",
			},
		},
		want: apis.ErrInvalidValue("application/text", "spec.payloadEncoding"),
	}, {
		name: "missing VAuthSpec",
		c: &VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: VSphereSourceSpec{
				SourceSpec: validSourceSpec,
				// VAuthSpec:  validVAuthSpec,
				PayloadEncoding: cloudevents.ApplicationXML,
			},
		},
		want: apis.ErrMissingField("spec.address.host", "spec.secretRef.name"),
	}, {
		name: "missing SourceSpec",
		c: &VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: VSphereSourceSpec{
				// SourceSpec: validSourceSpec,
				VAuthSpec:       validVAuthSpec,
				PayloadEncoding: cloudevents.ApplicationXML,
			},
		},
		want: apis.ErrGeneric("expected at least one, got none", "spec.sink.ref", "spec.sink.uri"),
	}, {
		name: "invalid CheckpointConfig",
		c: &VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: VSphereSourceSpec{
				SourceSpec: validSourceSpec,
				VAuthSpec:  validVAuthSpec,
				CheckpointConfig: VCheckpointSpec{
					MaxAgeSeconds: -10,
					PeriodSeconds: -5,
				},
				PayloadEncoding: cloudevents.ApplicationXML,
			},
		},
		want: apis.ErrInvalidValue("-10", "spec.checkpointConfig.maxAgeSeconds").Also(apis.ErrInvalidValue("-5",
			"spec.checkpointConfig.periodSeconds")),
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.c.Validate(context.Background())
			if !cmp.Equal(test.want.Error(), got.Error()) {
				t.Errorf("Validate (-want, +got) = %v",
					cmp.Diff(test.want.Error(), got.Error()))
			}
		})
	}
}
