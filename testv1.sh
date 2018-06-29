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

function test_init_project {
  header_text "performing init project"
  kubebuilder init --project-version=v1 --domain example.com <<< "y"
  make
}

function test_init_project_manual_dep_ensure {
  header_text "performing init project w/o dep ensure"
  kubebuilder init --project-version=v1 --domain example.com <<< "n"
  dep ensure
  make
}

function test_create_api_controller {
  header_text "performing creating api and controller"
  kubebuilder create api --group insect --version v1beta1 --kind Bee <<EOF
y
y
EOF
}

function test_create_api_only {
  header_text "performing creating api only"
  kubebuilder create api --group insect --version v1beta1 --kind Bee <<EOF
y
n
EOF
}

function test_create_skip {
  header_text "performing creating but skipping everything"
  kubebuilder create api --group insect --version v1beta1 --kind Bee <<EOF
n
n
EOF
}

function test_create_coretype_controller {
  header_text "performing creating coretype controller"
  kubebuilder create api --group apps --version v1 --kind Deployment <<EOF
n
y
EOF
}

prepare_staging_dir
fetch_tools
build_kb

setup_envs

prepare_testdir_under_gopath
test_init_project

prepare_testdir_under_gopath
test_init_project_manual_dep_ensure

prepare_testdir_under_gopath
test_init_project
test_create_api_controller

prepare_testdir_under_gopath
test_init_project
test_create_api_only

# enable this test case after fixing it
#prepare_testdir_under_gopath
#test_init_project
#test_create_coretype_controller

exit $rc
