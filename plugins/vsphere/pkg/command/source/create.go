/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package source

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	"github.com/vmware-tanzu/sources-for-knative/pkg/vsphere"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
)

func NewSourceCreateCommand(clients *pkg.Clients, opts *Options) *cobra.Command {
	result := cobra.Command{
		Use:   "create",
		Short: "Create a vSphere source to react to vSphere events",
		Long:  "Create a vSphere source to react to vSphere events",
		Example: `# Create the source in the default namespace, sending events to the specified sink URI
kn vsphere source create --name vc-01-source --vc-address https://my-vsphere-endpoint.local --skip-tls-verify --secret-ref vsphere-credentials --sink-uri http://where.to.send.stuff

# Create the source in the specified namespace, sending events to the specified service
kn vsphere source create --namespace ns --name vc-01-source --vc-address https://my-vsphere-endpoint.local --skip-tls-verify --secret-ref vsphere-credentials --sink-api-version v1 --sink-kind Service --sink-name the-service-name

# Create the source in the specified namespace, sending events to the specified service with custom checkpoint behavior
kn vsphere source create --namespace ns --name vc-01-source --vc-address https://my-vsphere-endpoint.local --skip-tls-verify --secret-ref vsphere-credentials --sink-api-version v1 --sink-kind Service --sink-name the-service-name --checkpoint-age 1h --checkpoint-period 30s
`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.Name == "" {
				return fmt.Errorf("'name' requires a nonempty name provided with the --name option")
			}
			if opts.VCAddress == "" {
				return fmt.Errorf("'address' requires a nonempty address provided with the --vc-address option")
			}
			if opts.SecretRef == "" {
				return fmt.Errorf("'secret-ref' requires a nonempty secret reference provided with the --secret-ref option")
			}
			sinkCoordinatesAllEmpty := opts.SinkAPIVersion == "" && opts.SinkKind == "" && opts.SinkName == ""
			sinkCoordinatesAllSet := opts.SinkAPIVersion != "" && opts.SinkKind != "" && opts.SinkName != ""
			if opts.SinkURI == "" && sinkCoordinatesAllEmpty ||
				(!sinkCoordinatesAllEmpty && !sinkCoordinatesAllSet) {
				return fmt.Errorf("sink requires an URI" +
					"\nand/or a nonempty API version --sink-api-version option," +
					"\nwith a nonempty kind --sink-kind option," +
					"\nand with a nonempty name with the --sink-name")
			}

			// verify supported datacontentencoding schemes
			validEncodings := sets.String{
				"xml":  {},
				"json": {},
			}
			if _, ok := validEncodings[strings.ToLower(opts.PayloadEncoding)]; !ok {
				return fmt.Errorf("invalid encoding scheme %q", opts.PayloadEncoding)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			namespace, err := clients.GetExplicitOrDefaultNamespace(opts.Namespace)
			if err != nil {
				return fmt.Errorf("failed to get namespace: %v", err)
			}
			address, err := url.Parse(opts.VCAddress)
			if err != nil {
				return fmt.Errorf("failed to parse source address: %v", err)
			}
			sinkDestination, err := opts.AsSinkDestination(namespace)
			if err != nil {
				return fmt.Errorf("failed to parse sink address: %v", err)
			}
			if _, err = clients.VSphereClientSet.
				SourcesV1alpha1().
				VSphereSources(namespace).
				Create(cmd.Context(), newSource(namespace, sinkDestination, address, *opts), metav1.CreateOptions{}); err != nil {
				return fmt.Errorf("failed to create source: %v", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Created source")
			return nil
		},
	}

	flags := result.Flags()
	flags.StringVar(&opts.Name, "name", "", "name of the source to create")
	flags.StringVarP(&opts.VCAddress, "vc-address", "a", "", "URL of vCenter instance to connect to retrieve events")
	flags.BoolVarP(&opts.SkipTLSVerify, "skip-tls-verify", "k", false, "disables certificate verification for the source address")
	flags.StringVarP(&opts.SecretRef, "secret-ref", "s", "", "reference to the Kubernetes secret for the vSphere credentials needed for the source address")
	flags.StringVarP(&opts.SinkURI, "sink-uri", "u", "", "sink URI (can be absolute, or relative to the referred sink resource)")
	flags.StringVar(&opts.SinkAPIVersion, "sink-api-version", "", "sink API version")
	flags.StringVar(&opts.SinkKind, "sink-kind", "", "sink kind")
	flags.StringVar(&opts.SinkName, "sink-name", "", "sink name")
	flags.StringVar(&opts.ServiceAccountName, "service-account-name", "", "service account name")
	flags.StringVar(&opts.PayloadEncoding, "encoding", "xml", "CloudEvent data encoding scheme (xml or json)")
	flags.DurationVar(&opts.CheckpointMaxAge, "checkpoint-age", vsphere.CheckpointDefaultAge,
		"maximum allowed age for replaying events determined by last successful event in checkpoint")
	flags.DurationVar(&opts.CheckpointPeriod, "checkpoint-period", vsphere.CheckpointDefaultPeriod,
		"period between saving checkpoints")

	_ = result.MarkFlagRequired("name")
	_ = result.MarkFlagRequired("vc-address")
	_ = result.MarkFlagRequired("secret-ref")

	return &result
}

func newSource(namespace string, sinkDestination *duckv1.Destination, address *url.URL, options Options) *v1alpha1.VSphereSource {
	serviceAccountName := ""
	if options.ServiceAccountName != "" {
		serviceAccountName = options.ServiceAccountName
	}
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
				SkipTLSVerify: options.SkipTLSVerify,
				SecretRef: corev1.LocalObjectReference{
					Name: options.SecretRef,
				},
			},
			CheckpointConfig: v1alpha1.VCheckpointSpec{
				// rounding errors are ok here
				MaxAgeSeconds: int64(options.CheckpointMaxAge.Seconds()),
				PeriodSeconds: int64(options.CheckpointPeriod.Seconds()),
			},
			PayloadEncoding:    fmt.Sprintf("application/%s", strings.ToLower(options.PayloadEncoding)),
			ServiceAccountName: serviceAccountName,
		},
	}
}
