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
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/tracker"
)

type BindingOptions struct {
	Namespace     string
	Name          string
	Address       string
	SkipTlsVerify bool
	SecretRef     string

	SubjectApiVersion string
	SubjectKind       string
	SubjectName       string
	SubjectSelector   string
}

func NewBindingCommand(clients *pkg.Clients) *cobra.Command {
	options := BindingOptions{}

	result := cobra.Command{
		Use:   "binding",
		Short: "Create a vSphere binding to call into the vSphere API",
		Long:  "Create a vSphere binding to call into the vSphere API",
		Example: `# Create the binding in the default namespace, targeting a Deployment subject
kn vsphere binding --name binding --address https://my-vsphere-endpoint.local --skip-tls-verify --secret-ref vsphere-credentials --subject-api-version app/v1 --subject-kind Deployment --subject-name my-simple-app
# Create the binding in the specified namespace, targeting a selection of Job subjects
kn vsphere binding --namespace ns --name source --address https://my-vsphere-endpoint.local --skip-tls-verify --secret-ref vsphere-credentials --subject-api-version batch/v1 --subject-kind Job --subject-selector foo=bar
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
			if options.SubjectApiVersion == "" {
				return fmt.Errorf("'subject-api-version' requires a nonempty subject API version provided with the --subject-api-version option")
			}
			if options.SubjectKind == "" {
				return fmt.Errorf("'subject-kind' requires a nonempty subject kind provided with the --subject-kind option")
			}
			subjectName := options.SubjectName
			subjectSelector := options.SubjectSelector
			if subjectName == "" && subjectSelector == "" {
				return fmt.Errorf("subject requires a nonempty subject name provided with the --subject-name option," +
					"\nor a nonempty subject selector with the --subject-selector option")
			}
			if !MutuallyExclusiveStringFlags(subjectName, subjectSelector) {
				return fmt.Errorf("subject can optionally be configured with one of the following flags (but several were set):\n\t" +
					"--subject-name, --subject-selector")
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
				return fmt.Errorf("failed to parse binding address: %+v", err)
			}
			selector, err := metav1.ParseToLabelSelector(options.SubjectSelector)
			if err != nil {
				return fmt.Errorf("failed to parse subject selector: %+v", err)
			}
			if _, err := clients.VSphereClientSet.
				SourcesV1alpha1().
				VSphereBindings(namespace).
				Create(newBinding(namespace, address, selector, options)); err != nil {
				return fmt.Errorf("failed to create Binding: %+v", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Created binding")
			return nil
		},
	}

	flags := result.Flags()
	flags.StringVarP(&options.Namespace, "namespace", "n", "", "namespace of the binding to create (default namespace if omitted)")
	flags.StringVar(&options.Name, "name", "", "name of the binding to create")
	_ = result.MarkFlagRequired("name")
	flags.StringVarP(&options.Address, "address", "a", "", "URL of the events to fetch")
	_ = result.MarkFlagRequired("address")
	flags.BoolVarP(&options.SkipTlsVerify, "skip-tls-verify", "k", false, "disables certificate verification for the source address (same as GOVC_INSECURE)")
	flags.StringVarP(&options.SecretRef, "secret-ref", "s", "", "reference to the Kubernetes secret for the vSphere credentials needed for the source address")
	_ = result.MarkFlagRequired("secret-ref")
	flags.StringVar(&options.SubjectApiVersion, "subject-api-version", "", "subject API version")
	_ = result.MarkFlagRequired("subject-api-version")
	flags.StringVar(&options.SubjectKind, "subject-kind", "", "subject kind")
	_ = result.MarkFlagRequired("subject-kind")
	flags.StringVar(&options.SubjectName, "subject-name", "", "subject name (cannot be used with --subject-selector)")
	flags.StringVar(&options.SubjectSelector, "subject-selector", "", "subject selector (cannot be used with --subject-name)")
	return &result
}

func newBinding(namespace string, address *url.URL, selector *metav1.LabelSelector, options BindingOptions) *v1alpha1.VSphereBinding {
	return &v1alpha1.VSphereBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      options.Name,
		},
		Spec: v1alpha1.VSphereBindingSpec{
			BindingSpec: duckv1alpha1.BindingSpec{
				Subject: tracker.Reference{
					APIVersion: options.SubjectApiVersion,
					Kind:       options.SubjectKind,
					Namespace:  namespace,
					Name:       options.SubjectName,
					Selector:   selector,
				},
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
