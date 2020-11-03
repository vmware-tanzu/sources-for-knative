/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package vsphere holds utilities for bootstrapping a vSphere API client
// from the metadata injected by the VSphereSource.  Within a receive adapter,
// users can create a new vSphere SOAP API client with automatic keep-alive:
//    client, err := vsphere.NewSOAPClient(ctx)
//
// To properly release vSphere API resources, it is recommended to log out when the client is not needed anymore:
//    defer client.Logout(context.Background())
//
// This is modeled after the Bindings pattern.
package vsphere
