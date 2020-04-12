// +build e2e

/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package e2e

import (
	"testing"

	"knative.dev/pkg/test/logstream"

	"github.com/vmware-tanzu/sources-for-knative/test"
)

// TestSource creates a Job that completes after receiving events, and a Source targeting that job.
func TestSource(t *testing.T) {
	// t.Parallel()
	defer logstream.Start(t)()

	clients := test.Setup(t)

	// Create a job/svc that listens for events and then quits N seconds after the first is received.
	name, wait, cancel := RunJobListener(t, clients)
	defer cancel()

	// Create a source that emits events from the vcsim.
	defer CreateSource(t, clients, name)()

	// Wait for the job to complete, and then cleanup.
	wait()
}
