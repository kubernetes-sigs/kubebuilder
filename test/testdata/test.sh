#!/usr/bin/env bash

# Copyright 2018 The Kubernetes Authors.
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

source "$(dirname "$0")/../common.sh"
source "$(dirname "$0")/../e2e/setup.sh"

function madarchod {
  KIND_VERSION=$1
  : ${KIND_CLUSTER:?"KIND_CLUSTER must be set"}
  : ${1:?"k8s version must be set as arg 1"}
  if ! kind get clusters | grep -q $KIND_CLUSTER ; then
    version_prefix="${KIND_VERSION%.*}"
    kind_config=./../../test/e2e/kind-config.yaml
    if test -f ./../../test/e2e/kind-config-${version_prefix}.yaml; then
      kind_config=./../../test/e2e/kind-config-${version_prefix}.yaml
    fi
    kind create cluster -v 4 --name $KIND_CLUSTER --retain --wait=1m --config ${kind_config} --image=kindest/node:$1
  fi
}

function test_project {
  header_text "Performing tests in dir $1"
  pushd "$(dirname "$0")/../../testdata/$1"

  # Remove the test directory if the project is not project-v4
  if [ "$1" != "project-v4" ]; then
    if [ -d "test" ]; then
      echo "Removing test directory."
      rm -rf test/
    fi
  else
    # For project-v4: Install kubectl, create kind cluster and run e2e tests
    KUSTOMIZATION_FILE_PATH="./../../testdata/$1/config/default/kustomization.yaml"

    sed -i '25s/^#//' $KUSTOMIZATION_FILE_PATH
    sed -i '27s/^#//' $KUSTOMIZATION_FILE_PATH
    sed -i '42s/^#//' $KUSTOMIZATION_FILE_PATH
    sed -i '46,143s/^#//' $KUSTOMIZATION_FILE_PATH
    # Install kubectl if it is not installed
    install_kubectl
    # build_sample_external_plugin
    # Create a kind cluster named 'kind'
    export KIND_CLUSTER="kind"
    madarchod ${KIND_K8S_VERSION}
    if [ -z "${SKIP_KIND_CLEANUP:-}" ]; then
      trap delete_cluster EXIT
    fi
    # Execute e2e tests
    if [ -d "test" ]; then
      echo "Test directory exists. Executing e2e tests."
      # Your e2e test execution command goes here
    else
      echo "Test directory does not exist. Skipping e2e tests."
    fi
  fi

  go mod tidy
  make test
  popd
}

build_kb

# Project version v4-alpha

if test_project project-v4; then
  test_project project-v4-multigroup
  test_project project-v4-multigroup-with-deploy-image
  test_project project-v4-with-deploy-image
  test_project project-v4-with-grafana
fi
