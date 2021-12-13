/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package binding_test

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/tracker"

	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	vsphere "github.com/vmware-tanzu/sources-for-knative/pkg/client/clientset/versioned"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/binding"
)

func TestNewBindingCreateCommand(t *testing.T) {
	const (
		bindingName    = "spring"
		secretRef      = "street-creds"
		bindingAddress = "https://my-vsphere-endpoint.example.com"
	)

	t.Run("defines basic metadata", func(t *testing.T) {
		cmd := binding.NewBindingCreateCommand(&pkg.Clients{}, &binding.Options{})

		assert.Equal(t, cmd.Use, "create")
		assert.Check(t, len(cmd.Short) > 0,
			"command should have a nonempty short description")
		assert.Check(t, len(cmd.Long) > 0,
			"command should have a nonempty long description")
		command.CheckFlag(t, cmd, "name")
		command.CheckFlag(t, cmd, "vc-address")
		command.CheckFlag(t, cmd, "skip-tls-verify")
		command.CheckFlag(t, cmd, "secret-ref")
		command.CheckFlag(t, cmd, "subject-api-version")
		command.CheckFlag(t, cmd, "subject-kind")
		command.CheckFlag(t, cmd, "subject-name")
		command.CheckFlag(t, cmd, "subject-selector")
		assert.Assert(t, cmd.RunE != nil)
	})

	t.Run("fails to execute with an empty name", func(t *testing.T) {
		cmd, _ := bindingTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--vc-address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
			"--subject-name", "my-simple-app",
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "requires a nonempty name provided with the --name option")
	})

	t.Run("fails to execute with an empty address", func(t *testing.T) {
		cmd, _ := bindingTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--name", bindingName,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
			"--subject-name", "my-simple-app",
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "requires a nonempty address provided with the --vc-address option")
	})

	t.Run("fails to execute with an empty secret reference", func(t *testing.T) {
		cmd, _ := bindingTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--name", bindingName,
			"--vc-address", bindingAddress,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
			"--subject-name", "my-simple-app",
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "requires a nonempty secret reference provided with the --secret-ref option")
	})

	t.Run("fails to execute with an empty subject API version", func(t *testing.T) {
		cmd, _ := bindingTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--name", bindingName,
			"--vc-address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-kind", "Deployment",
			"--subject-name", "my-simple-app",
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "requires a nonempty subject API version provided with the --subject-api-version option")
	})

	t.Run("fails to execute with an empty subject kind", func(t *testing.T) {
		cmd, _ := bindingTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--name", bindingName,
			"--vc-address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-name", "my-simple-app",
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "requires a nonempty subject kind provided with the --subject-kind option")
	})

	t.Run("fails to execute with an empty subject name and empty selector", func(t *testing.T) {
		cmd, _ := bindingTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--name", bindingName,
			"--vc-address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, `requires a nonempty subject name provided with the --subject-name option,
or a nonempty subject selector with the --subject-selector option`)
	})

	t.Run("fails to execute with both subject name and selector set", func(t *testing.T) {
		cmd, _ := bindingTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--name", bindingName,
			"--vc-address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
			"--subject-name", "my-simple-app",
			"--subject-selector", "foo:bar",
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, `subject can optionally be configured with one of the following flags (but several were set):
	--subject-name, --subject-selector`)
	})

	t.Run("creates basic binding in default namespace", func(t *testing.T) {
		var (
			subjectAPIVersion = "apps/v1"
			subjectKind       = "Deployment"
			subjectName       = "my-simple-app"
		)

		cmd, vSphereClientSet := bindingTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--name", bindingName,
			"--vc-address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", subjectAPIVersion,
			"--subject-kind", subjectKind,
			"--subject-name", subjectName,
		})

		err := cmd.Execute()

		bnd := retrieveCreatedBinding(t, err, vSphereClientSet, command.DefaultNamespace, bindingName)
		assertBasicBinding(t, &bnd.Spec, bindingAddress, secretRef, false)
		assertSubject(t, &bnd.Spec.Subject,
			subjectAPIVersion, subjectKind, command.DefaultNamespace, subjectName, defaultSelector())
	})

	t.Run("creates insecure binding in explicit namespace", func(t *testing.T) {
		var (
			namespace         = "ns"
			subjectAPIVersion = "apps/v1"
			subjectKind       = "Deployment"
			subjectName       = "my-simple-app"
			skipTLSVerify     = true
		)

		cmd, vSphereClientSet := bindingTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--namespace", namespace,
			"--name", bindingName,
			"--vc-address", bindingAddress,
			"--skip-tls-verify", strconv.FormatBool(skipTLSVerify),
			"--secret-ref", secretRef,
			"--subject-api-version", subjectAPIVersion,
			"--subject-kind", subjectKind,
			"--subject-name", subjectName,
		})

		err := cmd.Execute()

		bnd := retrieveCreatedBinding(t, err, vSphereClientSet, namespace, bindingName)
		assertBasicBinding(t, &bnd.Spec, bindingAddress, secretRef, skipTLSVerify)
		assertSubject(t, &bnd.Spec.Subject,
			subjectAPIVersion, subjectKind, namespace, subjectName, defaultSelector())
	})

	t.Run("creates binding with subject label selector in default namespace", func(t *testing.T) {
		var (
			subjectAPIVersion = "apps/v1"
			subjectKind       = "Deployment"
			labelName         = "foo"
			labelValue        = "bar"
			skipTLSVerify     = true
		)

		cmd, vSphereClientSet := bindingTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--namespace", command.DefaultNamespace,
			"--name", bindingName,
			"--vc-address", bindingAddress,
			"--skip-tls-verify", strconv.FormatBool(skipTLSVerify),
			"--secret-ref", secretRef,
			"--subject-api-version", subjectAPIVersion,
			"--subject-kind", subjectKind,
			"--subject-kind", subjectKind,
			"--subject-selector", fmt.Sprintf("%s=%s", labelName, labelValue),
		})

		err := cmd.Execute()

		bnd := retrieveCreatedBinding(t, err, vSphereClientSet, command.DefaultNamespace, bindingName)
		assertBasicBinding(t, &bnd.Spec, bindingAddress, secretRef, skipTLSVerify)
		assertSubject(t, &bnd.Spec.Subject,
			subjectAPIVersion, subjectKind, command.DefaultNamespace, "", &metav1.LabelSelector{
				MatchLabels:      map[string]string{labelName: labelValue},
				MatchExpressions: []metav1.LabelSelectorRequirement{},
			})
	})

	t.Run("fails to execute when default namespace retrieval fails", func(t *testing.T) {
		namespaceError := fmt.Errorf("no default namespace, oops")
		cmd, _ := bindingTestCommand(command.FailingClientConfig(namespaceError))
		cmd.SetArgs([]string{
			"create",
			"--name", bindingName,
			"--vc-address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
			"--subject-name", "my-simple-app",
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "failed to get namespace")
	})

	t.Run("fails to execute with an invalid source URI", func(t *testing.T) {
		invalidBindingAddress := "more cow\x07"

		cmd, _ := bindingTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--name", bindingName,
			"--vc-address", invalidBindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
			"--subject-name", "my-simple-app",
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "failed to parse binding address")
	})

	t.Run("fails to execute with an invalid subject selector", func(t *testing.T) {
		invalidSelector := "not a selector"

		cmd, _ := bindingTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--name", bindingName,
			"--vc-address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
			"--subject-selector", invalidSelector,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "failed to parse subject selector")
	})

	t.Run("fails to execute when the binding creation fails", func(t *testing.T) {
		bindingCreationErrorMsg := "cannot create binding"

		cmd, vSphereSourcesClient := bindingTestCommand(command.RegularClientConfig())
		vSphereSourcesClient.PrependReactor("create", "vspherebindings", func(a k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, fmt.Errorf(bindingCreationErrorMsg)
		})

		cmd.SetArgs([]string{
			"create",
			"--name", bindingName,
			"--vc-address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
			"--subject-name", "my-simple-app",
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, fmt.Sprintf("failed to create Binding: %s", bindingCreationErrorMsg))
	})

	t.Run("fails to execute when trying to create a duplicate binding", func(t *testing.T) {
		var (
			subjectAPIVersion = "apps/v1"
			subjectKind       = "Deployment"
			subjectName       = "my-simple-app"
		)

		existingBinding := newBinding(t, command.DefaultNamespace, bindingName, bindingAddress, secretRef, subjectAPIVersion, subjectKind, subjectName)
		cmd, _ := bindingTestCommand(command.RegularClientConfig(), existingBinding)
		cmd.SetArgs([]string{
			"create",
			"--name", bindingName,
			"--vc-address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", subjectAPIVersion,
			"--subject-kind", subjectKind,
			"--subject-name", subjectName,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, fmt.Sprintf(`"%s" already exists`, bindingName))
	})
}

func retrieveCreatedBinding(t *testing.T, err error, vSphereClientSet vsphere.Interface, namespace, sourceName string) *v1alpha1.VSphereBinding {
	assert.NilError(t, err)
	source, err := vSphereClientSet.SourcesV1alpha1().
		VSphereBindings(namespace).
		Get(context.Background(), sourceName, metav1.GetOptions{})
	assert.NilError(t, err)
	return source
}

func assertBasicBinding(t *testing.T, bindingSpec *v1alpha1.VSphereBindingSpec, bindingAddress string, secretRef string, skipTLSVerify bool) {
	assert.Equal(t, bindingSpec.Address.String(), bindingAddress)
	assert.Equal(t, bindingSpec.SecretRef.Name, secretRef)
	assert.Check(t, bindingSpec.SkipTLSVerify == skipTLSVerify)
}

func defaultSelector() *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels:      map[string]string{},
		MatchExpressions: []metav1.LabelSelectorRequirement{},
	}
}

func assertSubject(t *testing.T, subject *tracker.Reference, apiVersion, kind, namespace, name string, selector *metav1.LabelSelector) {
	assert.Equal(t, subject.APIVersion, apiVersion)
	assert.Equal(t, subject.Kind, kind)
	assert.Equal(t, subject.Namespace, namespace)
	assert.Equal(t, subject.Name, name)
	assert.Check(t, reflect.DeepEqual(subject.Selector, selector))
}

func newBinding(t *testing.T, namespace, name, address, secretRef, subjectAPIVersion, subjectKind, subjectName string) runtime.Object {
	bindingAddress := command.ParseURI(t, address)
	return &v1alpha1.VSphereBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: v1alpha1.VSphereBindingSpec{
			BindingSpec: duckv1alpha1.BindingSpec{
				Subject: tracker.Reference{
					APIVersion: subjectAPIVersion,
					Kind:       subjectKind,
					Namespace:  namespace,
					Name:       subjectName,
				},
			},
			VAuthSpec: v1alpha1.VAuthSpec{
				Address:       bindingAddress,
				SkipTLSVerify: false,
				SecretRef: corev1.LocalObjectReference{
					Name: secretRef,
				},
			},
		},
	}
}
