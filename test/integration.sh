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

source $(dirname "$0")/common.sh

header_text "Running kubebuilder integration tests"

build_kb
fetch_tools

pushd .

kb_test_dir=$kb_root_dir/test
kb_test_cache_dir=$kb_root_dir/cache

function prepare_test_dir {
  header_text "Preparing test directory $kb_test_dir"
  rm -rf "$kb_test_dir" && mkdir -p "$kb_test_dir" && cd "$kb_test_dir"
  header_text "Running kubebuilder commands in test directory $kb_test_dir"
}

function cache_project {
  header_text "Caching project '$1'"
  if [ -d "$kb_test_cache_dir/$1" ]; then
    rm -rf "$kb_test_cache_dir/$1"
  fi
  mkdir -p "$kb_test_cache_dir/$1"
  cp -r $PWD/* $kb_test_cache_dir/$1
}

function dump_project {
  header_text "Restoring cached project '$1'"
  if [ -d "$kb_test_cache_dir/$1" ]; then
    cp -r $kb_test_cache_dir/$1/* .
  fi
}


function test_init_project {
  header_text "Init project"
  go mod init kubebuilder.io/test
  if [ "go.work" == "$1" ]; then
    $kb init --domain example.com --workspace <<< "n"
  else
    $kb init --domain example.com <<< "n"
  fi
}

function test_make_project {
  header_text "Running make"
  make
}

function test_create_api_controller {
  header_text "Creating api and controller"
  $kb create api --group insect --version v1beta1 --kind Bee --namespaced false <<EOF
y
y
EOF
}

function test_create_namespaced_api_controller {
  header_text "Creating namespaced api and controller"
  $kb create api --group insect --version v1beta1 --kind Bee --namespaced true <<EOF
y
y
EOF
}

function test_create_api_only {
  header_text "Creating api only"
  $kb create api --group insect --version v1beta1 --kind Bee --namespaced false <<EOF
y
n
EOF
}

function test_create_namespaced_api_only {
  header_text "Creating namespaced api only"
  $kb create api --group insect --version v1beta1 --kind Bee --namespaced true <<EOF
y
n
EOF
}

function test_create_core_type_controller {
  header_text "Creating coretype controller"
  $kb create api --group apps --version v1 --kind Deployment --namespaced false <<EOF
n
y
EOF
}


prepare_test_dir
test_init_project "go.mod"
cache_project "init"
test_make_project

prepare_test_dir
dump_project "init"
test_create_api_controller

prepare_test_dir
dump_project "init"
test_create_namespaced_api_controller

prepare_test_dir
dump_project "init"
test_create_api_only

prepare_test_dir
dump_project "init"
test_create_namespaced_api_only

prepare_test_dir
dump_project "init"
test_create_core_type_controller

prepare_test_dir
test_init_project "go.work"
cache_project "init-workspace"
test_make_project

prepare_test_dir
dump_project "init-workspace"
test_create_api_controller

prepare_test_dir
dump_project "init-workspace"
test_create_namespaced_api_controller

prepare_test_dir
dump_project "init-workspace"
test_create_api_only

prepare_test_dir
dump_project "init-workspace"
test_create_namespaced_api_only

prepare_test_dir
dump_project "init-workspace"
test_create_core_type_controller

popd
