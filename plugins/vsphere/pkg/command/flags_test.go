package command_test

import (
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
	"gotest.tools/assert"
	"testing"
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
