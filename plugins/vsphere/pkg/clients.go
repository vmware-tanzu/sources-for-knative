/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package pkg

import (
	"fmt"
	"os"
	"path/filepath"

	vsphere "github.com/vmware-tanzu/sources-for-knative/pkg/client/clientset/versioned"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Clients struct {
	ClientConfig     clientcmd.ClientConfig
	ClientSet        kubernetes.Interface
	VSphereClientSet vsphere.Interface
}

func NewClients(kubeConfigPath string) (*Clients, error) {
	clientConfig, err := getClientConfig(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	vSphereConfig, err := vsphere.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		fmt.Println("failed to create Clients:", err)
		os.Exit(1)
	}
	return &Clients{
		ClientSet:        clientSet,
		ClientConfig:     clientConfig,
		VSphereClientSet: vSphereConfig,
	}, nil
}

func (c *Clients) GetExplicitOrDefaultNamespace(ns string) (string, error) {
	if ns != "" {
		return ns, nil
	}
	namespace, _, err := c.ClientConfig.Namespace()
	if err != nil {
		return "", err
	}
	return namespace, nil
}

func getClientConfig(kubeConfigPath string) (clientcmd.ClientConfig, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if len(kubeConfigPath) == 0 {
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{}), nil
	}
	_, err := os.Stat(kubeConfigPath)
	if err == nil {
		loadingRules.ExplicitPath = kubeConfigPath
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{}), nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	if len(filepath.SplitList(kubeConfigPath)) > 1 {
		return nil, fmt.Errorf("can not find config file. '%s' looks like a path. Please use the env var KUBECONFIG if you want to check for multiple configuration files", kubeConfigPath)
	}
	return nil, fmt.Errorf("config file '%s' can not be found", kubeConfigPath)
}
