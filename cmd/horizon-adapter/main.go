/*
Copyright 2022 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"knative.dev/eventing/pkg/adapter/v2"

	myadapter "github.com/vmware-tanzu/sources-for-knative/pkg/horizon"
)

const (
	adapterName = "horizon-source-adapter"
)

func main() {
	adapter.Main(adapterName, myadapter.NewEnv, myadapter.NewAdapter)
}
