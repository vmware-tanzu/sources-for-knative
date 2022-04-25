/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestRegisterHelpers(t *testing.T) {
	if got, want := Kind("VsphereSource"), "VsphereSource.sources.tanzu.vmware.com"; got.String() != want {
		t.Errorf("Kind(VsphereSource) = %v, want %v", got.String(), want)
	}

	if got, want := Resource("VsphereSource"), "VsphereSource.sources.tanzu.vmware.com"; got.String() != want {
		t.Errorf("Resource(VsphereSource) = %v, want %v", got.String(), want)
	}

	if got, want := SchemeGroupVersion.String(), "sources.tanzu.vmware.com/v1alpha1"; got != want {
		t.Errorf("SchemeGroupVersion() = %v, want %v", got, want)
	}

	if got, want := Kind("HorizonSource"), "HorizonSource.sources.tanzu.vmware.com"; got.String() != want {
		t.Errorf("Kind(HorizonSource) = %v, want %v", got.String(), want)
	}

	if got, want := Resource("HorizonSource"), "HorizonSource.sources.tanzu.vmware.com"; got.String() != want {
		t.Errorf("Resource(HorizonSource) = %v, want %v", got.String(), want)
	}

	scheme := runtime.NewScheme()
	if err := addKnownTypes(scheme); err != nil {
		t.Errorf("addKnownTypes() = %v", err)
	}
}
