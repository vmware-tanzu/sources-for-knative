/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	// The set of controllers this controller process runs.
	"github.com/vmware-tanzu/sources-for-knative/pkg/reconciler/vspheresource"

	// This defines the shared main for injected controllers.
	"knative.dev/pkg/injection/sharedmain"
)

func main() {
	sharedmain.Main("controller",
		vspheresource.NewController,
	)
}
