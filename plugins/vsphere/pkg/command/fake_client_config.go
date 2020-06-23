/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package command

import (
	rest "k8s.io/client-go/rest"
	clientcmd "k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type FakeClientConfig struct {
	DefaultNamespaceProvider func() (string, error)
}

func (f FakeClientConfig) RawConfig() (clientcmdapi.Config, error) {
	panic("implement me")
}

func (f FakeClientConfig) ClientConfig() (*rest.Config, error) {
	panic("implement me")
}

func (f FakeClientConfig) Namespace() (string, bool, error) {
	ns, err := f.DefaultNamespaceProvider()
	return ns, false, err
}

func (f FakeClientConfig) ConfigAccess() clientcmd.ConfigAccess {
	panic("implement me")
}
