/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package binding_test

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
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/binding"
)

func TestNewListCommand(t *testing.T) {
	const (
		bindingName       = "spring"
		secretRef         = "street-creds"
		bindingAddress    = "https://my-vsphere-endpoint.example.com"
		subjectAPIVersion = "apps/v1"
		subjectKind       = "Deployment"
		subjectName       = "my-simple-app"
	)

	t.Run("defines basic metadata", func(t *testing.T) {
		cmd := binding.NewBindingListCommand(&pkg.Clients{}, &binding.Options{})

		assert.Equal(t, cmd.Use, "list")
		assert.Check(t, cmd.HasAlias("ls"))
		assert.Check(t, len(cmd.Short) > 0,
			"command should have a nonempty short description")
		assert.Check(t, len(cmd.Long) > 0,
			"command should have a nonempty long description")
		command.CheckFlag(t, cmd, "all-namespaces")
	})

	t.Run("fails when '--namespace' and '--all-namespaces' are both set", func(t *testing.T) {
		cmd, _ := bindingTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"list",
			"--all-namespaces",
			"--namespace",
			command.DefaultNamespace,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "mutually exclusive")
	})

	t.Run("lists bindings in default namespace", func(t *testing.T) {
		bnd1 := newBinding(t, command.DefaultNamespace, bindingName+"-1", bindingAddress, secretRef, subjectAPIVersion, subjectKind, subjectName)
		bnd2 := newBinding(t, command.DefaultNamespace, bindingName+"-2", bindingAddress, secretRef, subjectAPIVersion, subjectKind, subjectName)
		testBindings := []runtime.Object{bnd1, bnd2}
		headers := []string{"NAME", "VCENTER", "INSECURE", "CREDENTIALS", "AGE", "CONDITIONS", "READY", "REASON"}

		cmd, _ := bindingTestCommand(command.RegularClientConfig(), testBindings...)
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
		assert.Check(t, util.ContainsAll(rows[1], bindingName+"-1"))
		assert.Check(t, util.ContainsAll(rows[2], bindingName+"-2"))
	})

	t.Run("lists bindings in specified namespace", func(t *testing.T) {
		ns := "ns"
		bnd1 := newBinding(t, command.DefaultNamespace, bindingName+"-1", bindingAddress, secretRef, subjectAPIVersion, subjectKind, subjectName)
		bnd2 := newBinding(t, ns, bindingName+"-2", bindingAddress, secretRef, subjectAPIVersion, subjectKind, subjectName)
		testBindings := []runtime.Object{bnd1, bnd2}
		headers := []string{"NAME", "VCENTER", "INSECURE", "CREDENTIALS", "AGE", "CONDITIONS", "READY", "REASON"}

		cmd, _ := bindingTestCommand(command.RegularClientConfig(), testBindings...)
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
		assert.Check(t, util.ContainsNone(rows[1], bindingName+"-1"))
		assert.Check(t, util.ContainsAll(rows[1], bindingName+"-2"))
	})

	t.Run("lists bindings in all namespaces", func(t *testing.T) {
		ns := "ns"
		bnd1 := newBinding(t, command.DefaultNamespace, bindingName+"-1", bindingAddress, secretRef, subjectAPIVersion, subjectKind, subjectName)
		bnd2 := newBinding(t, ns, bindingName+"-2", bindingAddress, secretRef, subjectAPIVersion, subjectKind, subjectName)
		testBindings := []runtime.Object{bnd1, bnd2}
		headers := []string{"NAME", "VCENTER", "INSECURE", "CREDENTIALS", "AGE", "CONDITIONS", "READY", "REASON"}

		cmd, _ := bindingTestCommand(command.RegularClientConfig(), testBindings...)
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
		assert.Check(t, util.ContainsAll(rows[1], bindingName+"-1"))
		assert.Check(t, util.ContainsAll(rows[2], bindingName+"-2"))
	})

	t.Run("prints bindings in all namespaces in JSON output", func(t *testing.T) {
		ns := "ns"
		bnd1 := newBinding(t, command.DefaultNamespace, bindingName+"-1", bindingAddress, secretRef, subjectAPIVersion, subjectKind, subjectName)
		bnd2 := newBinding(t, ns, bindingName+"-2", bindingAddress, secretRef, subjectAPIVersion, subjectKind, subjectName)

		err := util.UpdateGroupVersionKindWithScheme(bnd1, v1alpha1.SchemeGroupVersion, scheme.Scheme)
		assert.NilError(t, err)

		err = util.UpdateGroupVersionKindWithScheme(bnd2, v1alpha1.SchemeGroupVersion, scheme.Scheme)
		assert.NilError(t, err)

		testBindingsList := v1alpha1.VSphereBindingList{
			Items: []v1alpha1.VSphereBinding{
				*(bnd1).(*v1alpha1.VSphereBinding),
				*(bnd2).(*v1alpha1.VSphereBinding),
			},
		}

		err = util.UpdateGroupVersionKindWithScheme(&testBindingsList, v1alpha1.SchemeGroupVersion, scheme.Scheme)
		assert.NilError(t, err)

		cmd, _ := bindingTestCommand(command.RegularClientConfig(), bnd1, bnd2)
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

		var result v1alpha1.VSphereBindingList
		err = json.Unmarshal(buf.Bytes(), &result)
		assert.NilError(t, err)
		assert.DeepEqual(t, testBindingsList.Items, result.Items)
	})

	t.Run("fails to execute when default namespace retrieval fails", func(t *testing.T) {
		namespaceError := fmt.Errorf("no default namespace, oops")
		cmd, _ := bindingTestCommand(command.FailingClientConfig(namespaceError))
		cmd.SetArgs([]string{
			"list",
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "failed to get namespace")
	})
}
