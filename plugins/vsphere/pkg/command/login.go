/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package command

import (
	"fmt"
	"io/ioutil"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LoginOptions struct {
	Namespace     string
	Username      string
	Password      string
	SecretName    string
	PasswordStdIn bool
}

func NewLoginCommand(clients *pkg.Client) *cobra.Command {
	options := &LoginOptions{}

	result := &cobra.Command{
		Use:   "login",
		Short: "Create the required vSphere credentials",
		Long:  "Create the required vSphere credentials",
		Example: `# Log in the default namespace
kn vsphere login --username jane-doe --password s3cr3t --secret-name vsphere-credentials
# Log in the specified namespace
kn vsphere login --namespace ns --username john-doe --password s3cr3t --secret-name vsphere-credentials
# Log in the specified namespace with the password retrieved via standard input
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
			credentials := newSecret(namespace, password, options)
			if _, err := clients.ClientSet.CoreV1().Secrets(namespace).Create(credentials); err != nil {
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
	flags.BoolVarP(&options.PasswordStdIn, "password-stdin", "i", false, "read password from standard input")
	flags.StringVarP(&options.SecretName, "secret-name", "s", "", "name of the Secret created for the credentials")
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
	if terminal.IsTerminal(syscall.Stdin) {
		password, err := terminal.ReadPassword(syscall.Stdin)
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
