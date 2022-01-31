/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package source_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/client/pkg/util"

	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	"github.com/vmware-tanzu/sources-for-knative/pkg/client/clientset/versioned/scheme"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/source"
)

func TestNewListCommand(t *testing.T) {
	const (
		sourceName    = "vcenter-source"
		secretRef     = "street-creds"
		sourceAddress = "https://my-vsphere-endpoint.example.com"
		sinkURI       = "https://sink.example.com"
	)

	t.Run("defines basic metadata", func(t *testing.T) {
		cmd := source.NewSourceListCommand(&pkg.Clients{}, &source.Options{})

		assert.Equal(t, cmd.Use, "list")
		assert.Check(t, cmd.HasAlias("ls"))
		assert.Check(t, len(cmd.Short) > 0,
			"command should have a nonempty short description")
		assert.Check(t, len(cmd.Long) > 0,
			"command should have a nonempty long description")
		command.CheckFlag(t, cmd, "all-namespaces")
	})

	t.Run("fails when '--namespace' and '--all-namespaces' are both set", func(t *testing.T) {
		cmd, _ := sourceTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"list",
			"--all-namespaces",
			"--namespace",
			command.DefaultNamespace,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "mutually exclusive")
	})

	t.Run("lists sources in default namespace", func(t *testing.T) {
		src1 := newSource(t, command.DefaultNamespace, sourceName+"-1", sourceAddress, secretRef, sinkURI)
		src2 := newSource(t, command.DefaultNamespace, sourceName+"-2", sourceAddress, secretRef, sinkURI)
		testSources := []runtime.Object{src1, src2}
		headers := []string{"NAME", "VCENTER", "INSECURE", "CREDENTIALS", "AGE", "CONDITIONS", "READY", "REASON"}

		cmd, _ := sourceTestCommand(command.RegularClientConfig(), testSources...)
		cmd.SetArgs([]string{
			"list",
		})

		buf := bytes.Buffer{}
		cmd.SetOut(&buf)

		err := cmd.Execute()
		assert.NilError(t, err)
		assert.Check(t, buf.String() != "")

		rows := strings.Split(buf.String(), "\n")
		assert.Check(t, util.ContainsAll(rows[0], headers...))
		assert.Check(t, util.ContainsAll(rows[1], sourceName+"-1"))
		assert.Check(t, util.ContainsAll(rows[2], sourceName+"-2"))
	})

	t.Run("lists sources in specified namespace", func(t *testing.T) {
		ns := "ns"
		src1 := newSource(t, command.DefaultNamespace, sourceName+"-1", sourceAddress, secretRef, sinkURI) // in default
		src2 := newSource(t, ns, sourceName+"-2", sourceAddress, secretRef, sinkURI)                       // in specified
		testSources := []runtime.Object{src1, src2}
		headers := []string{"NAME", "VCENTER", "INSECURE", "CREDENTIALS", "AGE", "CONDITIONS", "READY", "REASON"}

		cmd, _ := sourceTestCommand(command.RegularClientConfig(), testSources...)
		cmd.SetArgs([]string{
			"list",
			"--namespace",
			ns,
		})

		buf := bytes.Buffer{}
		cmd.SetOut(&buf)

		err := cmd.Execute()
		assert.NilError(t, err)
		assert.Check(t, buf.String() != "")

		rows := strings.Split(buf.String(), "\n")
		assert.Check(t, util.ContainsAll(rows[0], headers...))
		assert.Check(t, util.ContainsNone(rows[1], sourceName+"-1"))
		assert.Check(t, util.ContainsAll(rows[1], sourceName+"-2"))
	})

	t.Run("lists sources in all namespaces", func(t *testing.T) {
		ns := "ns"
		src1 := newSource(t, command.DefaultNamespace, sourceName+"-1", sourceAddress, secretRef, sinkURI) // in default
		src2 := newSource(t, ns, sourceName+"-2", sourceAddress, secretRef, sinkURI)                       // in specified
		testSources := []runtime.Object{src1, src2}
		headers := []string{"NAMESPACE", "NAME", "VCENTER", "INSECURE", "CREDENTIALS", "AGE", "CONDITIONS", "READY", "REASON"}

		cmd, _ := sourceTestCommand(command.RegularClientConfig(), testSources...)
		cmd.SetArgs([]string{
			"list",
			"-A",
		})

		buf := bytes.Buffer{}
		cmd.SetOut(&buf)

		err := cmd.Execute()
		assert.NilError(t, err)
		assert.Check(t, buf.String() != "")

		rows := strings.Split(buf.String(), "\n")
		assert.Check(t, util.ContainsAll(rows[0], headers...))
		assert.Check(t, util.ContainsAll(rows[1], sourceName+"-1"))
		assert.Check(t, util.ContainsAll(rows[2], sourceName+"-2"))
	})

	t.Run("prints sources in all namespaces in JSON output", func(t *testing.T) {
		ns := "ns"
		src1 := newSource(t, command.DefaultNamespace, sourceName+"-1", sourceAddress, secretRef, sinkURI) // in default
		src2 := newSource(t, ns, sourceName+"-2", sourceAddress, secretRef, sinkURI)                       // in specified

		err := util.UpdateGroupVersionKindWithScheme(src1, v1alpha1.SchemeGroupVersion, scheme.Scheme)
		assert.NilError(t, err)

		err = util.UpdateGroupVersionKindWithScheme(src2, v1alpha1.SchemeGroupVersion, scheme.Scheme)
		assert.NilError(t, err)

		testSourcesList := v1alpha1.VSphereSourceList{
			Items: []v1alpha1.VSphereSource{
				*(src1).(*v1alpha1.VSphereSource),
				*(src2).(*v1alpha1.VSphereSource),
			},
		}

		err = util.UpdateGroupVersionKindWithScheme(&testSourcesList, v1alpha1.SchemeGroupVersion, scheme.Scheme)
		assert.NilError(t, err)

		cmd, _ := sourceTestCommand(command.RegularClientConfig(), src1, src2)
		cmd.SetArgs([]string{
			"list",
			"-A",
			"-o",
			"json",
		})

		buf := bytes.Buffer{}
		cmd.SetOut(&buf)

		err = cmd.Execute()
		assert.NilError(t, err)

		var result v1alpha1.VSphereSourceList
		err = json.Unmarshal(buf.Bytes(), &result)
		assert.NilError(t, err)
		assert.DeepEqual(t, testSourcesList.Items, result.Items)
	})

	t.Run("fails to execute when default namespace retrieval fails", func(t *testing.T) {
		namespaceError := fmt.Errorf("no default namespace, oops")
		cmd, _ := sourceTestCommand(command.FailingClientConfig(namespaceError))
		cmd.SetArgs([]string{
			"list",
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "failed to get namespace")
	})
}
