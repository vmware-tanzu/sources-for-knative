/*
Copyright 2022 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	// The set of controllers this controller process runs.
	"github.com/vmware-tanzu/sources-for-knative/pkg/reconciler/horizonsource"

	// This defines the shared main for injected controllers.
	"knative.dev/pkg/injection/sharedmain"
)

const (
	controllerName = "horizon-source-controller"
)

func main() {
	sharedmain.Main(controllerName, horizonsource.NewController)
}
