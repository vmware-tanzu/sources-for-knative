#!/usr/bin/env bash

# Copyright 2020 VMware, Inc.
# SPDX-License-Identifier: Apache-2.0

readonly ROOT_DIR=$(dirname $0)/..
source ${ROOT_DIR}/vendor/knative.dev/test-infra/scripts/library.sh

set -o errexit
set -o nounset
set -o pipefail

cd ${ROOT_DIR}

# This controls the release we track.
VERSION="v0.19"

# This controls the Kubernetes release we track.
K8S_VERSION="v0.18"

# We need these flags for things to work properly.
export GO111MODULE=on

# Parse flags to determine any we should pass to dep.
UPGRADE=0
while [[ $# -ne 0 ]]; do
  parameter=$1
  case ${parameter} in
    --upgrade) UPGRADE=1 ;;
    --release) shift; VERSION="$1";;
    --k8s-release) shift; K8S_VERSION="$1";;
    *) abort "unknown option ${parameter}" ;;
  esac
  shift
done
readonly UPGRADE
readonly VERSION

#     -require="${dep}@${K8S_VERSION}" \
#    -replace="${dep}=${dep}@${K8S_VERSION}"

if (( UPGRADE )); then
  K8S_DEPS=( $(run_go_tool tableflip.dev/buoy buoy float ${ROOT_DIR}/go.mod --domain k8s.io --release ${K8S_VERSION} --strict) )
  echo "Floating k8s deps to ${K8S_DEPS[@]}"
  for dep in "${K8S_DEPS[@]}"
  do
    # note, ${dep%@*} strips the the string after the '@', i.e.: k8s.io/code-generator@v0.18.10 => k8s.io/code-generator
    go mod edit -replace="${dep%@*}=${dep}"
  done
fi

if (( UPGRADE )); then
    FLOATING_DEPS=( $(run_go_tool tableflip.dev/buoy buoy float ${ROOT_DIR}/go.mod --release ${VERSION}) )
    echo "Floating deps to ${FLOATING_DEPS[@]}"
    go get -d ${FLOATING_DEPS[@]}
fi

# Prune modules.
go mod tidy
go mod vendor

rm -rf $(find vendor/ -name 'OWNERS')
rm -rf $(find vendor/ -name '*_test.go')
