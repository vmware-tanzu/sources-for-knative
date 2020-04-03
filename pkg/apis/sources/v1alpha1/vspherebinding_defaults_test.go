/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/tracker"
)

func TestVSphereBindingDefaulting(t *testing.T) {
	tests := []struct {
		name string
		c    *VSphereBinding
		want *VSphereBinding
	}{{
		name: "no change",
		c: &VSphereBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: VSphereBindingSpec{
				BindingSpec: validBindingSpec,
				VAuthSpec:   validVAuthSpec,
			},
		},
		want: &VSphereBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: VSphereBindingSpec{
				BindingSpec: validBindingSpec,
				VAuthSpec:   validVAuthSpec,
			},
		},
	}, {
		name: "binding gets namespace",
		c: &VSphereBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "valid",
				Namespace: "with-namespace",
			},
			Spec: VSphereBindingSpec{
				BindingSpec: duckv1alpha1.BindingSpec{
					Subject: tracker.Reference{
						APIVersion: "serving.knative.dev",
						Kind:       "Service",
						Name:       "no-namespace",
					},
				},
				VAuthSpec: validVAuthSpec,
			},
		},
		want: &VSphereBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "valid",
				Namespace: "with-namespace",
			},
			Spec: VSphereBindingSpec{
				BindingSpec: duckv1alpha1.BindingSpec{
					Subject: tracker.Reference{
						APIVersion: "serving.knative.dev",
						Kind:       "Service",
						Name:       "no-namespace",
						Namespace:  "with-namespace",
					},
				},
				VAuthSpec: validVAuthSpec,
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.c.DeepCopy()
			got.SetDefaults(context.Background())
			if !cmp.Equal(test.want, got) {
				t.Errorf("SetDefaults (-want, +got) = %v", cmp.Diff(test.want, got))
			}
		})
	}
}
