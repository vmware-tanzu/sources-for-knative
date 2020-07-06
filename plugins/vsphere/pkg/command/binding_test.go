/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package command_test

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	vsphere "github.com/vmware-tanzu/sources-for-knative/pkg/client/clientset/versioned"
	vspherefake "github.com/vmware-tanzu/sources-for-knative/pkg/client/clientset/versioned/fake"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/clientcmd"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/tracker"
)

func TestNewBindingCommand(t *testing.T) {

	const bindingName = "spring"
	const secretRef = "street-creds"
	const bindingAddress = "https://my-vsphere-endpoint.example.com"

	t.Run("defines basic metadata", func(t *testing.T) {
		bindingCommand, _ := bindingCommand(regularClientConfig())

		assert.Equal(t, bindingCommand.Use, "binding")
		assert.Check(t, len(bindingCommand.Short) > 0,
			"command should have a nonempty short description")
		assert.Check(t, len(bindingCommand.Long) > 0,
			"command should have a nonempty long description")
		checkFlag(t, bindingCommand, "namespace")
		checkFlag(t, bindingCommand, "name")
		checkFlag(t, bindingCommand, "address")
		checkFlag(t, bindingCommand, "skip-tls-verify")
		checkFlag(t, bindingCommand, "secret-ref")
		checkFlag(t, bindingCommand, "subject-api-version")
		checkFlag(t, bindingCommand, "subject-kind")
		checkFlag(t, bindingCommand, "subject-name")
		checkFlag(t, bindingCommand, "subject-selector")
		assert.Assert(t, bindingCommand.RunE != nil)
	})

	t.Run("fails to execute with an empty name", func(t *testing.T) {
		bindingCommand, _ := bindingCommand(regularClientConfig())
		bindingCommand.SetArgs([]string{
			"--address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
			"--subject-name", "my-simple-app",
		})

		err := bindingCommand.Execute()

		assert.ErrorContains(t, err, "requires a nonempty name provided with the --name option")
	})

	t.Run("fails to execute with an empty address", func(t *testing.T) {
		bindingCommand, _ := bindingCommand(regularClientConfig())
		bindingCommand.SetArgs([]string{
			"--name", bindingName,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
			"--subject-name", "my-simple-app",
		})

		err := bindingCommand.Execute()

		assert.ErrorContains(t, err, "requires a nonempty address provided with the --address option")
	})

	t.Run("fails to execute with an empty secret reference", func(t *testing.T) {
		bindingCommand, _ := bindingCommand(regularClientConfig())
		bindingCommand.SetArgs([]string{
			"--name", bindingName,
			"--address", bindingAddress,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
			"--subject-name", "my-simple-app",
		})

		err := bindingCommand.Execute()

		assert.ErrorContains(t, err, "requires a nonempty secret reference provided with the --secret-ref option")
	})

	t.Run("fails to execute with an empty subject API version", func(t *testing.T) {
		bindingCommand, _ := bindingCommand(regularClientConfig())
		bindingCommand.SetArgs([]string{
			"--name", bindingName,
			"--address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-kind", "Deployment",
			"--subject-name", "my-simple-app",
		})

		err := bindingCommand.Execute()

		assert.ErrorContains(t, err, "requires a nonempty subject API version provided with the --subject-api-version option")
	})

	t.Run("fails to execute with an empty subject kind", func(t *testing.T) {
		bindingCommand, _ := bindingCommand(regularClientConfig())
		bindingCommand.SetArgs([]string{
			"--name", bindingName,
			"--address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-name", "my-simple-app",
		})

		err := bindingCommand.Execute()

		assert.ErrorContains(t, err, "requires a nonempty subject kind provided with the --subject-kind option")
	})

	t.Run("fails to execute with an empty subject name and empty selector", func(t *testing.T) {
		bindingCommand, _ := bindingCommand(regularClientConfig())
		bindingCommand.SetArgs([]string{
			"--name", bindingName,
			"--address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
		})

		err := bindingCommand.Execute()

		assert.ErrorContains(t, err, `requires a nonempty subject name provided with the --subject-name option,
or a nonempty subject selector with the --subject-selector option`)
	})

	t.Run("fails to execute with both subject name and selector set", func(t *testing.T) {
		bindingCommand, _ := bindingCommand(regularClientConfig())
		bindingCommand.SetArgs([]string{
			"--name", bindingName,
			"--address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
			"--subject-name", "my-simple-app",
			"--subject-selector", "foo:bar",
		})

		err := bindingCommand.Execute()

		assert.ErrorContains(t, err, `subject can optionally be configured with one of the following flags (but several were set):
	--subject-name, --subject-selector`)
	})

	t.Run("creates basic binding in default namespace", func(t *testing.T) {
		bindingCommand, vSphereClientSet := bindingCommand(regularClientConfig())
		subjectApiVersion := "apps/v1"
		subjectKind := "Deployment"
		subjectName := "my-simple-app"
		bindingCommand.SetArgs([]string{
			"--name", bindingName,
			"--address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", subjectApiVersion,
			"--subject-kind", subjectKind,
			"--subject-name", subjectName,
		})

		err := bindingCommand.Execute()

		binding := retrieveCreatedBinding(t, err, vSphereClientSet, defaultNamespace, bindingName)
		assertBasicBinding(t, &binding.Spec, bindingAddress, secretRef, false)
		assertSubject(t, &binding.Spec.Subject,
			subjectApiVersion, subjectKind, defaultNamespace, subjectName, defaultSelector())
	})

	t.Run("creates insecure binding in explicit namespace", func(t *testing.T) {
		namespace := "ns"
		bindingCommand, vSphereClientSet := bindingCommand(regularClientConfig())
		subjectApiVersion := "apps/v1"
		subjectKind := "Deployment"
		subjectName := "my-simple-app"
		skipTlsVerify := true
		bindingCommand.SetArgs([]string{
			"--namespace", namespace,
			"--name", bindingName,
			"--address", bindingAddress,
			"--skip-tls-verify", strconv.FormatBool(skipTlsVerify),
			"--secret-ref", secretRef,
			"--subject-api-version", subjectApiVersion,
			"--subject-kind", subjectKind,
			"--subject-name", subjectName,
		})

		err := bindingCommand.Execute()

		binding := retrieveCreatedBinding(t, err, vSphereClientSet, namespace, bindingName)
		assertBasicBinding(t, &binding.Spec, bindingAddress, secretRef, skipTlsVerify)
		assertSubject(t, &binding.Spec.Subject,
			subjectApiVersion, subjectKind, namespace, subjectName, defaultSelector())
	})

	t.Run("creates binding with subject label selector in default namespace", func(t *testing.T) {
		bindingCommand, vSphereClientSet := bindingCommand(regularClientConfig())
		subjectApiVersion := "apps/v1"
		subjectKind := "Deployment"
		labelName := "foo"
		labelValue := "bar"
		skipTlsVerify := true
		bindingCommand.SetArgs([]string{
			"--namespace", defaultNamespace,
			"--name", bindingName,
			"--address", bindingAddress,
			"--skip-tls-verify", strconv.FormatBool(skipTlsVerify),
			"--secret-ref", secretRef,
			"--subject-api-version", subjectApiVersion,
			"--subject-kind", subjectKind,
			"--subject-kind", subjectKind,
			"--subject-selector", fmt.Sprintf("%s=%s", labelName, labelValue),
		})

		err := bindingCommand.Execute()

		binding := retrieveCreatedBinding(t, err, vSphereClientSet, defaultNamespace, bindingName)
		assertBasicBinding(t, &binding.Spec, bindingAddress, secretRef, skipTlsVerify)
		assertSubject(t, &binding.Spec.Subject,
			subjectApiVersion, subjectKind, defaultNamespace, "", &metav1.LabelSelector{
				MatchLabels:      map[string]string{labelName: labelValue},
				MatchExpressions: []metav1.LabelSelectorRequirement{},
			})
	})

	t.Run("fails to execute when default namespace retrieval fails", func(t *testing.T) {
		namespaceError := fmt.Errorf("no default namespace, oops")
		bindingCommand, _ := bindingCommand(failingClientConfig(namespaceError))
		bindingCommand.SetArgs([]string{
			"--name", bindingName,
			"--address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
			"--subject-name", "my-simple-app",
		})

		err := bindingCommand.Execute()

		assert.ErrorContains(t, err, "failed to get namespace")
	})

	t.Run("fails to execute with an invalid source URI", func(t *testing.T) {
		invalidBindingAddress := "more cow\x07"
		bindingCommand, _ := bindingCommand(regularClientConfig())
		bindingCommand.SetArgs([]string{
			"--name", bindingName,
			"--address", invalidBindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
			"--subject-name", "my-simple-app",
		})

		err := bindingCommand.Execute()

		assert.ErrorContains(t, err, "failed to parse binding address")
	})

	t.Run("fails to execute with an invalid subject selector", func(t *testing.T) {
		invalidSelector := "not a selector"
		bindingCommand, _ := bindingCommand(regularClientConfig())
		bindingCommand.SetArgs([]string{
			"--name", bindingName,
			"--address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
			"--subject-selector", invalidSelector,
		})

		err := bindingCommand.Execute()

		assert.ErrorContains(t, err, "failed to parse subject selector")
	})

	t.Run("fails to execute when the binding creation fails", func(t *testing.T) {
		bindingCreationErrorMsg := "cannot create binding"
		bindingCommand, vSphereSourcesClient := bindingCommand(regularClientConfig())
		vSphereSourcesClient.PrependReactor("create", "vspherebindings", func(a k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, fmt.Errorf(bindingCreationErrorMsg)
		})
		bindingCommand.SetArgs([]string{
			"--name", bindingName,
			"--address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", "apps/v1",
			"--subject-kind", "Deployment",
			"--subject-name", "my-simple-app",
		})

		err := bindingCommand.Execute()

		assert.ErrorContains(t, err, fmt.Sprintf("failed to create Binding: %s", bindingCreationErrorMsg))
	})

	t.Run("fails to execute when trying to create a duplicate binding", func(t *testing.T) {
		subjectApiVersion := "apps/v1"
		subjectKind := "Deployment"
		subjectName := "my-simple-app"
		existingBinding := newBinding(t, defaultNamespace, bindingName, bindingAddress, secretRef, subjectApiVersion, subjectKind, subjectName)
		bindingCommand, _ := bindingCommand(regularClientConfig(), existingBinding)
		bindingCommand.SetArgs([]string{
			"--name", bindingName,
			"--address", bindingAddress,
			"--secret-ref", secretRef,
			"--subject-api-version", subjectApiVersion,
			"--subject-kind", subjectKind,
			"--subject-name", subjectName,
		})

		err := bindingCommand.Execute()

		assert.ErrorContains(t, err, fmt.Sprintf(`"%s" already exists`, bindingName))
	})
}

func bindingCommand(clientConfig clientcmd.ClientConfig, objects ...runtime.Object) (*cobra.Command, *vspherefake.Clientset) {
	vSphereSourcesClient := vspherefake.NewSimpleClientset(objects...)
	bindingCommand := command.NewBindingCommand(&pkg.Clients{
		ClientSet:        k8sfake.NewSimpleClientset(),
		ClientConfig:     clientConfig,
		VSphereClientSet: vSphereSourcesClient,
	})
	bindingCommand.SetErr(ioutil.Discard)
	bindingCommand.SetOut(ioutil.Discard)
	return bindingCommand, vSphereSourcesClient
}

func retrieveCreatedBinding(t *testing.T, err error, vSphereClientSet vsphere.Interface, namespace, sourceName string) *v1alpha1.VSphereBinding {
	assert.NilError(t, err)
	source, err := vSphereClientSet.SourcesV1alpha1().
		VSphereBindings(namespace).
		Get(sourceName, metav1.GetOptions{})
	assert.NilError(t, err)
	return source
}

func assertBasicBinding(t *testing.T, bindingSpec *v1alpha1.VSphereBindingSpec, bindingAddress string, secretRef string, skipTlsVerify bool) {
	assert.Equal(t, bindingSpec.Address.String(), bindingAddress)
	assert.Equal(t, bindingSpec.SecretRef.Name, secretRef)
	assert.Check(t, bindingSpec.SkipTLSVerify == skipTlsVerify)
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

func newBinding(t *testing.T, namespace, name, address, secretRef, subjectApiVersion, subjectKind, subjectName string) runtime.Object {
	bindingAddress := parseUri(t, address)
	return &v1alpha1.VSphereBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: v1alpha1.VSphereBindingSpec{
			BindingSpec: duckv1alpha1.BindingSpec{
				Subject: tracker.Reference{
					APIVersion: subjectApiVersion,
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
