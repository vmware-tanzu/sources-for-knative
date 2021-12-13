/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package binding

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/flags"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/tracker"
)

func NewBindingCreateCommand(clients *pkg.Clients, opts *Options) *cobra.Command {
	result := cobra.Command{
		Use:   "create",
		Short: "Create a vSphere binding to call into the vSphere API",
		Long:  "Create a vSphere binding to call into the vSphere API",
		Example: `# Create the binding in the default namespace, targeting a Deployment subject
kn vsphere binding create --name vc-binding --vc-address https://my-vsphere-endpoint.
local --skip-tls-verify --secret-ref vsphere-credentials --subject-api-version apps/v1 --subject-kind Deployment --subject-name my-simple-app

# Create the binding in the specified namespace, targeting a selection of Job subjects
kn vsphere binding create --namespace ns --name vc-binding --vc-address https://my-vsphere-endpoint.local --skip-tls-verify --secret-ref vsphere-credentials --subject-api-version batch/v1 --subject-kind Job --subject-selector foo=bar
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
			if opts.SubjectAPIVersion == "" {
				return fmt.Errorf("'subject-api-version' requires a nonempty subject API version provided with the --subject-api-version option")
			}
			if opts.SubjectKind == "" {
				return fmt.Errorf("'subject-kind' requires a nonempty subject kind provided with the --subject-kind option")
			}
			subjectName := opts.SubjectName
			subjectSelector := opts.SubjectSelector
			if subjectName == "" && subjectSelector == "" {
				return fmt.Errorf("subject requires a nonempty subject name provided with the --subject-name option," +
					"\nor a nonempty subject selector with the --subject-selector option")
			}
			if !flags.MutuallyExclusiveStringFlags(subjectName, subjectSelector) {
				return fmt.Errorf("subject can optionally be configured with one of the following flags (but several were set):\n\t" +
					"--subject-name, --subject-selector")
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
				return fmt.Errorf("failed to parse binding address: %v", err)
			}
			selector, err := metav1.ParseToLabelSelector(opts.SubjectSelector)
			if err != nil {
				return fmt.Errorf("failed to parse subject selector: %v", err)
			}
			if _, err = clients.VSphereClientSet.
				SourcesV1alpha1().
				VSphereBindings(namespace).
				Create(cmd.Context(), newBinding(namespace, address, selector, *opts), metav1.CreateOptions{}); err != nil {
				return fmt.Errorf("failed to create Binding: %v", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Created binding")
			return nil
		},
	}

	fl := result.Flags()
	fl.StringVar(&opts.Name, "name", "", "name of the binding to create")
	fl.StringVarP(&opts.VCAddress, "vc-address", "a", "", "URL of vCenter instance to associate the binding with")
	fl.BoolVarP(&opts.SkipTLSVerify, "skip-tls-verify", "k", false, "disables certificate verification for the binding API address")
	fl.StringVarP(&opts.SecretRef, "secret-ref", "s", "", "reference to the Kubernetes secret for the vSphere credentials needed for the binding API address")
	fl.StringVar(&opts.SubjectAPIVersion, "subject-api-version", "", "subject API version")
	fl.StringVar(&opts.SubjectKind, "subject-kind", "", "subject kind")
	fl.StringVar(&opts.SubjectName, "subject-name", "", "subject name (cannot be used with --subject-selector)")
	fl.StringVar(&opts.SubjectSelector, "subject-selector", "", "subject selector (cannot be used with --subject-name)")

	_ = result.MarkFlagRequired("name")
	_ = result.MarkFlagRequired("vc-address")
	_ = result.MarkFlagRequired("secret-ref")
	_ = result.MarkFlagRequired("subject-api-version")
	_ = result.MarkFlagRequired("subject-kind")

	return &result
}

func newBinding(namespace string, address *url.URL, selector *metav1.LabelSelector, options Options) *v1alpha1.VSphereBinding {
	return &v1alpha1.VSphereBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      options.Name,
		},
		Spec: v1alpha1.VSphereBindingSpec{
			BindingSpec: duckv1alpha1.BindingSpec{
				Subject: tracker.Reference{
					APIVersion: options.SubjectAPIVersion,
					Kind:       options.SubjectKind,
					Namespace:  namespace,
					Name:       options.SubjectName,
					Selector:   selector,
				},
			},
			VAuthSpec: v1alpha1.VAuthSpec{
				Address:       apis.URL(*address),
				SkipTLSVerify: options.SkipTLSVerify,
				SecretRef: corev1.LocalObjectReference{
					Name: options.SecretRef,
				},
			},
		},
	}
}
