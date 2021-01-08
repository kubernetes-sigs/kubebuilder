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
export PATH=$kb_root_dir/bin:$PATH
fetch_tools
install_kind

# Creates a kind cluster given a k8s version and a cluster name.
#
# Usage:
#
#   create_cluster <k8s version> <kind cluster name>
function create_cluster {
  if ! kind get clusters | grep -q $2 ; then
    kind create cluster -v 4 --name $2 --retain --wait=1m --config $(dirname "$0")/kind-config.yaml --image=kindest/node:$1
  fi
}

# Deletes a kind cluster by cluster name. The kind cluster needs to be defined as a variable instead of an argument
# so that this function can be used with `trap`
#
# Usage:
#
#   kind_cluster=<kind cluster name>
#   delete_cluster
function delete_cluster {
    kind delete cluster --name $kind_cluster
}

function test_cluster {
  local flags="$@"

  go test $(dirname "$0")/v2 $flags
  go test $(dirname "$0")/v3 $flags -timeout 20m
}
