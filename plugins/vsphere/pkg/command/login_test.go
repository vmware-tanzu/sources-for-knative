/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package command_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
	"testing"

	"github.com/vmware/govmomi/simulator"
	"github.com/vmware/govmomi/vim25"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const defaultNamespace = "configuredDefault"

func TestNewLoginCommand(t *testing.T) {
	const username = "fbiville"
	const password = "s3cr3t"
	const secretName = "creds"

	t.Run("defines basic metadata", func(t *testing.T) {
		loginCommand := loginCommand(&pkg.Clients{})

		assert.Equal(t, loginCommand.Use, "login")
		assert.Check(t, len(loginCommand.Short) > 0,
			"command should have a nonempty short description")
		assert.Check(t, len(loginCommand.Long) > 0,
			"command should have a nonempty long description")
		checkFlag(t, loginCommand, "namespace")
		checkFlag(t, loginCommand, "username")
		checkFlag(t, loginCommand, "password")
		assert.Assert(t, loginCommand.RunE != nil)
	})

	t.Run("fails to execute with an empty username", func(t *testing.T) {
		loginCommand := loginCommand(&pkg.Clients{})
		loginCommand.SetArgs([]string{"--password", password, "--secret-name", secretName})

		err := loginCommand.Execute()

		assert.ErrorContains(t, err, "requires a nonempty username provided with the --username option")
	})

	t.Run("fails to execute with an empty password", func(t *testing.T) {
		loginCommand := loginCommand(&pkg.Clients{})
		loginCommand.SetArgs([]string{"--username", username, "--secret-name", secretName})

		err := loginCommand.Execute()

		assert.ErrorContains(t, err, "'password' requires a nonempty password provided with the --password option or prompted later via the --password-std-in option")
	})

	t.Run("fails to execute with an explicit password and a stdin flag set", func(t *testing.T) {
		loginCommand := loginCommand(&pkg.Clients{})
		loginCommand.SetArgs([]string{"--username", username, "--secret-name", secretName, "--password", password, "--password-stdin"})

		err := loginCommand.Execute()

		assert.ErrorContains(t, err, "either set an explicit password with the --password option or set the --password-stdin option to get prompted for one, do not set both")
	})

	t.Run("fails to execute with an empty secret name", func(t *testing.T) {
		loginCommand := loginCommand(&pkg.Clients{})
		loginCommand.SetArgs([]string{"--username", username, "--password", password})

		err := loginCommand.Execute()

		assert.ErrorContains(t, err, "requires a nonempty secret name provided with the --secret-name option")
	})

	t.Run("logs in default namespace", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		loginCommand := loginCommand(&pkg.Clients{ClientSet: client, ClientConfig: regularClientConfig()})
		loginCommand.SetArgs([]string{
			"--username", username,
			"--password", password,
			"--secret-name", secretName,
		})

		err := loginCommand.Execute()

		secret := retrieveCreatedSecret(t, err, client, defaultNamespace, secretName)
		assertSecret(t, secret, username, password)
	})

	t.Run("logs in default namespace with prompted password", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		loginCommand := loginCommand(&pkg.Clients{ClientSet: client, ClientConfig: regularClientConfig()})
		loginCommand.SetIn(strings.NewReader(password))
		loginCommand.SetArgs([]string{
			"--username", username,
			"--password-stdin",
			"--secret-name", secretName,
		})

		err := loginCommand.Execute()

		secret := retrieveCreatedSecret(t, err, client, defaultNamespace, secretName)
		assertSecret(t, secret, username, password)
	})

	t.Run("fails to execute if password cannot be retrieved from standard input", func(t *testing.T) {
		stdInError := "oops"
		loginCommand := loginCommand(&pkg.Clients{ClientSet: fake.NewSimpleClientset(), ClientConfig: regularClientConfig()})
		loginCommand.SetIn(&failingReader{errorMessage: stdInError})
		loginCommand.SetArgs([]string{
			"--username", username,
			"--password-stdin",
			"--secret-name", secretName,
		})

		err := loginCommand.Execute()

		assert.ErrorContains(t, err, "failed to get password: "+stdInError)
	})

	t.Run("logs in specified namespace", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		loginCommand := loginCommand(&pkg.Clients{ClientSet: client})
		namespace := "ns"
		loginCommand.SetArgs([]string{
			"--namespace", namespace,
			"--username", username,
			"--password", password,
			"--secret-name", secretName,
		})

		err := loginCommand.Execute()

		secret := retrieveCreatedSecret(t, err, client, namespace, secretName)
		assertSecret(t, secret, username, password)
	})

	t.Run("fails to execute when default namespace retrieval fails", func(t *testing.T) {
		errorMsg := "a girl has no name...space"
		clientConfig := failingClientConfig(fmt.Errorf(errorMsg))
		loginCommand := loginCommand(&pkg.Clients{ClientSet: fake.NewSimpleClientset(), ClientConfig: clientConfig})
		loginCommand.SetArgs([]string{
			"--username", username,
			"--password", password,
			"--secret-name", secretName,
		})

		err := loginCommand.Execute()

		assert.ErrorContains(t, err, "failed to get namespace: "+errorMsg)
	})

	t.Run("fails to execute if secret creation fails", func(t *testing.T) {
		secretCreationErrorMsg := "secret creation fail"
		client := fake.NewSimpleClientset()
		client.PrependReactor("create", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, fmt.Errorf(secretCreationErrorMsg)
		})
		loginCommand := loginCommand(&pkg.Clients{ClientSet: client, ClientConfig: regularClientConfig()})
		loginCommand.SetArgs([]string{
			"--username", username,
			"--password", password,
			"--secret-name", secretName,
		})

		err := loginCommand.Execute()

		assert.ErrorContains(t, err, fmt.Sprintf("failed to create Secret: %s", secretCreationErrorMsg))
	})

	t.Run("fails to execute when trying to create a duplicate secret", func(t *testing.T) {
		existingSecret := newSecret(defaultNamespace, secretName, username, password)
		loginCommand := loginCommand(&pkg.Clients{ClientSet: fake.NewSimpleClientset(existingSecret), ClientConfig: regularClientConfig()})
		loginCommand.SetArgs([]string{
			"--username", username,
			"--password", password,
			"--secret-name", secretName,
		})

		err := loginCommand.Execute()

		assert.ErrorContains(t, err, fmt.Sprintf(`failed to create Secret: secrets "%s" already exists`, secretName))
	})

	t.Run("fails verification against vCenter due to untrusted certificate", func(t *testing.T) {
		simulator.Run(func(ctx context.Context, vc *vim25.Client) error {
			client := fake.NewSimpleClientset()
			loginCommand := loginCommand(&pkg.Clients{ClientSet: client, ClientConfig: regularClientConfig()})
			loginCommand.SetArgs([]string{
				"--username", username,
				"--password", password,
				"--secret-name", secretName,
				"--verify-url", vc.URL().String(),
			})

			err := loginCommand.Execute()

			assert.ErrorContains(t, err, fmt.Sprintf(`failed to authenticate with vCenter: Post %q: x509: certificate signed by unknown authority`, vc.URL().String()))
			return nil
		})
	})

	t.Run("fails verification against vCenter due to incorrect username", func(t *testing.T) {
		model := simulator.VPX()

		defer model.Remove()
		err := model.Create()
		if err != nil {
			log.Fatal(err)
		}

		model.Service.Listen = &url.URL{
			User: url.UserPassword("not-my-username", password),
		}

		simulator.Run(func(ctx context.Context, vc *vim25.Client) error {
			client := fake.NewSimpleClientset()
			loginCommand := loginCommand(&pkg.Clients{ClientSet: client, ClientConfig: regularClientConfig()})
			loginCommand.SetArgs([]string{
				"--username", username,
				"--password", password,
				"--secret-name", secretName,
				"--verify-url", vc.URL().String(),
				"--verify-insecure", // required to pass against vc simulator
			})

			err := loginCommand.Execute()

			assert.ErrorContains(t, err, "failed to authenticate with vCenter: ServerFaultCode: Login failure")
			return nil
		}, model)
	})

	t.Run("passes verification against vCenter with insecure flag and logs in default namespace", func(t *testing.T) {
		simulator.Run(func(ctx context.Context, vc *vim25.Client) error {
			client := fake.NewSimpleClientset()
			loginCommand := loginCommand(&pkg.Clients{ClientSet: client, ClientConfig: regularClientConfig()})
			loginCommand.SetArgs([]string{
				"--username", username,
				"--password", password,
				"--secret-name", secretName,
				"--verify-url", vc.URL().String(),
				"--verify-insecure",
			})

			err := loginCommand.Execute()

			secret := retrieveCreatedSecret(t, err, client, defaultNamespace, secretName)
			assertSecret(t, secret, username, password)
			return nil
		})
	})
}

func loginCommand(client *pkg.Clients) *cobra.Command {
	loginCommand := command.NewLoginCommand(client)
	loginCommand.SetErr(ioutil.Discard)
	loginCommand.SetOut(ioutil.Discard)
	return loginCommand
}

func newSecret(namespace, secretName, username, password string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      secretName,
		},
		Type: corev1.SecretTypeBasicAuth,
		StringData: map[string]string{
			corev1.BasicAuthUsernameKey: username,
			corev1.BasicAuthPasswordKey: password,
		},
	}
}

func retrieveCreatedSecret(t *testing.T, err error, client *fake.Clientset, ns, name string) *corev1.Secret {
	assert.NilError(t, err)
	secret, err := client.CoreV1().
		Secrets(ns).
		Get(context.Background(), name, metav1.GetOptions{})
	assert.NilError(t, err)
	return secret
}

func assertSecret(t *testing.T, secret *corev1.Secret, username string, password string) {
	assert.Equal(t, secret.Type, corev1.SecretTypeBasicAuth)
	assert.Equal(t, secret.StringData[corev1.BasicAuthUsernameKey], username)
	assert.Equal(t, secret.StringData[corev1.BasicAuthPasswordKey], password)
}

func checkFlag(t *testing.T, command *cobra.Command, flagName string) bool {
	return assert.Check(t, command.Flag(flagName) != nil, "command should have a '%s' flag", flagName)
}

type failingReader struct {
	errorMessage string
}

func (f *failingReader) Read(ignored []byte) (n int, err error) {
	return 0, fmt.Errorf(f.errorMessage)
}

func regularClientConfig() clientcmd.ClientConfig {
	return command.FakeClientConfig{DefaultNamespaceProvider: func() (string, error) {
		return defaultNamespace, nil
	}}
}

func failingClientConfig(err error) clientcmd.ClientConfig {
	return command.FakeClientConfig{DefaultNamespaceProvider: func() (string, error) {
		return "", err
	}}
}
