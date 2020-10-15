/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package test

import (
	"time"

	clients "github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"knative.dev/pkg/test"

	// Support running e2e on GKE
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/vmware-tanzu/sources-for-knative/pkg/client/clientset/versioned"
	sources "github.com/vmware-tanzu/sources-for-knative/pkg/client/clientset/versioned/typed/sources/v1alpha1"
)

const (
	// To simplify testing against our sample yamls, run our testing
	// in the default namespace.
	Namespace = "default"

	// PollInterval is how frequently e2e tests will poll for updates.
	PollInterval = 1 * time.Second
	// PollTimeout is how long e2e tests will wait for resource updates when polling.
	PollTimeout = 1 * time.Minute
)

type Clients struct {
	KubeClient   *test.KubeClient
	VMWareClient *VMWareClients
	// for interop with clients.Clients -- see AsPluginClients
	clientConfig clientcmd.ClientConfig
}

type VMWareClients struct {
	Bindings sources.VSphereBindingInterface
	Sources  sources.VSphereSourceInterface
	// for interop with clients.Clients -- see AsPluginClients
	clientSet *versioned.Clientset
}

func NewClients(configPath, clusterName, namespace string) (*Clients, error) {
	clientConfig := BuildClientConfig(configPath, clusterName)
	return NewClientsFromConfig(clientConfig, namespace)
}

func Setup(t test.TLegacy) *Clients {
	t.Helper()
	result, err := NewClients(test.Flags.Kubeconfig, test.Flags.Cluster, Namespace)
	if err != nil {
		t.Fatal("Couldn't initialize clients", "error", err.Error())
	}
	return result
}

func NewClientsFromConfig(clientConfig clientcmd.ClientConfig, namespace string) (*Clients, error) {
	result := &Clients{}
	result.clientConfig = clientConfig
	cfg, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	result.KubeClient = &test.KubeClient{Interface: kubeClient}
	cs, err := versioned.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	result.VMWareClient = &VMWareClients{
		Bindings:  cs.SourcesV1alpha1().VSphereBindings(namespace),
		Sources:   cs.SourcesV1alpha1().VSphereSources(namespace),
		clientSet: cs,
	}

	return result, nil
}

func BuildClientConfig(kubeConfigPath string, clusterName string) clientcmd.ClientConfig {
	overrides := clientcmd.ConfigOverrides{}
	// Override the cluster name if provided.
	if clusterName != "" {
		overrides.Context.Cluster = clusterName
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath},
		&overrides)
}

func (c *Clients) AsPluginClients() *clients.Clients {
	return &clients.Clients{
		ClientConfig:     c.clientConfig,
		ClientSet:        c.KubeClient,
		VSphereClientSet: c.VMWareClient.clientSet,
	}
}
