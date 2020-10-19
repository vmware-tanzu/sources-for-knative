/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package command_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
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
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestNewSourceCommand(t *testing.T) {

	const sourceName = "spring"
	const secretRef = "street-creds"
	const sourceAddress = "https://my-vsphere-endpoint.example.com"
	const sinkURI = "https://sink.example.com"

	t.Run("defines basic metadata", func(t *testing.T) {
		sourceCommand, _ := sourceCommand(regularClientConfig())

		assert.Equal(t, sourceCommand.Use, "source")
		assert.Check(t, len(sourceCommand.Short) > 0,
			"command should have a nonempty short description")
		assert.Check(t, len(sourceCommand.Long) > 0,
			"command should have a nonempty long description")
		checkFlag(t, sourceCommand, "namespace")
		checkFlag(t, sourceCommand, "name")
		checkFlag(t, sourceCommand, "address")
		checkFlag(t, sourceCommand, "skip-tls-verify")
		checkFlag(t, sourceCommand, "secret-ref")
		checkFlag(t, sourceCommand, "sink-uri")
		checkFlag(t, sourceCommand, "sink-api-version")
		checkFlag(t, sourceCommand, "sink-kind")
		checkFlag(t, sourceCommand, "sink-name")
		assert.Assert(t, sourceCommand.RunE != nil)
	})

	t.Run("fails to execute with an empty name", func(t *testing.T) {
		sourceCommand, _ := sourceCommand(regularClientConfig())
		sourceCommand.SetArgs([]string{
			"--address", sourceAddress,
			"--secret-ref", secretRef,
			"--sink-uri", sinkURI,
		})

		err := sourceCommand.Execute()

		assert.ErrorContains(t, err, "requires a nonempty name provided with the --name option")
	})

	t.Run("fails to execute with an empty address", func(t *testing.T) {
		sourceCommand, _ := sourceCommand(regularClientConfig())
		sourceCommand.SetArgs([]string{
			"--name", sourceName,
			"--secret-ref", secretRef,
			"--sink-uri", sinkURI,
		})

		err := sourceCommand.Execute()

		assert.ErrorContains(t, err, "requires a nonempty address provided with the --address option")
	})

	t.Run("fails to execute with an empty secret reference", func(t *testing.T) {
		sourceCommand, _ := sourceCommand(regularClientConfig())
		sourceCommand.SetArgs([]string{
			"--name", sourceName,
			"--address", sourceAddress,
			"--sink-uri", sinkURI,
		})

		err := sourceCommand.Execute()

		assert.ErrorContains(t, err, "requires a nonempty secret reference provided with the --secret-ref option")
	})

	invalidSinkMatrix := []struct {
		description string
		args        []string
	}{
		{
			"no", []string{},
		},
		{"all but name", []string{
			"--sink-uri", sinkURI,
			"--sink-api-version", "some-api-version",
			"--sink-kind", "some-kind"},
		},
		{"all but kind", []string{
			"--sink-uri", sinkURI,
			"--sink-api-version", "some-api-version",
			"--sink-name", "some-name"},
		},
		{"all but API version", []string{
			"--sink-uri", sinkURI,
			"--sink-kind", "some-kind",
			"--sink-name", "some-name"},
		},
		{"only API version and kind", []string{
			"--sink-api-version", "some-api-version",
			"--sink-kind", "some-kind"},
		},
		{"only API version and name", []string{
			"--sink-api-version", "some-api-version",
			"--sink-name", "some-name"},
		},
		{"only kind and name", []string{
			"--sink-kind", "some-kind",
			"--sink-name", "some-name"},
		},
		{"only API version", []string{
			"--sink-api-version", "some-api-version"},
		},
		{"only kind", []string{
			"--sink-kind", "some-kind"},
		},
		{"only name", []string{
			"--sink-name", "some-name"},
		},
	}
	for _, sinkTestCase := range invalidSinkMatrix {
		t.Run(fmt.Sprintf("fails to execute with %s sink flags set", sinkTestCase.description), func(t *testing.T) {
			sourceCommand, _ := sourceCommand(regularClientConfig())
			sourceCommand.SetArgs(append([]string{
				"--name", sourceName,
				"--address", sourceAddress,
				"--secret-ref", secretRef,
			}, sinkTestCase.args...))

			err := sourceCommand.Execute()

			assert.ErrorContains(t, err, `sink requires an URI
and/or a nonempty API version --sink-api-version option,
with a nonempty kind --sink-kind option,
and with a nonempty name with the --sink-name`)
		})
	}

	t.Run("creates basic source with sink URI in default namespace", func(t *testing.T) {
		sourceCommand, vSphereClientSet := sourceCommand(regularClientConfig())
		sourceCommand.SetArgs([]string{
			"--name", sourceName,
			"--address", sourceAddress,
			"--secret-ref", secretRef,
			"--sink-uri", sinkURI,
		})

		err := sourceCommand.Execute()

		source := retrieveCreatedSource(t, err, vSphereClientSet, defaultNamespace, sourceName)
		assertBasicSource(t, &source.Spec, sourceAddress, secretRef, false)
		assert.Equal(t, source.Spec.Sink.URI.String(), sinkURI)
		assert.Check(t, source.Spec.Sink.Ref == nil)
	})

	t.Run("creates insecure source with Service and relative sink URI in explicit namespace", func(t *testing.T) {
		namespace := "ns"
		sinkURI := "/relative/uri"
		sourceCommand, vSphereClientSet := sourceCommand(regularClientConfig())
		skipTLSVerify := true
		sinkAPIVersion := "v1"
		sinkKind := "Service"
		sinkName := "some-service"
		sourceCommand.SetArgs([]string{
			"--namespace", namespace,
			"--name", sourceName,
			"--address", sourceAddress,
			"--skip-tls-verify", strconv.FormatBool(skipTLSVerify),
			"--secret-ref", secretRef,
			"--sink-uri", sinkURI,
			"--sink-api-version", sinkAPIVersion,
			"--sink-kind", sinkKind,
			"--sink-name", sinkName,
		})

		err := sourceCommand.Execute()

		source := retrieveCreatedSource(t, err, vSphereClientSet, namespace, sourceName)
		assertBasicSource(t, &source.Spec, sourceAddress, secretRef, skipTLSVerify)
		assert.Equal(t, source.Spec.Sink.URI.String(), sinkURI)
		assertSinkReference(t, source.Spec.Sink.Ref, sinkAPIVersion, sinkKind, namespace, sinkName)
	})

	t.Run("fails to execute when default namespace retrieval fails", func(t *testing.T) {
		namespaceError := fmt.Errorf("no default namespace, oops")
		sourceCommand, _ := sourceCommand(failingClientConfig(namespaceError))
		sourceCommand.SetArgs([]string{
			"--name", sourceName,
			"--address", sourceAddress,
			"--secret-ref", secretRef,
			"--sink-uri", sinkURI,
		})

		err := sourceCommand.Execute()

		assert.ErrorContains(t, err, "failed to get namespace")
	})

	t.Run("fails to execute with an invalid source URI", func(t *testing.T) {
		invalidSourceAddress := "more cow\x07"
		sourceCommand, _ := sourceCommand(regularClientConfig())
		sourceCommand.SetArgs([]string{
			"--name", sourceName,
			"--address", invalidSourceAddress,
			"--secret-ref", secretRef,
			"--sink-uri", sinkURI,
		})

		err := sourceCommand.Execute()

		assert.ErrorContains(t, err, "invalid control character in URL")
	})

	t.Run("fails to execute with an invalid source URI", func(t *testing.T) {
		invalidSinkAddress := "more cow\x07"
		sourceCommand, _ := sourceCommand(regularClientConfig())
		sourceCommand.SetArgs([]string{
			"--name", sourceName,
			"--address", sourceAddress,
			"--secret-ref", secretRef,
			"--sink-uri", invalidSinkAddress,
		})

		err := sourceCommand.Execute()

		assert.ErrorContains(t, err, "invalid control character in URL")
	})

	t.Run("fails to execute when the source creation fails", func(t *testing.T) {
		sourceCreationErrorMsg := "cannot create source"
		sourceCommand, vSphereSourcesClient := sourceCommand(regularClientConfig())
		vSphereSourcesClient.PrependReactor("create", "vspheresources", func(a k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, fmt.Errorf(sourceCreationErrorMsg)
		})
		sourceCommand.SetArgs([]string{
			"--name", sourceName,
			"--address", sourceAddress,
			"--secret-ref", secretRef,
			"--sink-uri", sinkURI,
		})

		err := sourceCommand.Execute()

		assert.ErrorContains(t, err, fmt.Sprintf("failed to create source: %s", sourceCreationErrorMsg))
	})

	t.Run("fails to execute when trying to create a duplicate source", func(t *testing.T) {
		existingSource := newSource(t, defaultNamespace, sourceName, sourceAddress, secretRef, sinkURI)
		sourceCommand, _ := sourceCommand(regularClientConfig(), existingSource)
		sourceCommand.SetArgs([]string{
			"--name", sourceName,
			"--address", sourceAddress,
			"--secret-ref", secretRef,
			"--sink-uri", sinkURI,
		})

		err := sourceCommand.Execute()

		assert.ErrorContains(t, err, fmt.Sprintf(`"%s" already exists`, sourceName))
	})
}

func sourceCommand(clientConfig clientcmd.ClientConfig, objects ...runtime.Object) (*cobra.Command, *vspherefake.Clientset) {
	vSphereSourcesClient := vspherefake.NewSimpleClientset(objects...)
	sourceCommand := command.NewSourceCommand(&pkg.Clients{
		ClientSet:        k8sfake.NewSimpleClientset(),
		ClientConfig:     clientConfig,
		VSphereClientSet: vSphereSourcesClient,
	})
	sourceCommand.SetErr(ioutil.Discard)
	sourceCommand.SetOut(ioutil.Discard)
	return sourceCommand, vSphereSourcesClient
}

func retrieveCreatedSource(t *testing.T, err error, vSphereClientSet vsphere.Interface, namespace, sourceName string) *v1alpha1.VSphereSource {
	assert.NilError(t, err)
	source, err := vSphereClientSet.SourcesV1alpha1().
		VSphereSources(namespace).
		Get(context.Background(), sourceName, metav1.GetOptions{})
	assert.NilError(t, err)
	return source
}

func assertBasicSource(t *testing.T, sourceSpec *v1alpha1.VSphereSourceSpec, sourceAddress string, secretRef string, skipTLSVerify bool) {
	assert.Equal(t, sourceSpec.Address.String(), sourceAddress)
	assert.Equal(t, sourceSpec.SecretRef.Name, secretRef)
	assert.Check(t, sourceSpec.SkipTLSVerify == skipTLSVerify)
}

func assertSinkReference(t *testing.T, sinkRef *duckv1.KReference, apiVersion, kind, namespace, name string) {
	assert.Equal(t, sinkRef.APIVersion, apiVersion)
	assert.Equal(t, sinkRef.Kind, kind)
	assert.Equal(t, sinkRef.Namespace, namespace)
	assert.Equal(t, sinkRef.Name, name)
}

func newSource(t *testing.T, namespace, name, address, secretRef, sinkURI string) runtime.Object {
	sourceAddress := parseURI(t, address)
	return &v1alpha1.VSphereSource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: v1alpha1.VSphereSourceSpec{
			SourceSpec: duckv1.SourceSpec{
				Sink: duckv1.Destination{
					URI: &sourceAddress,
				},
			},
			VAuthSpec: v1alpha1.VAuthSpec{
				Address:       parseURI(t, sinkURI),
				SkipTLSVerify: false,
				SecretRef: corev1.LocalObjectReference{
					Name: secretRef,
				},
			},
		},
	}
}

func parseURI(t *testing.T, uri string) apis.URL {
	result, err := url.Parse(uri)
	assert.NilError(t, err)
	return apis.URL(*result)
}
