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

set -o errexit
set -o nounset
set -o pipefail

source common.sh

export TRACE=1
export GO111MODULE=on
export KIND_K8S_VERSION=${KIND_K8S_VERSION:-v1.18.0}

fetch_tools
install_kind
build_kb

setup_envs

export KIND_CLUSTER=local-kubebuilder-e2e
if ! kind get clusters | grep -q $KIND_CLUSTER ; then
    source "$(pwd)/scripts/setup.sh" ${KIND_K8S_VERSION} $KIND_CLUSTER
    docker pull gcr.io/kubebuilder/kube-rbac-proxy:v0.5.0
    kind load --name $KIND_CLUSTER docker-image gcr.io/kubebuilder/kube-rbac-proxy:v0.5.0
fi

kind export kubeconfig --kubeconfig $tmp_root/kubeconfig --name $KIND_CLUSTER
export KUBECONFIG=$tmp_root/kubeconfig

# remove running containers on exit
function cleanup() {
    kind delete cluster --name $KIND_CLUSTER
}

if [ -z "${SKIP_KIND_CLEANUP:-}" ]; then
    trap cleanup EXIT
fi

# when changing these commands, make sure to keep in sync with ./test_e2e.sh
go test ./test/e2e/v2 -v -ginkgo.v
go test ./test/e2e/v3 -v -ginkgo.v -timeout 15m
