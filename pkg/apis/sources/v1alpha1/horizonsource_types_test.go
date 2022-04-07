/*
Copyright 2022 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"testing"
)

func TestHorizonSource_GetGroupVersionKind(t *testing.T) {
	src := HorizonSource{}
	gvk := src.GetGroupVersionKind()

	if gvk.Kind != "HorizonSource" {
		t.Errorf("Should be 'HorizonSource'.")
	}
}
