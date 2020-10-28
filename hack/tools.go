// +build tools

/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package tools

// This package imports things required by this repository, to force `go mod` to see them as dependencies
import (
	_ "k8s.io/code-generator"

	// codegen: hack/generate-knative.sh
	_ "knative.dev/hack"
	_ "knative.dev/pkg/hack"

	_ "k8s.io/code-generator/cmd/client-gen"
	_ "k8s.io/code-generator/cmd/deepcopy-gen"
	_ "k8s.io/code-generator/cmd/defaulter-gen"
	_ "k8s.io/code-generator/cmd/informer-gen"
	_ "k8s.io/code-generator/cmd/lister-gen"
	_ "k8s.io/kube-openapi/cmd/openapi-gen"
	_ "knative.dev/pkg/codegen/cmd/injection-gen"

	// For gotty
	_ "github.com/yudai/gotty"

	// For testing
	_ "github.com/vmware/govmomi/govc"
	_ "github.com/vmware/govmomi/vcsim"
)
