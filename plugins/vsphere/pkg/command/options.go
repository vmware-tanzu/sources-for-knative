/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package command

type GenericOptions struct {
	Namespace     string
	Name          string
	AllNamespaces bool // used in list commands
	// TODO: add label flag for filtering/attaching on/to created objects?
}
