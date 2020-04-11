// +build e2e

/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package e2e

import (
	"strings"
	"testing"

	pkgtest "knative.dev/pkg/test"
	"knative.dev/pkg/test/logstream"

	"github.com/vmware-tanzu/sources-for-knative/test"
)

// TestBinding creates a Binding and a Job that should be bound with credentials and run to completion successfully.
func TestBinding(t *testing.T) {
	// t.Parallel()
	defer logstream.Start(t)()

	clients := test.Setup(t)

	selector, cancel := CreateJobBinding(t, clients)
	defer cancel()

	script := strings.Join([]string{
		"govc tags.category.create testing",
		"govc tags.create -c testing shrug",
	}, "\n")

	// Run a simple script as a Job on the cluster.
	RunJobScript(t, clients, pkgtest.ImagePath("govc"), script, selector)
}
