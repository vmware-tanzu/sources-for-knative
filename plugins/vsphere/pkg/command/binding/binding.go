/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package binding

import (
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
)

type Options struct {
	command.GenericOptions

	VCAddress     string
	SkipTLSVerify bool
	SecretRef     string

	SubjectAPIVersion string
	SubjectKind       string
	SubjectName       string
	SubjectSelector   string
}

func NewBindingCommand(clients *pkg.Clients) *cobra.Command {
	options := Options{}

	result := cobra.Command{
		Use:   "binding",
		Short: "Manage vSphere API bindings",
		Long:  "Manage vSphere API bindings",
	}

	fl := result.PersistentFlags()
	fl.StringVarP(&options.Namespace, "namespace", "n", "", "namespace to use (default namespace if omitted)")

	result.AddCommand(NewBindingCreateCommand(clients, &options))
	result.AddCommand(NewBindingDeleteCommand(clients, &options))
	result.AddCommand(NewBindingListCommand(clients, &options))

	return &result
}
