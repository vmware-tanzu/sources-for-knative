/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package source

import (
	"time"

	"github.com/spf13/cobra"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg"
	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"
)

type Options struct {
	command.GenericOptions

	VCAddress     string
	SkipTLSVerify bool
	SecretRef     string

	SinkURI            string
	SinkAPIVersion     string
	SinkKind           string
	SinkName           string
	ServiceAccountName string

	CheckpointMaxAge time.Duration
	CheckpointPeriod time.Duration

	PayloadEncoding string
}

func (so *Options) AsSinkDestination(namespace string) (*duckv1.Destination, error) {
	apiURL, err := apis.ParseURL(so.SinkURI)
	if err != nil {
		return nil, err
	}

	return &duckv1.Destination{
		Ref: so.sinkReference(namespace),
		URI: apiURL,
	}, nil
}

func (so *Options) sinkReference(namespace string) *duckv1.KReference {
	if so.SinkAPIVersion == "" {
		return nil
	}
	return &duckv1.KReference{
		APIVersion: so.SinkAPIVersion,
		Kind:       so.SinkKind,
		Namespace:  namespace,
		Name:       so.SinkName,
	}
}

func NewSourceCommand(clients *pkg.Clients) *cobra.Command {
	options := Options{}

	result := cobra.Command{
		Use:   "source",
		Short: "Manage vSphere Event Sources",
		Long:  "Manage vSphere Event Sources",
	}

	flags := result.PersistentFlags()
	flags.StringVarP(&options.Namespace, "namespace", "n", "", "namespace to use (default namespace if omitted)")

	result.AddCommand(NewSourceCreateCommand(clients, &options))
	result.AddCommand(NewSourceDeleteCommand(clients, &options))
	result.AddCommand(NewSourceListCommand(clients, &options))

	return &result
}
