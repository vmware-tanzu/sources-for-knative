/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package command_test

import (
	"testing"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
	"gotest.tools/assert"
)

func TestMutuallyExclusiveStringFlags(t *testing.T) {

	t.Run("mutually exclusive with 0 to 1 set value", func(t *testing.T) {
		assert.Check(t, command.MutuallyExclusiveStringFlags("", "", ""))
		assert.Check(t, command.MutuallyExclusiveStringFlags("set", "", ""))
		assert.Check(t, command.MutuallyExclusiveStringFlags("", "set", ""))
		assert.Check(t, command.MutuallyExclusiveStringFlags("", "", "set"))
	})

	t.Run("not mutually exclusive with more than 1 set value", func(t *testing.T) {
		assert.Check(t, !command.MutuallyExclusiveStringFlags("set", "set", ""))
		assert.Check(t, !command.MutuallyExclusiveStringFlags("set", "", "set"))
		assert.Check(t, !command.MutuallyExclusiveStringFlags("", "set", "set"))
		assert.Check(t, !command.MutuallyExclusiveStringFlags("set", "set", "set"))
	})
}
