/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package auth

import (
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
)

func NewDeleteCommand(clients *pkg.Clients, opts *Options) *cobra.Command {
	result := &cobra.Command{
		Use:   "delete",
		Short: "Delete vSphere credentials",
		Long:  "Delete vSphere credentials",
		Example: `# Delete vSphere credentials in the default namespace
kn vsphere auth delete --name vsphere-credentials

# Delete vSphere credentials in the specified namespace
kn vsphere auth delete --namespace ns --name vsphere-credentials
`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			secretName := opts.Name
			if secretName == "" {
				return fmt.Errorf("'name' requires a nonempty secret name provided with the --name option")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			namespace, err := clients.GetExplicitOrDefaultNamespace(opts.Namespace)
			if err != nil {
				return fmt.Errorf("failed to get namespace: %w", err)
			}

			secret := opts.Name
			if err = clients.ClientSet.CoreV1().Secrets(namespace).Delete(cmd.Context(), secret, metav1.DeleteOptions{}); err != nil {
				return fmt.Errorf("failed to delete Secret: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Deleted vSphere credentials %q", secret)
			return nil
		},
	}

	flags := result.Flags()
	flags.StringVar(&opts.Name, "name", "", "name of the credentials Secret to delete")
	_ = result.MarkFlagRequired("name")

	return result
}
