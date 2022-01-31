/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package root

import (
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/auth"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/binding"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/source"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/version"
)

// NewRootCommand returns the root command of the CLI
func NewRootCommand(clients *pkg.Clients) *cobra.Command {
	result := cobra.Command{
		Use:   "kn-vsphere",
		Short: "Knative plugin to create Knative compatible Event Sources for VMware vSphere events,\nand Bindings to access the vSphere API",
	}

	result.AddCommand(auth.NewAuthCommand(clients))
	result.AddCommand(source.NewSourceCommand(clients))
	result.AddCommand(binding.NewBindingCommand(clients))
	result.AddCommand(version.NewVersionCommand())

	return &result
}
