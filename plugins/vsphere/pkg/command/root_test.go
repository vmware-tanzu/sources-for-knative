/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package command_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
	"gotest.tools/assert"
)

func TestNewRootCommand(t *testing.T) {
	rootCommand := command.NewRootCommand(&pkg.Clients{})

	assert.Equal(t, "kn-vsphere", rootCommand.Name())
	assert.Check(t, len(rootCommand.Short) > 0,
		"command should have a nonempty description")
	assert.Check(t, len(rootCommand.Commands()) == 3, "unexpected number of subcommands")
	assert.Check(t, HasLeafCommand(rootCommand, "login"),
		"command should have subcommand login")
	assert.Check(t, HasLeafCommand(rootCommand, "source"),
		"command should have subcommand source")
	assert.Check(t, HasLeafCommand(rootCommand, "version"),
		"command should have subcommand version")
}

func HasLeafCommand(command *cobra.Command, subcommandName string) bool {
	_, unprocessed, err := command.Find([]string{subcommandName})
	return err == nil && len(unprocessed) == 0
}
