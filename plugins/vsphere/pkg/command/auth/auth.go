/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package auth

import (
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
)

type Options struct {
	command.GenericOptions

	Username      string
	Password      string
	PasswordStdIn bool
	VerifyURL     string
	Insecure      bool
}

func NewAuthCommand(clients *pkg.Clients) *cobra.Command {
	options := Options{}

	result := cobra.Command{
		Use:   "auth",
		Short: "Manage vSphere credentials",
		Long:  "Manage vSphere credentials",
	}

	flags := result.PersistentFlags()
	flags.StringVarP(&options.Namespace, "namespace", "n", "", "namespace to use (default namespace if omitted)")

	result.AddCommand(NewCreateCommand(clients, &options))
	result.AddCommand(NewDeleteCommand(clients, &options))

	return &result
}
