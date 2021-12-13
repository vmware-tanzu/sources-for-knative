/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package binding

import (
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
)

func NewBindingDeleteCommand(clients *pkg.Clients, opts *Options) *cobra.Command {
	result := cobra.Command{
		Use:   "delete",
		Short: "Delete a vSphere binding",
		Long:  "Delete a vSphere binding",
		Example: `# Delete the binding in the default namespace
kn vsphere binding delete --name vc-binding

# Delete the binding in the specified namespace
kn vsphere binding delete --namespace ns --name vc-binding
`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.Name == "" {
				return fmt.Errorf("'name' requires a nonempty name provided with the --name option")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			namespace, err := clients.GetExplicitOrDefaultNamespace(opts.Namespace)
			if err != nil {
				return fmt.Errorf("failed to get namespace: %v", err)
			}
			if err = clients.VSphereClientSet.
				SourcesV1alpha1().
				VSphereBindings(namespace).
				Delete(cmd.Context(), opts.Name, metav1.DeleteOptions{}); err != nil {
				return fmt.Errorf("failed to delete binding: %v", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Deleted binding")
			return nil
		},
	}

	flags := result.Flags()
	flags.StringVar(&opts.Name, "name", "", "name of the binding to delete")
	_ = result.MarkFlagRequired("name")

	return &result
}
