/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package flags_test

import (
	"testing"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/flags"

	"gotest.tools/v3/assert"
)

func TestMutuallyExclusiveStringFlags(t *testing.T) {

	t.Run("mutually exclusive with 0 to 1 set value", func(t *testing.T) {
		assert.Check(t, flags.MutuallyExclusiveStringFlags("", "", ""))
		assert.Check(t, flags.MutuallyExclusiveStringFlags("set", "", ""))
		assert.Check(t, flags.MutuallyExclusiveStringFlags("", "set", ""))
		assert.Check(t, flags.MutuallyExclusiveStringFlags("", "", "set"))
	})

	t.Run("not mutually exclusive with more than 1 set value", func(t *testing.T) {
		assert.Check(t, !flags.MutuallyExclusiveStringFlags("set", "set", ""))
		assert.Check(t, !flags.MutuallyExclusiveStringFlags("set", "", "set"))
		assert.Check(t, !flags.MutuallyExclusiveStringFlags("", "set", "set"))
		assert.Check(t, !flags.MutuallyExclusiveStringFlags("set", "set", "set"))
	})
}
