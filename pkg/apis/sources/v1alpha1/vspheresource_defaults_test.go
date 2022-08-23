/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"context"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/vmware-tanzu/sources-for-knative/pkg/vsphere"
)

func TestVSphereSourceDefaulting(t *testing.T) {
	tests := []struct {
		name string
		c    *VSphereSource
		want *VSphereSource
	}{{
		name: "CheckpointConfig and PayloadEncoding not set",
		c: &VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: VSphereSourceSpec{
				SourceSpec: validSourceSpec,
				VAuthSpec:  validVAuthSpec,
			},
		},
		want: &VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: VSphereSourceSpec{
				SourceSpec: validSourceSpec,
				VAuthSpec:  validVAuthSpec,
				CheckpointConfig: VCheckpointSpec{
					MaxAgeSeconds: 0,
					PeriodSeconds: int64(vsphere.CheckpointDefaultPeriod.Seconds()),
				},
				PayloadEncoding:    cloudevents.ApplicationXML,
				ServiceAccountName: "default",
			},
		},
	}, {
		name: "payloadEncoding set to JSON",
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
		want: &VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: VSphereSourceSpec{
				SourceSpec: validSourceSpec,
				VAuthSpec:  validVAuthSpec,
				CheckpointConfig: VCheckpointSpec{
					MaxAgeSeconds: 0,
					PeriodSeconds: int64(vsphere.CheckpointDefaultPeriod.Seconds()),
				},
				PayloadEncoding:    cloudevents.ApplicationJSON,
				ServiceAccountName: "default",
			},
		},
	}, {
		name: "ref gets namespace",
		c: &VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "valid",
				Namespace: "with-namespace",
			},
			Spec: VSphereSourceSpec{
				SourceSpec: duckv1.SourceSpec{
					Sink: duckv1.Destination{
						Ref: &duckv1.KReference{
							APIVersion: "serving.knative.dev",
							Kind:       "Service",
							Name:       "no-namespace",
						},
					},
				},
				VAuthSpec: validVAuthSpec,
			},
		},
		want: &VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "valid",
				Namespace: "with-namespace",
			},
			Spec: VSphereSourceSpec{
				SourceSpec: duckv1.SourceSpec{
					Sink: duckv1.Destination{
						Ref: &duckv1.KReference{
							APIVersion: "serving.knative.dev",
							Kind:       "Service",
							Namespace:  "with-namespace",
							Name:       "no-namespace",
						},
					},
				},
				VAuthSpec: validVAuthSpec,
				CheckpointConfig: VCheckpointSpec{
					MaxAgeSeconds: 0,
					PeriodSeconds: int64(vsphere.CheckpointDefaultPeriod.Seconds()),
				},
				PayloadEncoding:    cloudevents.ApplicationXML,
				ServiceAccountName: "default",
			},
		},
	}, {
		name: "custom checkpoint config",
		c: &VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: VSphereSourceSpec{
				SourceSpec: validSourceSpec,
				VAuthSpec:  validVAuthSpec,
				CheckpointConfig: VCheckpointSpec{
					MaxAgeSeconds: 3600,
					PeriodSeconds: 60,
				},
			},
		},
		want: &VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: VSphereSourceSpec{
				SourceSpec: validSourceSpec,
				VAuthSpec:  validVAuthSpec,
				CheckpointConfig: VCheckpointSpec{
					MaxAgeSeconds: 3600,
					PeriodSeconds: 60,
				},
				PayloadEncoding:    cloudevents.ApplicationXML,
				ServiceAccountName: "default",
			},
		},
	}, {
		name: "custom serviceAccountName",
		c: &VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: VSphereSourceSpec{
				SourceSpec:         validSourceSpec,
				VAuthSpec:          validVAuthSpec,
				ServiceAccountName: "test-svcacc",
			},
		},
		want: &VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: VSphereSourceSpec{
				SourceSpec: validSourceSpec,
				VAuthSpec:  validVAuthSpec,
				CheckpointConfig: VCheckpointSpec{
					MaxAgeSeconds: 0,
					PeriodSeconds: int64(vsphere.CheckpointDefaultPeriod.Seconds()),
				},
				PayloadEncoding:    cloudevents.ApplicationXML,
				ServiceAccountName: "test-svcacc",
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
