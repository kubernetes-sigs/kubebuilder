#!/usr/bin/env bash

# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

build_kb
fetch_tools
install_kind

# Creates a named kind cluster given a k8s version.
# The KIND_CLUSTER variable defines the cluster name and
# is expected to be defined in the calling environment.
#
# Usage:
#
#   export KIND_CLUSTER=<kind cluster name>
#   create_cluster <k8s version>
function create_cluster {
  KIND_VERSION=$1
  : ${KIND_CLUSTER:?"KIND_CLUSTER must be set"}
  : ${1:?"k8s version must be set as arg 1"}
  if ! kind get clusters | grep -q $KIND_CLUSTER ; then
    version_prefix="${KIND_VERSION%.*}"
    kind_config=$(dirname "$0")/kind-config.yaml
    if test -f $(dirname "$0")/kind-config-${version_prefix}.yaml; then
      kind_config=$(dirname "$0")/kind-config-${version_prefix}.yaml
    fi
    kind create cluster -v 4 --name $KIND_CLUSTER --retain --wait=1m --config ${kind_config} --image=kindest/node:$1
  fi
}

# Deletes a kind cluster by cluster name.
# The KIND_CLUSTER variable defines the cluster name and
# is expected to be defined in the calling environment.
#
# Usage:
#
#   export KIND_CLUSTER=<kind cluster name>
#   delete_cluster
function delete_cluster {
  : ${KIND_CLUSTER:?"KIND_CLUSTER must be set"}
  kind delete cluster --name $KIND_CLUSTER
}

function test_cluster {
  local flags="$@"

  go test $(dirname "$0")/deployimage $flags
  go test $(dirname "$0")/v2 $flags
  go test $(dirname "$0")/v3 $flags -timeout 30m
}
