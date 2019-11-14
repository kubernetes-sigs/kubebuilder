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

fetch_tools
# This test is used by prow and if the dep not be installed by git then it will face the GOBIN issue.
install_dep_by_git
install_kind
build_kb

setup_envs

source "$(pwd)/scripts/setup.sh" ${KIND_K8S_VERSION}
docker pull gcr.io/kubebuilder/kube-rbac-proxy:v0.4.1
kind load docker-image gcr.io/kubebuilder/kube-rbac-proxy:v0.4.1

# The v1 is deprecated
go test ./test/e2e/v1
go test ./test/e2e/v2
