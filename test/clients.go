/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package test

import (
	"testing"
	"time"

	clients "github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// Support running e2e on GKE
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	pkgtest "knative.dev/pkg/test"

	"github.com/vmware-tanzu/sources-for-knative/pkg/client/clientset/versioned"
	sources "github.com/vmware-tanzu/sources-for-knative/pkg/client/clientset/versioned/typed/sources/v1alpha1"
)

const (
	// To simplify testing against our sample yamls, run our testing
	// in the default namespace.
	Namespace = "default"

	// PollInterval is how frequently e2e tests will poll for updates.
	PollInterval = 5 * time.Second
	// PollTimeout is how long e2e tests will wait for resource updates when polling.
	PollTimeout = 5 * time.Minute
)

type Clients struct {
	KubeClient   kubernetes.Interface
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
	return NewClientsFromConfig(configPath, clusterName, namespace)
}

func Setup(t *testing.T) *Clients {
	t.Helper()
	result, err := NewClients(pkgtest.Flags.Kubeconfig, pkgtest.Flags.Cluster, Namespace)
	if err != nil {
		t.Fatal("Couldn't initialize clients", "error", err.Error())
	}
	return result
}

func NewClientsFromConfig(configPath, clusterName, namespace string) (*Clients, error) {
	result := &Clients{}
	cfg, err := pkgtest.BuildClientConfig(configPath, clusterName)
	if err != nil {
		return nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	result.KubeClient = kubeClient
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

func (c *Clients) AsPluginClients() *clients.Clients {
	return &clients.Clients{
		ClientConfig:     c.clientConfig,
		ClientSet:        c.KubeClient,
		VSphereClientSet: c.VMWareClient.clientSet,
	}
}
