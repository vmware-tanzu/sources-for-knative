/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package source

import (
	"fmt"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/client/pkg/kn/commands"
	"knative.dev/client/pkg/kn/commands/flags"
	hprinters "knative.dev/client/pkg/printers"
	"knative.dev/client/pkg/util"

	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	"github.com/vmware-tanzu/sources-for-knative/pkg/client/clientset/versioned/scheme"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
)

func NewSourceListCommand(clients *pkg.Clients, opts *Options) *cobra.Command {
	sourceListFlags := flags.NewListPrintFlags(ListHandlers)

	result := cobra.Command{
		Use:     "list",
		Short:   "List vSphere sources",
		Long:    "List vSphere sources",
		Aliases: []string{"ls"},
		Example: `# List the sources in the default namespace
kn vsphere source list

# List the sources in the specified namespace
kn vsphere source list --namespace ns

# List the sources in all namespaces with JSON output
kn vsphere source list --all-namespaces -o json
`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.Namespace != "" && opts.AllNamespaces {
				return fmt.Errorf("'--namespace' and '--all-namespaces' options are mutually exclusive")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			namespace := v1.NamespaceAll

			if !opts.AllNamespaces {
				ns, err := clients.GetExplicitOrDefaultNamespace(opts.Namespace)
				if err != nil {
					return fmt.Errorf("failed to get namespace: %v", err)
				}
				namespace = ns
			}

			// empty namespace indicates all-namespaces flag is specified
			if namespace == v1.NamespaceAll {
				sourceListFlags.EnsureWithNamespace()
			}

			sourceList, err := clients.VSphereClientSet.SourcesV1alpha1().VSphereSources(namespace).List(cmd.Context(), metav1.ListOptions{})
			if err != nil {
				return fmt.Errorf("list sources: %v", err)
			}

			if !sourceListFlags.GenericPrintFlags.OutputFlagSpecified() && len(sourceList.Items) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No sources found.\n")
				return nil
			}

			listCopy := sourceList.DeepCopy()
			if err = updateSourceGVK(listCopy); err != nil {
				return fmt.Errorf("update source GKV: %v", err)
			}

			err = sourceListFlags.Print(listCopy, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return nil
		},
	}

	fl := result.Flags()
	fl.BoolVarP(&opts.AllNamespaces, "all-namespaces", "A", false, "list objects in all namespaces")

	sourceListFlags.AddFlags(&result)

	return &result
}

// ListHandlers handles printing human-readable table for `kn vsphere source list` command's output
func ListHandlers(h hprinters.PrintHandler) {
	sourceColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Namespace", Type: "string", Description: "Namespace of the VSphereSource", Priority: 0},
		{Name: "Name", Type: "string", Description: "Name of the VSphereSource", Priority: 1},
		{Name: "VCenter", Type: "string", Description: "URL of the vCenter", Priority: 1},
		{Name: "Insecure", Type: "boolean", Description: "vCenter TLS certificate verification", Priority: 1},
		{Name: "Credentials", Type: "string", Description: "Credentials used to connect to vCenter", Priority: 1},
		{Name: "Age", Type: "string", Description: "Age of the VSphereSource", Priority: 1},
		{Name: "Conditions", Type: "string", Description: "Ready state conditions", Priority: 1},
		{Name: "Ready", Type: "string", Description: "Ready state of the VSphereSource", Priority: 1},
		{Name: "Reason", Type: "string", Description: "Reason if state is not Ready", Priority: 1},
	}
	if err := h.TableHandler(sourceColumnDefinitions, printSource); err != nil {
		panic("add print vsphere source table handler: " + err.Error())
	}

	if err := h.TableHandler(sourceColumnDefinitions, printSourceList); err != nil {
		panic("add print list vsphere source handler: " + err.Error())
	}
}

// printSourceList populates the source list table rows
func printSourceList(sourceList *v1alpha1.VSphereSourceList, printOptions hprinters.PrintOptions) ([]metav1beta1.TableRow, error) {
	rows := make([]metav1beta1.TableRow, 0, len(sourceList.Items))

	for i := range sourceList.Items {
		r, err := printSource(&sourceList.Items[i], printOptions)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

// printSource populates the source table rows
func printSource(source *v1alpha1.VSphereSource, printOptions hprinters.PrintOptions) ([]metav1beta1.TableRow, error) {
	name := source.Name
	url := source.Spec.VAuthSpec.Address.URL()
	secret := source.Spec.VAuthSpec.SecretRef.Name
	insecure := source.Spec.VAuthSpec.SkipTLSVerify
	age := commands.TranslateTimestampSince(source.CreationTimestamp)
	conditions := commands.ConditionsValue(source.Status.Conditions)
	ready := commands.ReadyCondition(source.Status.Conditions)
	reason := commands.NonReadyConditionReason(source.Status.Conditions)

	row := metav1beta1.TableRow{
		Object: runtime.RawExtension{Object: source},
	}

	if printOptions.AllNamespaces {
		row.Cells = append(row.Cells, source.Namespace)
	}

	row.Cells = append(row.Cells,
		name,
		url,
		insecure,
		secret,
		age,
		conditions,
		ready,
		reason)
	return []metav1beta1.TableRow{row}, nil
}

// update with the v1alpha1 group + version
func updateSourceGVK(obj runtime.Object) error {
	return util.UpdateGroupVersionKindWithScheme(obj, v1alpha1.SchemeGroupVersion, scheme.Scheme)
}
