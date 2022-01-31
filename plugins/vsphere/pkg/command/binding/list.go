/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package binding

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

func NewBindingListCommand(clients *pkg.Clients, opts *Options) *cobra.Command {
	bindingListFlags := flags.NewListPrintFlags(ListHandlers)

	result := cobra.Command{
		Use:     "list",
		Short:   "List vSphere bindings",
		Long:    "List vSphere bindings",
		Aliases: []string{"ls"},
		Example: `# List the bindings in the default namespace
kn vsphere binding list

# List the bindings in the specified namespace
kn vsphere binding list --namespace ns

# List the bindings in all namespaces with JSON output
kn vsphere binding list --all-namespaces -o json
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
				bindingListFlags.EnsureWithNamespace()
			}

			bindingList, err := clients.VSphereClientSet.SourcesV1alpha1().VSphereBindings(namespace).List(cmd.Context(), metav1.ListOptions{})
			if err != nil {
				return fmt.Errorf("list bindings: %v", err)
			}

			if !bindingListFlags.GenericPrintFlags.OutputFlagSpecified() && len(bindingList.Items) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No bindings found.\n")
				return nil
			}

			listCopy := bindingList.DeepCopy()
			if err = updateBindingGVK(listCopy); err != nil {
				return fmt.Errorf("update binding GKV: %v", err)
			}

			err = bindingListFlags.Print(listCopy, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return nil
		},
	}

	fl := result.Flags()
	fl.BoolVarP(&opts.AllNamespaces, "all-namespaces", "A", false, "list objects in all namespaces")

	bindingListFlags.AddFlags(&result)

	return &result
}

// ListHandlers handles printing human-readable table for `kn vsphere binding list` command's output
func ListHandlers(h hprinters.PrintHandler) {
	bindingColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Namespace", Type: "string", Description: "Namespace of the VSphereBinding", Priority: 0},
		{Name: "Name", Type: "string", Description: "Name of the VSphereBinding", Priority: 1},
		{Name: "VCenter", Type: "string", Description: "URL of the vCenter", Priority: 1},
		{Name: "Insecure", Type: "boolean", Description: "vCenter TLS certificate verification", Priority: 1},
		{Name: "Credentials", Type: "string", Description: "Credentials used to connect to vCenter", Priority: 1},
		{Name: "Age", Type: "string", Description: "Age of the VSphereBinding", Priority: 1},
		{Name: "Conditions", Type: "string", Description: "Ready state conditions", Priority: 1},
		{Name: "Ready", Type: "string", Description: "Ready state of the VSphereBinding", Priority: 1},
		{Name: "Reason", Type: "string", Description: "Reason if state is not Ready", Priority: 1},
	}
	if err := h.TableHandler(bindingColumnDefinitions, printBinding); err != nil {
		panic("add print vsphere binding table handler: " + err.Error())
	}

	if err := h.TableHandler(bindingColumnDefinitions, printBindingList); err != nil {
		panic("add print list vsphere binding handler: " + err.Error())
	}
}

// printBindingList populates the binding list table rows
func printBindingList(bindingList *v1alpha1.VSphereBindingList, printOptions hprinters.PrintOptions) ([]metav1beta1.TableRow, error) {
	rows := make([]metav1beta1.TableRow, 0, len(bindingList.Items))

	for i := range bindingList.Items {
		r, err := printBinding(&bindingList.Items[i], printOptions)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

// printBinding populates the binding table rows
func printBinding(binding *v1alpha1.VSphereBinding, printOptions hprinters.PrintOptions) ([]metav1beta1.TableRow, error) {
	name := binding.Name
	url := binding.Spec.VAuthSpec.Address.URL()
	secret := binding.Spec.VAuthSpec.SecretRef.Name
	insecure := binding.Spec.VAuthSpec.SkipTLSVerify
	age := commands.TranslateTimestampSince(binding.CreationTimestamp)
	conditions := commands.ConditionsValue(binding.Status.Conditions)
	ready := commands.ReadyCondition(binding.Status.Conditions)
	reason := commands.NonReadyConditionReason(binding.Status.Conditions)

	row := metav1beta1.TableRow{
		Object: runtime.RawExtension{Object: binding},
	}

	if printOptions.AllNamespaces {
		row.Cells = append(row.Cells, binding.Namespace)
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
func updateBindingGVK(obj runtime.Object) error {
	return util.UpdateGroupVersionKindWithScheme(obj, v1alpha1.SchemeGroupVersion, scheme.Scheme)
}
