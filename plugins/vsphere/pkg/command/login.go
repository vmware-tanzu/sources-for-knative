/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package command

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LoginOptions struct {
	Namespace  string
	Username   string
	Password   string
	SecretName string
}

func NewLoginCommand(client *pkg.Client) *cobra.Command {
	options := &LoginOptions{}

	result := &cobra.Command{
		Use:   "login",
		Short: "Create the required VSphere credentials",
		Long: `Create the required VSphere credentials

Examples:
# Log in the default namespace
kn vsphere login --username jane-doe --password s3cr3t --secret-name vsphere-credentials
# Log in the specified namespace
kn vsphere login --namespace ns --username john-doe --password s3cr3t --secret-name vsphere-credentials
`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			username := options.Username
			if len(username) == 0 {
				return fmt.Errorf("'login' requires a nonempty username provided with the --username option")
			}
			password := options.Password
			if len(password) == 0 {
				return fmt.Errorf("'password' requires a nonempty password provided with the --password option")
			}
			secretName := options.SecretName
			if len(secretName) == 0 {
				return fmt.Errorf("'secret-name' requires a nonempty secret name provided with the --secret-name option")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			namespace, err := client.GetExplicitOrDefaultNamespace(options.Namespace)
			if err != nil {
				return fmt.Errorf("failed to get namespace: %+v", err)
			}
			credentials := newSecret(namespace, options)
			if _, err := client.ClientSet.CoreV1().Secrets(namespace).Create(credentials); err != nil {
				return fmt.Errorf("failed to create Secret: %+v", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Created credentials")
			return nil
		},
	}

	flags := result.Flags()
	flags.StringVarP(&options.Namespace, "namespace", "n", "", "namespace of the credentials to create (default namespace if omitted)")
	flags.StringVarP(&options.Username, "username", "u", "", "username (same as GOVC_USERNAME)")
	_ = result.MarkFlagRequired("username")
	flags.StringVarP(&options.Password, "password", "p", "", "password (same as GOVC_PASSWORD)")
	_ = result.MarkFlagRequired("password")
	flags.StringVarP(&options.SecretName, "secret-name", "s", "", "name of the Secret created for the credentials")
	_ = result.MarkFlagRequired("secret-name")
	return result
}

func newSecret(namespace string, options *LoginOptions) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      options.SecretName,
		},
		Type: corev1.SecretTypeBasicAuth,
		StringData: map[string]string{
			"username": options.Username,
			"password": options.Password,
		},
	}
}
