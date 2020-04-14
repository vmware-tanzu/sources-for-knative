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

// TestBindingGOVC creates a Binding and a Job that should be bound with credentials and
// run to completion successfully.
func TestBindingGOVC(t *testing.T) {
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
	RunBashJob(t, clients, pkgtest.ImagePath("govc"), script, selector)
}

// TestBindingPowerCLICore creates a Binding and a Job that should be bound with credentials
// and run to completion successfully.
func TestBindingPowerCLICore(t *testing.T) {
	// t.Parallel()
	defer logstream.Start(t)()

	clients := test.Setup(t)

	selector, cancel := CreateJobBinding(t, clients)
	defer cancel()

	script := strings.Join([]string{
		// Log into the VI Server
		"Set-PowerCLIConfiguration -InvalidCertificateAction Ignore -Confirm:$false | Out-Null",
		"Connect-VIServer -Server ([System.Uri]$env:GOVC_URL).Host -User $env:GOVC_USERNAME -Password $env:GOVC_PASSWORD",

		// Get Events and write them out.
		"Get-VIEvent | Write-Host",
	}, "\n")

	// Run a simple script as a Job on the cluster.
	RunPowershellJob(t, clients, "docker.io/vmware/powerclicore", script, selector)
}
