/*
Copyright 2022 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"context"
)

// SetDefaults implements apis.Defaultable
func (vsb *VSphereBinding) SetDefaults(ctx context.Context) {
	if vsb.Spec.Subject.Namespace == "" {
		// Default the subject's namespace to our namespace.
		vsb.Spec.Subject.Namespace = vsb.Namespace
	}
}
