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
  echo "Getting kind config..."
  KIND_VERSION=$1
  : ${KIND_CLUSTER:?"KIND_CLUSTER must be set"}
  : ${1:?"k8s version must be set as arg 1"}
  if ! kind get clusters | grep -q $KIND_CLUSTER ; then
    version_prefix="${KIND_VERSION%.*}"
    kind_config=$(dirname "$0")/kind-config.yaml
    if test -f $(dirname "$0")/kind-config-${version_prefix}.yaml; then
      kind_config=$(dirname "$0")/kind-config-${version_prefix}.yaml
    fi
    echo "Creating cluster..."
    kind create cluster -v 4 --name $KIND_CLUSTER --retain --wait=5m --config ${kind_config} --image=kindest/node:$1
    
    echo "Waiting for cluster to be fully ready..."
    kubectl wait --for=condition=Ready nodes --all --timeout=5m
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

  # Detect the platform architecture for the kind cluster
  # Kind clusters now run natively on the host architecture (arm64 on Apple Silicon, amd64 on x86)
  local kind_platform="linux/amd64"
  if [[ "$OSTYPE" == "darwin"* ]] && [[ "$(uname -m)" == "arm64" ]]; then
    kind_platform="linux/arm64"
  elif [[ "$(uname -m)" == "aarch64" ]]; then
    kind_platform="linux/arm64"
  fi

  # Pull images for the correct platform
  docker pull --platform ${kind_platform} memcached:1.6.26-alpine3.19
  docker pull --platform ${kind_platform} busybox:1.36.1
  docker pull --platform ${kind_platform} bitnami/kubectl:latest

  # Load images directly with ctr to avoid kind's --all-platforms issue
  # kind load docker-image uses --all-platforms internally which breaks with multi-platform manifests
  docker save memcached:1.6.26-alpine3.19 | docker exec -i $KIND_CLUSTER-control-plane ctr --namespace=k8s.io images import /dev/stdin
  
  # Busybox has Docker save issues on some platforms, pull directly as fallback
  if ! docker save busybox:1.36.1 2>/dev/null | docker exec -i $KIND_CLUSTER-control-plane ctr --namespace=k8s.io images import /dev/stdin 2>/dev/null; then
    docker exec $KIND_CLUSTER-control-plane ctr --namespace=k8s.io images pull --platform ${kind_platform} docker.io/library/busybox:1.36.1 >/dev/null 2>&1
  fi

  go test $(dirname "$0")/all $flags -timeout 40m

  docker save bitnami/kubectl:latest | docker exec -i $KIND_CLUSTER-control-plane ctr --namespace=k8s.io images import /dev/stdin
}
