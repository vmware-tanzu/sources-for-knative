/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package source_test

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/source"
)

func TestNewSourceDeleteCommand(t *testing.T) {
	const (
		sourceName    = "spring"
		secretRef     = "street-creds"
		sourceAddress = "https://my-vsphere-endpoint.example.com"
		sinkURI       = "https://sink.example.com"
	)

	t.Run("defines basic metadata", func(t *testing.T) {
		cmd := source.NewSourceDeleteCommand(&pkg.Clients{}, &source.Options{})

		assert.Equal(t, cmd.Use, "delete")
		assert.Check(t, len(cmd.Short) > 0,
			"command should have a nonempty short description")
		assert.Check(t, len(cmd.Long) > 0,
			"command should have a nonempty long description")
		command.CheckFlag(t, cmd, "name")
		assert.Assert(t, cmd.RunE != nil)
	})

	t.Run("fails to execute with an empty name", func(t *testing.T) {
		cmd, _ := sourceTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"delete",
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "requires a nonempty name provided with the --name option")
	})

	t.Run("deletes source in default namespace", func(t *testing.T) {
		existingSource := newSource(t, command.DefaultNamespace, sourceName, sourceAddress, secretRef, sinkURI)
		cmd, client := sourceTestCommand(command.RegularClientConfig(), existingSource)
		cmd.SetArgs([]string{
			"delete",
			"--name", sourceName,
		})

		err := cmd.Execute()
		assert.NilError(t, err)

		_, err = client.SourcesV1alpha1().VSphereSources(command.DefaultNamespace).Get(cmd.Context(), sourceName, metav1.GetOptions{})
		assert.ErrorContains(t, err, fmt.Sprintf("vspheresources.sources.tanzu.vmware.com %q not found", sourceName))
	})

	t.Run("deletes source in custom namespace", func(t *testing.T) {
		ns := "ns"
		existingSource := newSource(t, ns, sourceName, sourceAddress, secretRef, sinkURI)
		cmd, client := sourceTestCommand(command.RegularClientConfig(), existingSource)
		cmd.SetArgs([]string{
			"delete",
			"--name", sourceName,
			"--namespace",
			ns,
		})

		err := cmd.Execute()
		assert.NilError(t, err)

		_, err = client.SourcesV1alpha1().VSphereSources(ns).Get(cmd.Context(), sourceName, metav1.GetOptions{})
		assert.ErrorContains(t, err, fmt.Sprintf("vspheresources.sources.tanzu.vmware.com %q not found", sourceName))
	})

	t.Run("fails to execute when default namespace retrieval fails", func(t *testing.T) {
		namespaceError := fmt.Errorf("no default namespace, oops")
		cmd, _ := sourceTestCommand(command.FailingClientConfig(namespaceError))
		cmd.SetArgs([]string{
			"delete",
			"--name", sourceName,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "failed to get namespace")
	})

	t.Run("fails to execute when the source does not exist", func(t *testing.T) {
		cmd, _ := sourceTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"delete",
			"--name", sourceName,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, fmt.Sprintf("vspheresources.sources.tanzu.vmware.com %q not found", sourceName))
	})

}
