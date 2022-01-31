/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package auth_test

import (
	"io/ioutil"
	"testing"

	"github.com/spf13/cobra"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/auth"
)

func TestNewAuthCommand(t *testing.T) {
	t.Run("defines basic metadata", func(t *testing.T) {
		cmd, _ := authTestCommand(command.RegularClientConfig())

		assert.Equal(t, cmd.Use, "auth")
		assert.Check(t, len(cmd.Short) > 0,
			"command should have a nonempty short description")
		assert.Check(t, len(cmd.Long) > 0,
			"command should have a nonempty long description")
		command.CheckFlag(t, cmd, "namespace")

		assert.Check(t, len(cmd.Commands()) == 2, "unexpected number of subcommands")
		assert.Check(t, command.HasLeafCommand(cmd, "create"), "command should have subcommand create")
		assert.Check(t, command.HasLeafCommand(cmd, "delete"), "command should have subcommand delete")
	})

}

func authTestCommand(clientConfig clientcmd.ClientConfig, objects ...runtime.Object) (*cobra.Command, *fake.Clientset) {
	client := fake.NewSimpleClientset(objects...)
	cmd := auth.NewAuthCommand(&pkg.Clients{
		ClientSet:    client,
		ClientConfig: clientConfig,
	})
	cmd.SetErr(ioutil.Discard)
	cmd.SetOut(ioutil.Discard)
	return cmd, client
}
