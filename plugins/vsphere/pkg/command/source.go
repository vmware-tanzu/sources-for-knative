/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package command

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

type SourceOptions struct {
	Namespace     string
	Name          string
	Address       string
	SkipTlsVerify bool
	SecretRef     string

	SinkUri        string
	SinkApiVersion string
	SinkKind       string
	SinkName       string
}

func (so *SourceOptions) AsSinkDestination(namespace string) (*duckv1.Destination, error) {
	apiUrl, err := so.sinkUrl()
	if err != nil {
		return nil, err
	}
	return &duckv1.Destination{
		Ref: so.sinkReference(namespace),
		URI: apiUrl,
	}, nil
}

func (so *SourceOptions) sinkUrl() (*apis.URL, error) {
	if so.SinkUri == "" {
		return nil, nil
	}
	address, err := url.Parse(so.SinkUri)
	if err != nil {
		return nil, err
	}
	result := apis.URL(*address)
	return &result, nil
}

func (so *SourceOptions) sinkReference(namespace string) *duckv1.KReference {
	if so.SinkApiVersion == "" {
		return nil
	}
	return &duckv1.KReference{
		APIVersion: so.SinkApiVersion,
		Kind:       so.SinkKind,
		Namespace:  namespace,
		Name:       so.SinkName,
	}
}

func NewSourceCommand(clients *pkg.Clients) *cobra.Command {
	options := SourceOptions{}
	result := cobra.Command{
		Use:   "source",
		Short: "Create a vSphere source to react to vSphere events",
		Long:  "Create a vSphere source to react to vSphere events",
		Example: `# Create the source in the default namespace, sending events to the specified sink URI
kn vsphere source --name source --address https://my-vsphere-endpoint.local --skip-tls-verify --secret-ref vsphere-credentials --sink-uri http://where.to.send.stuff
# Create the source in the specified namespace, sending events to the specified service
kn vsphere source --namespace ns --name source --address https://my-vsphere-endpoint.local --skip-tls-verify --secret-ref vsphere-credentials --sink-api-version v1 --sink-kind Service --sink-name the-service-name
`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if options.Name == "" {
				return fmt.Errorf("'name' requires a nonempty name provided with the --name option")
			}
			if options.Address == "" {
				return fmt.Errorf("'address' requires a nonempty address provided with the --address option")
			}
			if options.SecretRef == "" {
				return fmt.Errorf("'secret-ref' requires a nonempty secret reference provided with the --secret-ref option")
			}
			sinkCoordinatesAllEmpty := options.SinkApiVersion == "" && options.SinkKind == "" && options.SinkName == ""
			sinkCoordinatesAllSet := options.SinkApiVersion != "" && options.SinkKind != "" && options.SinkName != ""
			if options.SinkUri == "" && sinkCoordinatesAllEmpty ||
				(!sinkCoordinatesAllEmpty && !sinkCoordinatesAllSet) {
				return fmt.Errorf("sink requires an URI" +
					"\nand/or a nonempty API version --sink-api-version option," +
					"\nwith a nonempty kind --sink-kind option," +
					"\nand with a nonempty name with the --sink-name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			namespace, err := clients.GetExplicitOrDefaultNamespace(options.Namespace)
			if err != nil {
				return fmt.Errorf("failed to get namespace: %+v", err)
			}
			address, err := url.Parse(options.Address)
			if err != nil {
				return fmt.Errorf("failed to parse source address: %+v", err)
			}
			sinkDestination, err := options.AsSinkDestination(namespace)
			if err != nil {
				return fmt.Errorf("failed to parse sink address: %+v", err)
			}
			if _, err = clients.VSphereClientSet.
				SourcesV1alpha1().
				VSphereSources(namespace).
				Create(newSource(namespace, sinkDestination, address, options)); err != nil {
				return fmt.Errorf("failed to create source: %+v", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Created source")
			return nil
		},
	}
	flags := result.Flags()
	flags.StringVarP(&options.Namespace, "namespace", "n", "", "namespace of the source to create (default namespace if omitted)")
	flags.StringVar(&options.Name, "name", "", "name of the source to create")
	_ = result.MarkFlagRequired("name")
	flags.StringVarP(&options.Address, "address", "a", "", "URL of ESXi or vCenter instance to connect to (same as GOVC_URL)")
	_ = result.MarkFlagRequired("address")
	flags.BoolVarP(&options.SkipTlsVerify, "skip-tls-verify", "k", false, "disables certificate verification for the source address (same as GOVC_INSECURE)")
	flags.StringVarP(&options.SecretRef, "secret-ref", "s", "", "reference to the Kubernetes secret for the vSphere credentials needed for the source address")
	_ = result.MarkFlagRequired("secret-ref")
	flags.StringVarP(&options.SinkUri, "sink-uri", "u", "", "sink URI (can be absolute, or relative to the referred sink resource)")
	flags.StringVar(&options.SinkApiVersion, "sink-api-version", "", "sink API version")
	flags.StringVar(&options.SinkKind, "sink-kind", "", "sink kind")
	flags.StringVar(&options.SinkName, "sink-name", "", "sink name")
	return &result
}

func newSource(namespace string, sinkDestination *duckv1.Destination, address *url.URL, options SourceOptions) *v1alpha1.VSphereSource {
	return &v1alpha1.VSphereSource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      options.Name,
		},
		Spec: v1alpha1.VSphereSourceSpec{
			SourceSpec: duckv1.SourceSpec{
				Sink: *sinkDestination,
			},
			VAuthSpec: v1alpha1.VAuthSpec{
				Address:       apis.URL(*address),
				SkipTLSVerify: options.SkipTlsVerify,
				SecretRef: corev1.LocalObjectReference{
					Name: options.SecretRef,
				},
			},
		},
	}
}
