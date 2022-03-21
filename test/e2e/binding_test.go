//go:build e2e
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

	"github.com/vmware-tanzu/sources-for-knative/test"
)

// TestBindingGOVC creates a Binding and a Job that should be bound with credentials and
// run to completion successfully.
func TestBindingGOVC(t *testing.T) {
	// t.Parallel()
	clients := test.Setup(t)

	// create vcsim
	cleanupVcsim := CreateSimulator(t, clients)
	defer cleanupVcsim()

	selector, cancel := CreateJobBinding(t, clients)
	defer cancel()

	script := strings.Join([]string{
		"export GOVC_URL=$VC_URL",
		"export GOVC_INSECURE=$VC_INSECURE",
		"export GOVC_USERNAME=$VC_USERNAME",
		"export GOVC_PASSWORD=$VC_PASSWORD",
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
	clients := test.Setup(t)

	// create vcsim
	cleanupVcsim := CreateSimulator(t, clients)
	defer cleanupVcsim()

	selector, cancel := CreateJobBinding(t, clients)
	defer cancel()

	script := strings.Join([]string{
		// Log into the VI Server
		"Set-PowerCLIConfiguration -InvalidCertificateAction Ignore -Confirm:$false | Out-Null",
		"Connect-VIServer -Server ([System.Uri]$env:VC_URL).Host -User $env:VC_USERNAME -Password $env:VC_PASSWORD",

		// Get Events and write them out.
		"Get-VIEvent | Write-Host",
	}, "\n")

	// Run a simple script as a Job on the cluster.
	RunPowershellJob(t, clients, "docker.io/vmware/powerclicore", script, selector)
}
