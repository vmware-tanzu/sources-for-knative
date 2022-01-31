/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package auth

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"syscall"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/soap"
	"golang.org/x/term"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
)

func NewCreateCommand(clients *pkg.Clients, opts *Options) *cobra.Command {
	result := &cobra.Command{
		Use:   "create",
		Short: "Create vSphere credentials",
		Long:  "Create vSphere credentials",
		Example: `# Create vSphere credentials in the default namespace
kn vsphere auth create --username jane-doe --password s3cr3t --name vsphere-credentials

# Create vSphere credentials in the default namespace and validate against vCenter before creating the secret
kn vsphere auth create --username jane-doe --password s3cr3t --name vsphere-credentials --verify-url https://myvc.corp.local

# Create vSphere credentials in the specified namespace
kn vsphere auth create --namespace ns --username john-doe --password s3cr3t --name vsphere-credentials

# Create vSphere credentials in the specified namespace with the password retrieved via standard input
kn vsphere auth create --namespace ns --username john-doe --password-stdin --name vsphere-credentials
`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.Username == "" {
				return fmt.Errorf("'username' requires a nonempty username provided with the --username option")
			}

			password := opts.Password
			passwordViaStdIn := opts.PasswordStdIn
			if password == "" && !passwordViaStdIn {
				return fmt.Errorf("'password' requires a nonempty password provided with the --password option or prompted later via the --password-std-in option")
			}

			if password != "" && passwordViaStdIn {
				return fmt.Errorf("either set an explicit password with the --password option or set the --password-stdin option to get prompted for one, do not set both")
			}

			secretName := opts.Name
			if secretName == "" {
				return fmt.Errorf("'name' requires a nonempty secret name provided with the --name option")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			namespace, err := clients.GetExplicitOrDefaultNamespace(opts.Namespace)
			if err != nil {
				return fmt.Errorf("failed to get namespace: %v", err)
			}

			opts.Password, err = readPassword(cmd, opts)
			if err != nil {
				return fmt.Errorf("failed to get password: %v", err)
			}

			vcURL := opts.VerifyURL
			if vcURL != "" {
				// validate credentials before creating secret
				parsedURL, err := soap.ParseURL(vcURL)
				if err != nil {
					return fmt.Errorf("failed to parse vCenter URL: %v", err)
				}

				parsedURL.User = url.UserPassword(opts.Username, opts.Password)
				_, err = govmomi.NewClient(context.TODO(), parsedURL, opts.Insecure)
				if err != nil {
					return fmt.Errorf("failed to authenticate with vCenter: %v", err)
				}
			}

			credentials := newSecret(namespace, opts)
			if _, err := clients.ClientSet.CoreV1().Secrets(namespace).Create(cmd.Context(), credentials, metav1.CreateOptions{}); err != nil {
				return fmt.Errorf("failed to create Secret: %v", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Created vSphere credentials")
			return nil
		},
	}

	flags := result.Flags()
	flags.StringVarP(&opts.Username, "username", "u", "", "username")
	flags.StringVarP(&opts.Password, "password", "p", "", "password")
	flags.BoolVarP(&opts.PasswordStdIn, "password-stdin", "i", false, "read password from standard input")
	flags.StringVar(&opts.Name, "name", "", "name of the Secret created for the credentials")
	flags.StringVar(&opts.VerifyURL, "verify-url", "", "vCenter URL to verify specified credentials (optional)")
	flags.BoolVar(&opts.Insecure, "verify-insecure", false, "Ignore certificate errors during credential verification")

	_ = result.MarkFlagRequired("username")
	_ = result.MarkFlagRequired("name")

	return result
}

func newSecret(namespace string, options *Options) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      options.Name,
		},
		Type: corev1.SecretTypeBasicAuth,
		StringData: map[string]string{
			corev1.BasicAuthUsernameKey: options.Username,
			corev1.BasicAuthPasswordKey: options.Password,
		},
	}
}

func readPassword(cmd *cobra.Command, options *Options) (string, error) {
	if !options.PasswordStdIn {
		return options.Password, nil
	}
	if term.IsTerminal(syscall.Stdin) {
		cmd.Println("Password:")
		password, err := term.ReadPassword(syscall.Stdin)
		cmd.Println()
		return string(password), err
	}
	password, err := ioutil.ReadAll(cmd.InOrStdin())
	return string(password), err
}
