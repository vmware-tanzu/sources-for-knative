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
	"knative.dev/pkg/test/logstream"

	"github.com/vmware-tanzu/sources-for-knative/test"
)

// TestSource creates a Job that completes after receiving events, and a Source targeting that job.
func TestSource(t *testing.T) {
	// t.Parallel()
	defer logstream.Start(t)()

	clients := test.Setup(t)

	// Create a job/svc that listens for events and then quits N seconds after the first is received.
	name, wait, cancelListener := RunJobListener(t, clients)
	defer cancelListener()

	// Create a source that emits events from the vcsim.
	cancelSource := CreateSource(t, clients, name)
	defer cancelSource()

	// trigger events
	selector, cancelTrigger := CreateJobBinding(t, clients)
	defer cancelTrigger()

	script := strings.Join([]string{
		"export GOVC_URL=$VC_URL",
		"export GOVC_INSECURE=$VC_INSECURE",
		"export GOVC_USERNAME=$VC_USERNAME",
		"export GOVC_PASSWORD=$VC_PASSWORD",
		"sleep 5",
		"govc vm.power -off /DC0/vm/DC0_H0_VM0 && sleep 3",
		"govc vm.power -off /DC0/vm/DC0_H0_VM1 && sleep 3",
	}, "\n")

	// Run a simple script as a Job on the cluster.
	RunBashJob(t, clients, pkgtest.ImagePath("govc"), script, selector)

	// Wait for the job to complete, and then cleanup.
	wait()
}
