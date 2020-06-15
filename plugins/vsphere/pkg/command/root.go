/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package command

import (
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
)

// Returns the root command of the CLI
func NewRootCommand(clients *pkg.Clients) *cobra.Command {
	result := cobra.Command{
		Use:   "kn-vsphere",
		Short: "Knative plugin to create Knative compatible Event Sources for VSphere events,\nand Bindings to access the VSphere API",
	}
	result.AddCommand(NewLoginCommand(clients))
	result.AddCommand(NewSourceCommand(clients))
	result.AddCommand(NewBindingCommand(clients))
	result.AddCommand(NewVersionCommand())
	return &result
}
