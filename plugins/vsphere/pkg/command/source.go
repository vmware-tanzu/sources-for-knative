package command

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"net/url"
)

type SourceOptions struct {
	Namespace     string
	Name          string
	Address       string
	SkipTlsVerify bool
	SecretRef     string

	SinkUri               string
	SinkServiceRef        string
	SinkKnativeServiceRef string
	SinkKnativeBrokerRef  string
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
	if so.SinkServiceRef != "" {
		return &duckv1.KReference{
			APIVersion: "v1",
			Kind:       "Service",
			Namespace:  namespace,
			Name:       so.SinkServiceRef,
		}
	}
	if so.SinkKnativeServiceRef != "" {
		return &duckv1.KReference{
			APIVersion: "serving.knative.dev/v1",
			Kind:       "Service",
			Namespace:  namespace,
			Name:       so.SinkKnativeServiceRef,
		}
	}
	if so.SinkKnativeBrokerRef != "" {
		return &duckv1.KReference{
			APIVersion: "eventing.knative.dev/v1beta1",
			Kind:       "Broker",
			Namespace:  namespace,
			Name:       so.SinkKnativeBrokerRef,
		}
	}
	return nil
}

func NewSourceCommand(clients *pkg.Clients) *cobra.Command {
	options := SourceOptions{}
	result := cobra.Command{
		Use:   "source",
		Short: "Create a vSphere source to react to vSphere events",
		Long:  "Create a vSphere source to react to vSphere events",
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
			sinkUri := options.SinkUri
			sinkServiceRef := options.SinkServiceRef
			sinkKnativeServiceRef := options.SinkKnativeServiceRef
			sinkKnativeBrokerRef := options.SinkKnativeBrokerRef
			if sinkUri == "" && sinkServiceRef == "" && sinkKnativeServiceRef == "" && sinkKnativeBrokerRef == "" {
				return fmt.Errorf("sink requires a nonempty URI provided with the --sink-uri option," +
					"\nor a non-empty reference to a Service name with the --sink-service-ref option," +
					"\nor a non-empty reference to a Knative Service name with the --sink-knative-service-ref option," +
					"\nor a non-empty reference to a Knative Broker name with the --sink-knative-broker-ref option")
			}
			if !MutuallyExclusiveStringFlags(sinkServiceRef, sinkKnativeServiceRef, sinkKnativeBrokerRef) {
				return fmt.Errorf("sink can optionally be configured with one of the following flags (but several were set):\n\t" +
					"--sink-service-ref, --sink-knative-service-ref, --sink-knative-broker-ref")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			namespace, err := clients.GetExplicitOrDefaultNamespace(options.Namespace)
			if err != nil {
				return err
			}
			address, err := url.Parse(options.Address)
			if err != nil {
				return err
			}
			sinkDestination, err := options.AsSinkDestination(namespace)
			if err != nil {
				return err
			}
			if _, err = clients.VSphereClientSet.
				SourcesV1alpha1().
				VSphereSources(namespace).
				Create(newSource(namespace, options, sinkDestination, address)); err != nil {
				return fmt.Errorf("failed to create Source: %+v", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Created source")
			return nil
		},
	}
	flags := result.Flags()
	flags.StringVarP(&options.Namespace, "namespace", "n", "", "namespace of the source to create (default namespace if omitted)")
	flags.StringVar(&options.Name, "name", "", "name of the source to create")
	flags.StringVarP(&options.Address, "address", "a", "", "URL of ESXi or vCenter instance to connect to (same as GOVC_URL)")
	_ = result.MarkFlagRequired("address")
	flags.BoolVarP(&options.SkipTlsVerify, "skip-tls-verify", "k", false, "disables certificate verification for the source address (same as GOVC_INSECURE)")
	flags.StringVarP(&options.SecretRef, "secret-ref", "s", "", "reference to the Kubernetes secret for the vSphere credentials needed for the source address")
	_ = result.MarkFlagRequired("secret-ref")
	flags.StringVarP(&options.SinkUri, "sink-uri", "u", "", "sink URI (can be absolute, or relative to the referred sink resource)")
	flags.StringVar(&options.SinkServiceRef, "sink-service-ref", "", "reference to the Kubernetes Service sink (must be in the same namespace)")
	flags.StringVar(&options.SinkKnativeServiceRef, "sink-knative-service-ref", "", "reference to the Knative Service sink (must be in the same namespace)")
	flags.StringVar(&options.SinkKnativeBrokerRef, "sink-knative-broker-ref", "", "reference to the Knative Broker sink (must be in the same namespace)")
	return &result
}

func newSource(namespace string, options SourceOptions, sinkDestination *duckv1.Destination, address *url.URL) *v1alpha1.VSphereSource {
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
