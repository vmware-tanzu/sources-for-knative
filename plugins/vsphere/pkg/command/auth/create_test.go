/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package auth_test

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"testing"

	"github.com/vmware/govmomi/simulator"
	"github.com/vmware/govmomi/vim25"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/auth"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNewCreateCommand(t *testing.T) {
	const (
		username   = "fbiville"
		password   = "s3cr3t"
		secretName = "creds"
	)

	t.Run("defines basic metadata", func(t *testing.T) {
		cmd := auth.NewCreateCommand(&pkg.Clients{}, &auth.Options{})

		assert.Equal(t, cmd.Use, "create")
		assert.Check(t, len(cmd.Short) > 0,
			"command should have a nonempty short description")
		assert.Check(t, len(cmd.Long) > 0,
			"command should have a nonempty long description")
		command.CheckFlag(t, cmd, "username")
		command.CheckFlag(t, cmd, "password")
		assert.Assert(t, cmd.RunE != nil)
	})

	t.Run("fails to execute with an empty username", func(t *testing.T) {
		cmd, _ := authTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--password",
			password,
			"--name",
			secretName,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "requires a nonempty username provided with the --username option")
	})

	t.Run("fails to execute with an empty password", func(t *testing.T) {
		cmd, _ := authTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--username",
			username,
			"--name",
			secretName,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "'password' requires a nonempty password provided with the --password option or prompted later via the --password-std-in option")
	})

	t.Run("fails to execute with an explicit password and a stdin flag set", func(t *testing.T) {
		cmd, _ := authTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--username",
			username,
			"--name",
			secretName,
			"--password",
			password,
			"--password-stdin",
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "either set an explicit password with the --password option or set the --password-stdin option to get prompted for one, do not set both")
	})

	t.Run("fails to execute with an empty secret name", func(t *testing.T) {
		cmd, _ := authTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--username",
			username,
			"--password",
			password,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "requires a nonempty secret name provided with the --name option")
	})

	t.Run("logs in default namespace", func(t *testing.T) {
		cmd, client := authTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--username", username,
			"--password", password,
			"--name", secretName,
		})

		err := cmd.Execute()
		assert.NilError(t, err)

		secret := retrieveCreatedSecret(t, err, client, command.DefaultNamespace, secretName)
		assertSecret(t, secret, username, password)
	})

	t.Run("logs in default namespace with prompted password", func(t *testing.T) {
		cmd, client := authTestCommand(command.RegularClientConfig())
		cmd.SetIn(strings.NewReader(password))
		cmd.SetArgs([]string{
			"create",
			"--username", username,
			"--password-stdin",
			"--name", secretName,
		})

		err := cmd.Execute()
		assert.NilError(t, err)

		secret := retrieveCreatedSecret(t, err, client, command.DefaultNamespace, secretName)
		assertSecret(t, secret, username, password)
	})

	t.Run("fails to execute if password cannot be retrieved from standard input", func(t *testing.T) {
		stdInError := "oops"
		cmd, _ := authTestCommand(command.RegularClientConfig())

		cmd.SetIn(&failingReader{errorMessage: stdInError})
		cmd.SetArgs([]string{
			"create",
			"--username", username,
			"--password-stdin",
			"--name", secretName,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "failed to get password: "+stdInError)
	})

	t.Run("logs in specified namespace", func(t *testing.T) {
		namespace := "ns"

		cmd, client := authTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--namespace", namespace,
			"--username", username,
			"--password", password,
			"--name", secretName,
		})

		err := cmd.Execute()
		assert.NilError(t, err)

		secret := retrieveCreatedSecret(t, err, client, namespace, secretName)
		assertSecret(t, secret, username, password)
	})

	t.Run("fails to execute when default namespace retrieval fails", func(t *testing.T) {
		errorMsg := "a girl has no name...space"
		clientConfig := command.FailingClientConfig(fmt.Errorf(errorMsg))

		cmd, _ := authTestCommand(clientConfig)
		cmd.SetArgs([]string{
			"create",
			"--username", username,
			"--password", password,
			"--name", secretName,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "failed to get namespace: "+errorMsg)
	})

	t.Run("fails to execute if secret creation fails", func(t *testing.T) {
		cmd, client := authTestCommand(command.RegularClientConfig())

		secretCreationErrorMsg := "secret creation fail"
		client.PrependReactor("create", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, fmt.Errorf(secretCreationErrorMsg)
		})
		cmd.SetArgs([]string{
			"create",
			"--username", username,
			"--password", password,
			"--name", secretName,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, fmt.Sprintf("failed to create Secret: %s", secretCreationErrorMsg))
	})

	t.Run("fails to execute when trying to create a duplicate secret", func(t *testing.T) {
		existingSecret := newSecret(command.DefaultNamespace, secretName, username, password)
		cmd, _ := authTestCommand(command.RegularClientConfig(), existingSecret)
		cmd.SetArgs([]string{
			"create",
			"--username", username,
			"--password", password,
			"--name", secretName,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, fmt.Sprintf(`failed to create Secret: secrets "%s" already exists`, secretName))
	})

	t.Run("fails verification against vCenter due to untrusted certificate", func(t *testing.T) {
		simulator.Run(func(ctx context.Context, vc *vim25.Client) error {
			cmd, _ := authTestCommand(command.RegularClientConfig())
			cmd.SetArgs([]string{
				"create",
				"--username", username,
				"--password", password,
				"--name", secretName,
				"--verify-url", vc.URL().String(),
			})

			err := cmd.Execute()
			assert.ErrorContains(t, err, fmt.Sprintf(`failed to authenticate with vCenter: Post %q`, vc.URL().String()))
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
			cmd, _ := authTestCommand(command.RegularClientConfig())
			cmd.SetArgs([]string{
				"create",
				"--username", username,
				"--password", password,
				"--name", secretName,
				"--verify-url", vc.URL().String(),
				"--verify-insecure", // required to pass against vc simulator
			})

			err = cmd.Execute()
			assert.ErrorContains(t, err, "failed to authenticate with vCenter: ServerFaultCode: Login failure")
			return nil
		}, model)
	})

	t.Run("passes verification against vCenter with insecure flag and logs in default namespace", func(t *testing.T) {
		simulator.Run(func(ctx context.Context, vc *vim25.Client) error {
			cmd, client := authTestCommand(command.RegularClientConfig())
			cmd.SetArgs([]string{
				"create",
				"--username", username,
				"--password", password,
				"--name", secretName,
				"--verify-url", vc.URL().String(),
				"--verify-insecure",
			})

			err := cmd.Execute()
			assert.NilError(t, err)

			secret := retrieveCreatedSecret(t, err, client, command.DefaultNamespace, secretName)
			assertSecret(t, secret, username, password)
			return nil
		})
	})
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

type failingReader struct {
	errorMessage string
}

func (f *failingReader) Read(_ []byte) (n int, err error) {
	return 0, fmt.Errorf(f.errorMessage)
}
