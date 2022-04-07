/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"knative.dev/eventing/pkg/adapter/v2"

	myadapter "github.com/vmware-tanzu/sources-for-knative/pkg/horizon"
)

func main() {
	adapter.Main("horizon-source-adapter", myadapter.NewEnv, myadapter.NewAdapter)
}
