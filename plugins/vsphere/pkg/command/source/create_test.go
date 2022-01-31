/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package source_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	vsphere "github.com/vmware-tanzu/sources-for-knative/pkg/client/clientset/versioned"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/source"
)

func TestNewSourceCreateCommand(t *testing.T) {
	const (
		sourceName    = "spring"
		secretRef     = "street-creds"
		sourceAddress = "https://my-vsphere-endpoint.example.com"
		sinkURI       = "https://sink.example.com"
	)

	t.Run("defines basic metadata", func(t *testing.T) {
		cmd := source.NewSourceCreateCommand(&pkg.Clients{}, &source.Options{})

		assert.Equal(t, cmd.Use, "create")
		assert.Check(t, len(cmd.Short) > 0,
			"command should have a nonempty short description")
		assert.Check(t, len(cmd.Long) > 0,
			"command should have a nonempty long description")
		command.CheckFlag(t, cmd, "name")
		command.CheckFlag(t, cmd, "vc-address")
		command.CheckFlag(t, cmd, "skip-tls-verify")
		command.CheckFlag(t, cmd, "secret-ref")
		command.CheckFlag(t, cmd, "sink-uri")
		command.CheckFlag(t, cmd, "sink-api-version")
		command.CheckFlag(t, cmd, "sink-kind")
		command.CheckFlag(t, cmd, "sink-name")
		command.CheckFlag(t, cmd, "encoding")
		assert.Assert(t, cmd.RunE != nil)
	})

	t.Run("fails to execute with an empty name", func(t *testing.T) {
		cmd, _ := sourceTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--vc-address", sourceAddress,
			"--secret-ref", secretRef,
			"--sink-uri", sinkURI,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "requires a nonempty name provided with the --name option")
	})

	t.Run("fails to execute with an empty address", func(t *testing.T) {
		cmd, _ := sourceTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--name", sourceName,
			"--secret-ref", secretRef,
			"--sink-uri", sinkURI,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "requires a nonempty address provided with the --vc-address option")
	})

	t.Run("fails to execute with an empty secret reference", func(t *testing.T) {
		cmd, _ := sourceTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--name", sourceName,
			"--vc-address", sourceAddress,
			"--sink-uri", sinkURI,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "requires a nonempty secret reference provided with the --secret-ref option")
	})

	t.Run("fails to execute with an invalid encoding scheme", func(t *testing.T) {
		cmd, _ := sourceTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--name", sourceName,
			"--vc-address", sourceAddress,
			"--sink-uri", sinkURI,
			"--secret-ref", secretRef,
			"--encoding", "invalid",
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "invalid encoding scheme \"invalid\"")
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
			cmd, _ := sourceTestCommand(command.RegularClientConfig())
			cmd.SetArgs(append([]string{
				"create",
				"--name", sourceName,
				"--vc-address", sourceAddress,
				"--secret-ref", secretRef,
			}, sinkTestCase.args...))

			err := cmd.Execute()
			assert.ErrorContains(t, err, `sink requires an URI
and/or a nonempty API version --sink-api-version option,
with a nonempty kind --sink-kind option,
and with a nonempty name with the --sink-name`)
		})
	}

	t.Run("creates basic source with sink URI in default namespace", func(t *testing.T) {
		cmd, vSphereClientSet := sourceTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--name", sourceName,
			"--vc-address", sourceAddress,
			"--secret-ref", secretRef,
			"--sink-uri", sinkURI,
		})

		err := cmd.Execute()

		src := retrieveCreatedSource(t, err, vSphereClientSet, command.DefaultNamespace, sourceName)
		assertBasicSource(t, &src.Spec, sourceAddress, secretRef, false)
		assert.Equal(t, src.Spec.Sink.URI.String(), sinkURI)
		assert.Equal(t, src.Spec.PayloadEncoding, cloudevents.ApplicationXML) // assert default
		assert.Check(t, src.Spec.Sink.Ref == nil)
	})

	t.Run("creates basic source with JSON payload encoding scheme", func(t *testing.T) {
		cmd, vSphereClientSet := sourceTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--name", sourceName,
			"--vc-address", sourceAddress,
			"--secret-ref", secretRef,
			"--sink-uri", sinkURI,
			"--encoding", "JSON", // implicitly verify capitalization is ignored
		})

		err := cmd.Execute()

		src := retrieveCreatedSource(t, err, vSphereClientSet, command.DefaultNamespace, sourceName)
		assertBasicSource(t, &src.Spec, sourceAddress, secretRef, false)
		assert.Equal(t, src.Spec.PayloadEncoding, cloudevents.ApplicationJSON)
	})

	t.Run("creates insecure source with Service and relative sink URI in explicit namespace", func(t *testing.T) {
		namespace := "ns"
		sinkURI := "/relative/uri"
		cmd, vSphereClientSet := sourceTestCommand(command.RegularClientConfig())
		skipTLSVerify := true
		sinkAPIVersion := "v1"
		sinkKind := "Service"
		sinkName := "some-service"
		cmd.SetArgs([]string{
			"create",
			"--namespace", namespace,
			"--name", sourceName,
			"--vc-address", sourceAddress,
			"--skip-tls-verify", strconv.FormatBool(skipTLSVerify),
			"--secret-ref", secretRef,
			"--sink-uri", sinkURI,
			"--sink-api-version", sinkAPIVersion,
			"--sink-kind", sinkKind,
			"--sink-name", sinkName,
		})

		err := cmd.Execute()

		src := retrieveCreatedSource(t, err, vSphereClientSet, namespace, sourceName)
		assertBasicSource(t, &src.Spec, sourceAddress, secretRef, skipTLSVerify)
		assert.Equal(t, src.Spec.Sink.URI.String(), sinkURI)
		assertSinkReference(t, src.Spec.Sink.Ref, sinkAPIVersion, sinkKind, namespace, sinkName)
	})

	t.Run("fails to execute when default namespace retrieval fails", func(t *testing.T) {
		namespaceError := fmt.Errorf("no default namespace, oops")
		cmd, _ := sourceTestCommand(command.FailingClientConfig(namespaceError))
		cmd.SetArgs([]string{
			"create",
			"--name", sourceName,
			"--vc-address", sourceAddress,
			"--secret-ref", secretRef,
			"--sink-uri", sinkURI,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "failed to get namespace")
	})

	t.Run("fails to execute with an invalid source URI", func(t *testing.T) {
		invalidSourceAddress := "more cow\x07"
		cmd, _ := sourceTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--name", sourceName,
			"--vc-address", invalidSourceAddress,
			"--secret-ref", secretRef,
			"--sink-uri", sinkURI,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "invalid control character in URL")
	})

	t.Run("fails to execute with an invalid source URI", func(t *testing.T) {
		invalidSinkAddress := "more cow\x07"
		cmd, _ := sourceTestCommand(command.RegularClientConfig())
		cmd.SetArgs([]string{
			"create",
			"--name", sourceName,
			"--vc-address", sourceAddress,
			"--secret-ref", secretRef,
			"--sink-uri", invalidSinkAddress,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, "invalid control character in URL")
	})

	t.Run("fails to execute when the source creation fails", func(t *testing.T) {
		sourceCreationErrorMsg := "cannot create source"
		cmd, vSphereSourcesClient := sourceTestCommand(command.RegularClientConfig())
		vSphereSourcesClient.PrependReactor("create", "vspheresources", func(a k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, fmt.Errorf(sourceCreationErrorMsg)
		})
		cmd.SetArgs([]string{
			"create",
			"--name", sourceName,
			"--vc-address", sourceAddress,
			"--secret-ref", secretRef,
			"--sink-uri", sinkURI,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, fmt.Sprintf("failed to create source: %s", sourceCreationErrorMsg))
	})

	t.Run("fails to execute when trying to create a duplicate source", func(t *testing.T) {
		existingSource := newSource(t, command.DefaultNamespace, sourceName, sourceAddress, secretRef, sinkURI)
		cmd, _ := sourceTestCommand(command.RegularClientConfig(), existingSource)
		cmd.SetArgs([]string{
			"create",
			"--name", sourceName,
			"--vc-address", sourceAddress,
			"--secret-ref", secretRef,
			"--sink-uri", sinkURI,
		})

		err := cmd.Execute()
		assert.ErrorContains(t, err, fmt.Sprintf(`"%s" already exists`, sourceName))
	})
}

func retrieveCreatedSource(t *testing.T, err error, vSphereClientSet vsphere.Interface, namespace, sourceName string) *v1alpha1.VSphereSource {
	assert.NilError(t, err)
	src, err := vSphereClientSet.SourcesV1alpha1().
		VSphereSources(namespace).
		Get(context.Background(), sourceName, metav1.GetOptions{})
	assert.NilError(t, err)
	return src
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
	sourceAddress := command.ParseURI(t, address)
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
				Address:       command.ParseURI(t, sinkURI),
				SkipTLSVerify: false,
				SecretRef: corev1.LocalObjectReference{
					Name: secretRef,
				},
			},
		},
	}
}
