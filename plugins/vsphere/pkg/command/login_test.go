/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package command_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNewLoginCommand(t *testing.T) {
	username := "fbiville"
	password := "s3cr3t"
	secretName := "creds"

	t.Run("defines basic metadata", func(t *testing.T) {
		loginCommand := loginCommand(&pkg.Client{})

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
		loginCommand := loginCommand(&pkg.Client{})
		loginCommand.SetArgs([]string{"--password", password, "--secret-name", secretName})

		err := loginCommand.Execute()

		assert.ErrorContains(t, err, "requires a nonempty username provided with the --username option")
	})

	t.Run("fails to execute with an empty password", func(t *testing.T) {
		loginCommand := loginCommand(&pkg.Client{})
		loginCommand.SetArgs([]string{"--username", username, "--secret-name", secretName})

		err := loginCommand.Execute()

		assert.ErrorContains(t, err, "'password' requires a nonempty password provided with the --password option or prompted later via the --password-std-in option")
	})

	t.Run("fails to execute with an explicit password and a stdin flag set", func(t *testing.T) {
		loginCommand := loginCommand(&pkg.Client{})
		loginCommand.SetArgs([]string{"--username", username, "--secret-name", secretName, "--password", password, "--password-stdin"})

		err := loginCommand.Execute()

		assert.ErrorContains(t, err, "either set an explicit password with the --password option or set the --password-stdin option to get prompted for one, do not set both")
	})

	t.Run("fails to execute with an empty secret name", func(t *testing.T) {
		loginCommand := loginCommand(&pkg.Client{})
		loginCommand.SetArgs([]string{"--username", username, "--password", password})

		err := loginCommand.Execute()

		assert.ErrorContains(t, err, "requires a nonempty secret name provided with the --secret-name option")
	})

	t.Run("logs in default namespace", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		defaultNamespace := "configured-default"
		clientConfig := command.FakeClientConfig{DefaultNamespaceProvider: func() (string, error) {
			return defaultNamespace, nil
		}}
		loginCommand := loginCommand(&pkg.Client{ClientSet: client, ClientConfig: clientConfig})
		loginCommand.SetArgs([]string{
			"--username", username,
			"--password", password,
			"--secret-name", secretName,
		})

		err := loginCommand.Execute()

		assert.NilError(t, err)
		secret, err := client.CoreV1().
			Secrets(defaultNamespace).
			Get(secretName, metav1.GetOptions{})
		assert.NilError(t, err)
		assert.Equal(t, secret.Type, corev1.SecretTypeBasicAuth)
		assert.Equal(t, secret.StringData["username"], username)
		assert.Equal(t, secret.StringData["password"], password)
	})

	t.Run("logs in default namespace with prompted password", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		defaultNamespace := "configured-default"
		clientConfig := command.FakeClientConfig{DefaultNamespaceProvider: func() (string, error) {
			return defaultNamespace, nil
		}}
		loginCommand := loginCommand(&pkg.Client{ClientSet: client, ClientConfig: clientConfig})
		loginCommand.SetIn(strings.NewReader(password))
		loginCommand.SetArgs([]string{
			"--username", username,
			"--password-stdin",
			"--secret-name", secretName,
		})

		err := loginCommand.Execute()

		assert.NilError(t, err)
		secret, err := client.CoreV1().
			Secrets(defaultNamespace).
			Get(secretName, metav1.GetOptions{})
		assert.NilError(t, err)
		assert.Equal(t, secret.Type, corev1.SecretTypeBasicAuth)
		assert.Equal(t, secret.StringData["username"], username)
		assert.Equal(t, secret.StringData["password"], password)
	})

	t.Run("fails to execute if password cannot be retrieved from standard input", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		defaultNamespace := "configured-default"
		clientConfig := command.FakeClientConfig{DefaultNamespaceProvider: func() (string, error) {
			return defaultNamespace, nil
		}}
		errorMsg := "oops"
		loginCommand := loginCommand(&pkg.Client{ClientSet: client, ClientConfig: clientConfig})
		loginCommand.SetIn(&failingReader{errorMessage: errorMsg})
		loginCommand.SetArgs([]string{
			"--username", username,
			"--password-stdin",
			"--secret-name", secretName,
		})

		err := loginCommand.Execute()

		assert.ErrorContains(t, err, "failed to get password: "+errorMsg)
	})

	t.Run("logs in specified namespace", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		loginCommand := loginCommand(&pkg.Client{ClientSet: client})
		namespace := "ns"
		loginCommand.SetArgs([]string{
			"--namespace", namespace,
			"--username", username,
			"--password", password,
			"--secret-name", secretName,
		})

		err := loginCommand.Execute()

		assert.NilError(t, err)
		secret, err := client.CoreV1().
			Secrets(namespace).
			Get(secretName, metav1.GetOptions{})
		assert.NilError(t, err)
		assert.Equal(t, secret.Type, corev1.SecretTypeBasicAuth)
		assert.Equal(t, secret.StringData["username"], username)
		assert.Equal(t, secret.StringData["password"], password)
	})

	t.Run("fails to execute in default namespace when its retrieval fails", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		errorMsg := "a girl has no name...space"
		clientConfig := command.FakeClientConfig{DefaultNamespaceProvider: func() (string, error) {
			return "", errors.New(errorMsg)
		}}
		loginCommand := loginCommand(&pkg.Client{ClientSet: client, ClientConfig: clientConfig})
		loginCommand.SetArgs([]string{
			"--username", username,
			"--password", password,
			"--secret-name", secretName,
		})

		err := loginCommand.Execute()

		assert.ErrorContains(t, err, "failed to get namespace: "+errorMsg)
	})
}

func loginCommand(client *pkg.Client) *cobra.Command {
	loginCommand := command.NewLoginCommand(client)
	loginCommand.SetErr(ioutil.Discard)
	loginCommand.SetOut(ioutil.Discard)
	return loginCommand
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
