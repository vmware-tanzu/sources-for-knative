/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package command

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version string
var BuildDate string
var GitRevision string

// NewVersionCommand implements 'kn version' command
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Prints the plugin version",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Version:      %s\n", Version)
			fmt.Fprintf(out, "Build Date:   %s\n", BuildDate)
			fmt.Fprintf(out, "Git Revision: %s\n", GitRevision)
			return nil
		},
	}
}
