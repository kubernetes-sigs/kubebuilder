#!/usr/bin/env bash
# todo: remove this file when go/v2 be removed
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

# Executes the test of the testdata directories
function test_project {
  rm -f "$(command -v controller-gen)"
  rm -f "$(command -v kustomize)"

  header_text "Performing tests in dir $1"
  pushd "$(dirname "$0")/../../testdata/$1"
  go mod tidy
  go get -u golang.org/x/sys
  make test
  popd
}

build_kb

# Test project v2, which relies on pre-installed envtest tools to run 'make test'.
tools_k8s_version="1.19.2"
fetch_tools
test_project project-v2
test_project project-v3
