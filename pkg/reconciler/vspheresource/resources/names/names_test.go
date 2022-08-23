/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package names

import (
	"strings"
	"testing"

	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNames(t *testing.T) {
	tests := []struct {
		name string
		vss  *v1alpha1.VSphereSource
		f    func(*v1alpha1.VSphereSource) string
		want string
	}{{
		name: "Deployment too long",
		vss: &v1alpha1.VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: strings.Repeat("f", 63),
			},
		},
		f:    Deployment,
		want: "fffffffffffffffffffffff105d7597f637e83cc711605ac3ea4957-adapter",
	}, {
		name: "Deployment long enough",
		vss: &v1alpha1.VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: strings.Repeat("f", 52),
			},
		},
		f:    Deployment,
		want: strings.Repeat("f", 52) + "-adapter",
	}, {
		name: "Deployment",
		vss: &v1alpha1.VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		},
		f:    Deployment,
		want: "foo-adapter",
	}, {
		name: "vspherebinding",
		vss: &v1alpha1.VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: "baz",
			},
		},
		f:    VSphereBinding,
		want: "baz-vspherebinding",
	}, {
		name: "configmap",
		vss: &v1alpha1.VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: "baz",
			},
		},
		f:    ConfigMap,
		want: "baz-configmap",
	}, {
		name: "rolebinding",
		vss: &v1alpha1.VSphereSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: "baz",
			},
		},
		f:    RoleBinding,
		want: "baz-rolebinding",
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.f(test.vss)
			if got != test.want {
				t.Errorf("%s() = %v, wanted %v", test.name, got, test.want)
			}
		})
	}
}
