#!/usr/bin/env bash

# Copyright 2020 VMware, Inc.
# SPDX-License-Identifier: Apache-2.0

function build_flags() {
  local base="${1}"
  local now="$(date -u '+%Y-%m-%d %H:%M:%S')"
  local rev="$(git rev-parse --short HEAD)"
  local version="${TAG:-}"
  # Use vYYYYMMDD-local-<hash> for the version string, if not passed.
  if [[ -z "${version}" ]]; then
    # Get the commit, excluding any tags but keeping the "dirty" flag
    local commit="$(git describe --always --dirty --match '^$')"
    [[ -n "${commit}" ]] || abort "error getting the current commit"
    version="v$(date +%Y%m%d)-local-${commit}"
  fi

  echo "-X '${VERSION_PACKAGE}.BuildDate=${now}' -X ${VERSION_PACKAGE}.Version=${version} -X ${VERSION_PACKAGE}.GitRevision=${rev}"
}
