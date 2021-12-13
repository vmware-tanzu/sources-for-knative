/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package auth_test

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/auth"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewDeleteCommand(t *testing.T) {
	const (
		username   = "fbiville"
		password   = "s3cr3t"
		secretName = "creds"
	)

	t.Run("defines basic metadata", func(t *testing.T) {
		cmd := auth.NewDeleteCommand(&pkg.Clients{}, &auth.Options{})

		assert.Equal(t, cmd.Use, "delete")
		assert.Check(t, len(cmd.Short) > 0,
			"command should have a nonempty short description")
		assert.Check(t, len(cmd.Long) > 0,
			"command should have a nonempty long description")
		command.CheckFlag(t, cmd, "name")
		assert.Assert(t, cmd.RunE != nil)
	})

	t.Run("fails to execute with an empty secret name", func(t *testing.T) {
		cmd, _ := authTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"delete",
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "requires a nonempty secret name provided with the --name option")
	})

	t.Run("fails to execute when secret does not exist", func(t *testing.T) {
		const (
			notExists = "not_exists"
		)

		cmd, _ := authTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"delete",
			"--name", notExists,
		})

		err := cmd.Execute()
		expectErr := fmt.Sprintf("failed to delete Secret: secrets %q not found", notExists)
		assert.ErrorContains(t, err, expectErr)
	})

	t.Run("logs out in default namespace", func(t *testing.T) {
		cmd, client := authTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"delete",
			"--name", secretName,
		})

		s := newSecret(command.DefaultNamespace, secretName, "user", "pass")
		_, err := client.CoreV1().Secrets(command.DefaultNamespace).Create(cmd.Context(), s, metav1.CreateOptions{})
		assert.NilError(t, err, "create test secret")

		_, err = client.CoreV1().Secrets(command.DefaultNamespace).Get(cmd.Context(), secretName, metav1.GetOptions{})
		assert.NilError(t, err)

		err = cmd.Execute()
		assert.NilError(t, err, "delete")

		_, err = client.CoreV1().Secrets(command.DefaultNamespace).Get(cmd.Context(), secretName, metav1.GetOptions{})
		assert.ErrorContains(t, err, "not found")
	})

	t.Run("fails to execute when default namespace retrieval fails", func(t *testing.T) {
		errorMsg := "a girl has no name...space"
		clientConfig := command.FailingClientConfig(fmt.Errorf(errorMsg))
		cmd, _ := authTestCommand(clientConfig)
		cmd.SetArgs([]string{
			"delete",
			"--name", secretName,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "failed to get namespace: "+errorMsg)
	})
}
