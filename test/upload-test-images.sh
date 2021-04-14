#!/usr/bin/env bash

# Copyright 2020 VMware, Inc.
# SPDX-License-Identifier: Apache-2.0

: "${KO_DOCKER_REPO:?"You must set 'KO_DOCKER_REPO', see DEVELOPMENT.md"}"

export GO111MODULE=on
export GOFLAGS=-mod=vendor

cat << EOF| ko resolve -Bf -
images:
- ko://github.com/vmware/govmomi/govc
- ko://github.com/vmware-tanzu/sources-for-knative/test/test_images/listener
EOF
