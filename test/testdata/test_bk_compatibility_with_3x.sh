#!/usr/bin/env bash

# Copyright 2022 The Kubernetes Authors.
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

# The go/v3 stable plugin with kustomize/v1 was upgraded
# to move forward from its version 3.X to 4.X.
# However, we need to ensure the backwords compability
# and we cannot change the scaffolds mainly of create api and
# webhook commands to use the new features provide with kustomize 4.X
# in order to not introduce a breaking change for thoso who scaffold
# the projects with Kubebuilder CLI 3.X when they will use new versions
# of the tool. So, this test is for we ensure and does not
# allow maintainers break the stable plugin.
#
# Changes on the syntax for kustomize v4 must be addressed
# ONLY in the new plugin version kustomize/v2 instead.
#
# For further information see the doc about Plugin Versioning

source "$(dirname "$0")/../common.sh"

# Executes the test of the testdata directories
function test_bk_with_projects {
  rm -f "$(command -v controller-gen)"
  rm -f "$(command -v kustomize)"

  header_text "Performing tests in dir $1"
  pushd "$(dirname "$0")/../../testdata/$1"

  make kustomize

  header_text "Remove the kustomize bin if exists"
  rm -rf "bin/kustomize"

  header_text "Installing kustomize version v3.8.7"
  curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash -s -- 3.8.7 "./bin"
  ./bin/kustomize version

  header_text "Testing kustomize build manifests with 3.x to ensure backwords compability"
  ./bin/kustomize build "config/crd"
  ./bin/kustomize build "config/default"
  ./bin/kustomize build "config/webhook"
  ./bin/kustomize build "config/certmanager"

  popd
  header_text "No changes on the scaffold were done that breaks those who scaffold the projects with Kubebuilder 3.x and go/v3"
}

build_kb

# Test backwards compatibility of the scaffolds done with project v3 and kustomize v3.x
# IMPORTANT: Do not remove the test if it fails when you do changes in the scaffolds
# This test was done with the purpose to ensure that we will NOT change
# the kustomize/v1 scaffolds in a way that it no longer work with 3.x
test_bk_with_projects project-v3
test_bk_with_projects project-v3-multigroup
