/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package binding_test

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/binding"
)

func TestNewBindingDeleteCommand(t *testing.T) {
	const (
		bindingName       = "spring"
		secretRef         = "street-creds"
		bindingAddress    = "https://my-vsphere-endpoint.example.com"
		subjectAPIVersion = "apps/v1"
		subjectKind       = "Deployment"
		subjectName       = "my-simple-app"
	)

	t.Run("defines basic metadata", func(t *testing.T) {
		cmd := binding.NewBindingDeleteCommand(&pkg.Clients{}, &binding.Options{})

		assert.Equal(t, cmd.Use, "delete")
		assert.Check(t, len(cmd.Short) > 0,
			"command should have a nonempty short description")
		assert.Check(t, len(cmd.Long) > 0,
			"command should have a nonempty long description")
		command.CheckFlag(t, cmd, "name")
		assert.Assert(t, cmd.RunE != nil)
	})

	t.Run("fails to execute with an empty name", func(t *testing.T) {
		cmd, _ := bindingTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"delete",
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "requires a nonempty name provided with the --name option")
	})

	t.Run("deletes binding in default namespace", func(t *testing.T) {
		existingBinding := newBinding(t, command.DefaultNamespace, bindingName, bindingAddress, secretRef, subjectAPIVersion, subjectKind, subjectName)
		cmd, client := bindingTestCommand(command.RegularClientConfig(), existingBinding)
		cmd.SetArgs([]string{
			"delete",
			"--name", bindingName,
		})

		err := cmd.Execute()
		assert.NilError(t, err)

		_, err = client.SourcesV1alpha1().VSphereBindings(command.DefaultNamespace).Get(cmd.Context(), bindingName, metav1.GetOptions{})
		assert.ErrorContains(t, err, fmt.Sprintf("vspherebindings.sources.tanzu.vmware.com %q not found", bindingName))
	})

	t.Run("deletes source in custom namespace", func(t *testing.T) {
		ns := "ns"
		existingBinding := newBinding(t, ns, bindingName, bindingAddress, secretRef, subjectAPIVersion, subjectKind, subjectName)
		cmd, client := bindingTestCommand(command.RegularClientConfig(), existingBinding)
		cmd.SetArgs([]string{
			"delete",
			"--name", bindingName,
			"--namespace",
			ns,
		})

		err := cmd.Execute()
		assert.NilError(t, err)

		_, err = client.SourcesV1alpha1().VSphereBindings(ns).Get(cmd.Context(), bindingName, metav1.GetOptions{})
		assert.ErrorContains(t, err, fmt.Sprintf("vspherebindings.sources.tanzu.vmware.com %q not found", bindingName))
	})

	t.Run("fails to execute when default namespace retrieval fails", func(t *testing.T) {
		namespaceError := fmt.Errorf("no default namespace, oops")
		cmd, _ := bindingTestCommand(command.FailingClientConfig(namespaceError))
		cmd.SetArgs([]string{
			"delete",
			"--name", bindingName,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "failed to get namespace")
	})

	t.Run("fails to execute when the binding does not exist", func(t *testing.T) {
		cmd, _ := bindingTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"delete",
			"--name", bindingName,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, fmt.Sprintf("vspherebindings.sources.tanzu.vmware.com %q not found", bindingName))
	})

}
