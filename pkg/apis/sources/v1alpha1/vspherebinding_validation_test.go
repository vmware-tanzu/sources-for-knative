/*
Copyright 2022 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"context"
	"testing"

	"knative.dev/pkg/apis"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/tracker"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	validBindingSpec = duckv1alpha1.BindingSpec{
		Subject: tracker.Reference{
			APIVersion: "serving.knative.dev",
			Kind:       "Service",
			Namespace:  "knobots",
			Name:       "typo-bot",
		},
	}
	validVAuthSpec = VAuthSpec{
		Address: apis.URL{
			Scheme: "https",
			Host:   "tekton.dev",
			Path:   "/sdk",
		},
		SecretRef: corev1.LocalObjectReference{
			Name: "super-duper-secret",
		},
	}
)

func TestVSphereBindingValidation(t *testing.T) {
	tests := []struct {
		name string
		c    *VSphereBinding
		want *apis.FieldError
	}{{
		name: "valid",
		c: &VSphereBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "valid",
				Namespace: validBindingSpec.Subject.Namespace,
			},
			Spec: VSphereBindingSpec{
				BindingSpec: validBindingSpec,
				VAuthSpec:   validVAuthSpec,
			},
		},
		want: nil,
	}, {
		name: "missing BindingSpec",
		c: &VSphereBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "valid",
				Namespace: validBindingSpec.Subject.Namespace,
			},
			Spec: VSphereBindingSpec{
				// This is invalid because Namespace doesn't match.
				BindingSpec: duckv1alpha1.BindingSpec{
					Subject: tracker.Reference{
						APIVersion: "serving.knative.dev",
						Kind:       "Service",
						Namespace:  "different-namespace",
						Name:       "typo-bot",
					},
				},
				VAuthSpec: validVAuthSpec,
			},
		},
		want: apis.ErrInvalidValue("different-namespace", "spec.subject.namespace"),
	}, {
		name: "missing SecretRef",
		c: &VSphereBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "valid",
				Namespace: validBindingSpec.Subject.Namespace,
			},
			Spec: VSphereBindingSpec{
				BindingSpec: validBindingSpec,
				VAuthSpec: VAuthSpec{
					Address:   validVAuthSpec.Address,
					SecretRef: corev1.LocalObjectReference{
						// Name: "super-duper-secret",
					},
				},
			},
		},
		want: apis.ErrMissingField("spec.secretRef.name"),
	}, {
		name: "missing host address",
		c: &VSphereBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "valid",
				Namespace: validBindingSpec.Subject.Namespace,
			},
			Spec: VSphereBindingSpec{
				BindingSpec: validBindingSpec,
				VAuthSpec: VAuthSpec{
					Address: apis.URL{
						Scheme: "http",
						Path:   "/sdk",
					},
					SecretRef: validVAuthSpec.SecretRef,
				},
			},
		},
		want: apis.ErrMissingField("spec.address.host"),
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
