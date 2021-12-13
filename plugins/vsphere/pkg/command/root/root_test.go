/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package root_test

import (
	"testing"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/root"

	"gotest.tools/v3/assert"
)

func TestNewRootCommand(t *testing.T) {
	rootCommand := root.NewRootCommand(&pkg.Clients{})

	assert.Equal(t, "kn-vsphere", rootCommand.Name())
	assert.Check(t, len(rootCommand.Short) > 0,
		"command should have a nonempty description")
	assert.Check(t, len(rootCommand.Commands()) == 4, "unexpected number of subcommands")
	assert.Check(t, command.HasLeafCommand(rootCommand, "auth"),
		"command should have subcommand auth")
	assert.Check(t, command.HasLeafCommand(rootCommand, "source"),
		"command should have subcommand source")
	assert.Check(t, command.HasLeafCommand(rootCommand, "binding"),
		"command should have subcommand binding")
	assert.Check(t, command.HasLeafCommand(rootCommand, "version"),
		"command should have subcommand version")
}
