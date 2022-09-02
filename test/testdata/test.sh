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

# Executes the test of the testdata directories
function test_project {
  rm -f "$(command -v controller-gen)"
  rm -f "$(command -v kustomize)"

  header_text "Performing tests in dir $1"
  pushd "$(dirname "$0")/../../testdata/$1"
  if test -f "api/go.mod"; then
    cd api && go mod tidy && cd ..
  elif test -f "apis/go.mod"; then
    cd apis && go mod tidy && cd ..
  fi
  go mod tidy
  make test
  popd
}

build_kb

# Test project v3
test_project project-v3
test_project project-v3-multigroup 
test_project project-v3-multimodule
test_project project-v3-addon-and-grafana
test_project project-v3-config
test_project project-v3-with-deploy-image

# Project version v4-alpha
test_project project-v4
test_project project-v4-multigroup
test_project project-v4-multimodule
test_project project-v4-addon-and-grafana
test_project project-v4-config
test_project project-v4-with-deploy-image
