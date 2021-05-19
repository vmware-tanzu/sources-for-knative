/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package command

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

type LoginOptions struct {
	Namespace     string
	Username      string
	Password      string
	SecretName    string
	PasswordStdIn bool
	VerifyURL     string
	Insecure      bool
}

func NewLoginCommand(clients *pkg.Clients) *cobra.Command {
	options := &LoginOptions{}

	result := &cobra.Command{
		Use:   "login",
		Short: "Create vSphere credentials",
		Long:  "Create vSphere credentials",
		Example: `# Create login credentials in the default namespace
kn vsphere login --username jane-doe --password s3cr3t --secret-name vsphere-credentials
# Create login credentials in the default namespace and validate against vCenter before creating the secret
kn vsphere login --username jane-doe --password s3cr3t --secret-name vsphere-credentials --verify-url https://myvc.corp.local
# Create login credentials in the specified namespace
kn vsphere login --namespace ns --username john-doe --password s3cr3t --secret-name vsphere-credentials
# Create login credentials in the specified namespace with the password retrieved via standard input
kn vsphere login --namespace ns --username john-doe --password-stdin --secret-name vsphere-credentials
`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			username := options.Username
			if username == "" {
				return fmt.Errorf("'login' requires a nonempty username provided with the --username option")
			}

			password := options.Password
			passwordViaStdIn := options.PasswordStdIn
			if password == "" && !passwordViaStdIn {
				return fmt.Errorf("'password' requires a nonempty password provided with the --password option or prompted later via the --password-std-in option")
			}

			if password != "" && passwordViaStdIn {
				return fmt.Errorf("either set an explicit password with the --password option or set the --password-stdin option to get prompted for one, do not set both")
			}

			secretName := options.SecretName
			if secretName == "" {
				return fmt.Errorf("'secret-name' requires a nonempty secret name provided with the --secret-name option")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			namespace, err := clients.GetExplicitOrDefaultNamespace(options.Namespace)
			if err != nil {
				return fmt.Errorf("failed to get namespace: %+v", err)
			}

			password, err := readPassword(cmd, options)
			if err != nil {
				return fmt.Errorf("failed to get password: %+v", err)
			}

			vcURL := options.VerifyURL
			if vcURL != "" {
				// validate credentials before creating secret
				parsedURL, err := soap.ParseURL(vcURL)
				if err != nil {
					return fmt.Errorf("failed to parse vCenter URL: %+v", err)
				}

				parsedURL.User = url.UserPassword(options.Username, options.Password)
				_, err = govmomi.NewClient(context.TODO(), parsedURL, options.Insecure)
				if err != nil {
					return fmt.Errorf("failed to authenticate with vCenter: %+v", err)
				}
			}

			credentials := newSecret(namespace, password, options)
			if _, err := clients.ClientSet.CoreV1().Secrets(namespace).Create(cmd.Context(), credentials, metav1.CreateOptions{}); err != nil {
				return fmt.Errorf("failed to create Secret: %+v", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Created vSphere credentials")
			return nil
		},
	}

	flags := result.Flags()
	flags.StringVarP(&options.Namespace, "namespace", "n", "", "namespace of the credentials to create (default namespace if omitted)")
	flags.StringVarP(&options.Username, "username", "u", "", "username (same as VC_USERNAME)")
	_ = result.MarkFlagRequired("username")
	flags.StringVarP(&options.Password, "password", "p", "", "password (same as VC_PASSWORD)")
	flags.BoolVarP(&options.PasswordStdIn, "password-stdin", "i", false, "read password from standard input")
	flags.StringVarP(&options.SecretName, "secret-name", "s", "", "name of the Secret created for the credentials")
	flags.StringVar(&options.VerifyURL, "verify-url", "", "vCenter URL to verify specified credentials (optional)")
	flags.BoolVar(&options.Insecure, "verify-insecure", false, "Ignore certificate errors during credential verification")
	_ = result.MarkFlagRequired("secret-name")
	return result
}

func newSecret(namespace string, password string, options *LoginOptions) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      options.SecretName,
		},
		Type: corev1.SecretTypeBasicAuth,
		StringData: map[string]string{
			corev1.BasicAuthUsernameKey: options.Username,
			corev1.BasicAuthPasswordKey: password,
		},
	}
}

func readPassword(cmd *cobra.Command, options *LoginOptions) (string, error) {
	if !options.PasswordStdIn {
		return options.Password, nil
	}
	cmd.Println("Password:")
	if term.IsTerminal(syscall.Stdin) {
		password, err := term.ReadPassword(syscall.Stdin)
		cmd.Println()
		if err != nil {
			return "", err
		}
		return string(password), nil
	}
	password, err := ioutil.ReadAll(cmd.InOrStdin())
	if err != nil {
		return "", err
	}
	return string(password), nil
}
