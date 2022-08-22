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

func TestSource(t *testing.T) {
	tests := []struct {
		name, serviceAccountName string
	}{{
		name:               "default service account",
		serviceAccountName: "",
	}, {
		name:               "custom service account",
		serviceAccountName: "test-svc-name",
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			SourceTestHelper(t, test.serviceAccountName)
		})
	}
}

// TestSource creates a Job that completes after receiving events, and a Source targeting that job.
func SourceTestHelper(t *testing.T, serviceAccountName string) {
	// t.Parallel()
	clients := test.Setup(t)

	// create vcsim
	cleanupVcsim := CreateSimulator(t, clients)
	defer cleanupVcsim()

	// Create a job/svc that listens for expectedCount of events of expectedType
	// It will quit after meeting those criteria, or be cleaned up by cancelListener
	expectedType := "com.vmware.vsphere.VmPoweredOffEvent.v0"
	expectedCount := "2"

	t.Log("creating event listener")
	name, wait, cancelListener := RunJobListener(t, clients, expectedType, expectedCount)
	defer cancelListener()

	// Create a source that emits events from the vcsim.
	t.Log("creating event source")
	cancelSource := CreateSource(t, clients, name, serviceAccountName)
	defer cancelSource()

	t.Log("creating source binding")
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

	// Run a simple script as a Job on the cluster to trigger events
	t.Log("creating job to trigger events")
	RunBashJob(t, clients, pkgtest.ImagePath("govc"), script, selector)

	// Wait for the job to complete, and then cleanup.
	wait()
}
