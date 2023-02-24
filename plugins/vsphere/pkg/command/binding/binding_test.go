/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package binding_test

import (
	"io"
	"testing"

	"github.com/spf13/cobra"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/clientcmd"

	vspherefake "github.com/vmware-tanzu/sources-for-knative/pkg/client/clientset/versioned/fake"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/binding"
)

func TestNewBindingCommand(t *testing.T) {
	t.Run("defines basic metadata", func(t *testing.T) {
		cmd, _ := bindingTestCommand(command.RegularClientConfig())

		assert.Equal(t, cmd.Use, "binding")
		assert.Check(t, len(cmd.Short) > 0,
			"command should have a nonempty short description")
		assert.Check(t, len(cmd.Long) > 0,
			"command should have a nonempty long description")
		command.CheckFlag(t, cmd, "namespace")

		assert.Check(t, len(cmd.Commands()) == 3, "unexpected number of subcommands")
		assert.Check(t, command.HasLeafCommand(cmd, "create"), "command should have subcommand create")
		assert.Check(t, command.HasLeafCommand(cmd, "delete"), "command should have subcommand delete")
		assert.Check(t, command.HasLeafCommand(cmd, "list"), "command should have subcommand delete")
	})
}

func bindingTestCommand(clientConfig clientcmd.ClientConfig, objects ...runtime.Object) (*cobra.Command, *vspherefake.Clientset) {
	vSphereSourcesClient := vspherefake.NewSimpleClientset(objects...)
	cmd := binding.NewBindingCommand(&pkg.Clients{
		ClientSet:        k8sfake.NewSimpleClientset(),
		ClientConfig:     clientConfig,
		VSphereClientSet: vSphereSourcesClient,
	})
	cmd.SetErr(io.Discard)
	cmd.SetOut(io.Discard)
	return cmd, vSphereSourcesClient
}
