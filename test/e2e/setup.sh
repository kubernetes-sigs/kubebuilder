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

  docker pull memcached:1.6.23-alpine
  kind load docker-image --name $KIND_CLUSTER memcached:1.6.23-alpine

  docker pull busybox:1.36.1
  kind load docker-image --name $KIND_CLUSTER busybox:1.36.1

  go test $(dirname "$0")/grafana $flags -timeout 30m
  go test $(dirname "$0")/deployimage $flags -timeout 30m
  go test $(dirname "$0")/v4 $flags -timeout 30m
  go test $(dirname "$0")/externalplugin $flags -timeout 30m
  go test $(dirname "$0")/alphagenerate $flags -timeout 30m
}

function build_sample_external_plugin {
  if [ "$(uname -s)" == "Darwin" ]; then
    EXTERNAL_PLUGIN_DESTINATION_PREFIX="${HOME}/Library/Application Support/kubebuilder/plugins"
  else
    XDG_CONFIG_HOME="${HOME}/.config"
    EXTERNAL_PLUGIN_DESTINATION_PREFIX="$XDG_CONFIG_HOME/kubebuilder/plugins"
  fi

  PLUGIN_NAME="sampleexternalplugin"
  PLUGIN_VERSION="v1"
  EXTERNAL_PLUGIN_DESTINATION="${EXTERNAL_PLUGIN_DESTINATION_PREFIX}/${PLUGIN_NAME}/${PLUGIN_VERSION}"
  EXTERNAL_PLUGIN_PATH="${EXTERNAL_PLUGIN_DESTINATION}/${PLUGIN_NAME}"

  if [ -d "$EXTERNAL_PLUGIN_DESTINATION" ]; then
    echo "$EXTERNAL_PLUGIN_DESTINATION does exist."
    if [ -e "$EXTERNAL_PLUGIN_PATH" ]; then
      echo "clean up old binary..."
      rm "$EXTERNAL_PLUGIN_PATH"
    fi
  else
      mkdir -p "$EXTERNAL_PLUGIN_DESTINATION"
  fi

  REPO_ROOT_DIR="$(git rev-parse --show-toplevel)"
  SOURCE_DIR="${REPO_ROOT_DIR}/docs/book/src/simple-external-plugin-tutorial/testdata/sampleexternalplugin/v1"

  cd $SOURCE_DIR && go build -o $PLUGIN_NAME && mv $PLUGIN_NAME "$EXTERNAL_PLUGIN_PATH"

  cd $REPO_ROOT_DIR
}
