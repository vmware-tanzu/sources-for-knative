/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package command

import (
	"net/url"
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"knative.dev/pkg/apis"
)

const DefaultNamespace = "configuredDefault"

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

func CheckFlag(t *testing.T, command *cobra.Command, flagName string) bool {
	return assert.Check(t, command.Flag(flagName) != nil, "command should have a '%s' flag", flagName)
}

func RegularClientConfig() clientcmd.ClientConfig {
	return FakeClientConfig{DefaultNamespaceProvider: func() (string, error) {
		return DefaultNamespace, nil
	}}
}

func FailingClientConfig(err error) clientcmd.ClientConfig {
	return FakeClientConfig{DefaultNamespaceProvider: func() (string, error) {
		return "", err
	}}
}

func ParseURI(t *testing.T, uri string) apis.URL {
	result, err := url.Parse(uri)
	assert.NilError(t, err)
	return apis.URL(*result)
}

func HasLeafCommand(command *cobra.Command, subcommandName string) bool {
	_, unprocessed, err := command.Find([]string{subcommandName})
	return err == nil && len(unprocessed) == 0
}
