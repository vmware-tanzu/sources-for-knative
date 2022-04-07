/*
Copyright 2022 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"context"
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/google/go-cmp/cmp"
)

func TestHorizonSourceDefaults(t *testing.T) {
	testCases := map[string]struct {
		initial  HorizonSource
		expected HorizonSource
	}{
		"nil spec": {
			initial: HorizonSource{},
			expected: HorizonSource{
				Spec: HorizonSourceSpec{
					ServiceAccountName: "default",
				},
			},
		},
		"no namespace in sink reference": {
			initial: HorizonSource{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "parent",
				},
				Spec: HorizonSourceSpec{
					ServiceAccountName: "default",
					SourceSpec: duckv1.SourceSpec{
						Sink: duckv1.Destination{
							Ref: &duckv1.KReference{},
						},
					},
				},
			},
			expected: HorizonSource{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "parent",
				},
				Spec: HorizonSourceSpec{
					ServiceAccountName: "default",
					SourceSpec: duckv1.SourceSpec{
						Sink: duckv1.Destination{
							Ref: &duckv1.KReference{
								Namespace: "parent",
							},
						},
					},
				},
			},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			tc.initial.SetDefaults(context.TODO())
			if diff := cmp.Diff(tc.expected, tc.initial); diff != "" {
				t.Fatalf("Unexpected defaults (-want, +got): %s", diff)
			}
		})
	}
}
