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

# check if modules are enabled
MODULES_ENABLED=""
if go mod edit -json &>/dev/null ; then
  MODULES_ENABLED="1"
fi

MOD_OPT=""
MODULES_OPT=${MODULES_OPT:-""}
if [[ -n "${MODULES_OPT}" && -n "${MODULES_ENABLED}" ]]; then
    MOD_OPT="-mod=${MODULES_OPT}"
fi


# Enable tracing in this script off by setting the TRACE variable in your
# environment to any value:
#
# $ TRACE=1 test.sh
TRACE=${TRACE:-""}
if [ -n "$TRACE" ]; then
  set -x
fi

# By setting INJECT_KB_VERSION variable in your environment, KB will be compiled
# with this version. This is to assist testing functionality which depends on
# version .e.g gopkg.toml generation.
#
# $ INJECT_KB_VERSION=0.1.7 test.sh
INJECT_KB_VERSION=${INJECT_KB_VERSION:-unknown}

# Make sure, we run in the root of the repo and
# therefore run the tests on all packages
base_dir="$( cd "$(dirname "$0")/" && pwd )"
cd "$base_dir" || {
  echo "Cannot cd to '$base_dir'. Aborting." >&2
  exit 1
}

go_workspace=''
export GOPATH=${GOPATH:-$(go env GOPATH)}
for p in ${GOPATH//:/ }; do
  if [[ $PWD/ = $p/* ]]; then
    go_workspace=$p
  fi
done

k8s_version=1.16.4
goarch=amd64
goos="unknown"

if [[ "$OSTYPE" == "linux-gnu" ]]; then
  goos="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
  goos="darwin"
fi

if [[ "$goos" == "unknown" ]]; then
  echo "OS '$OSTYPE' not supported. Aborting." >&2
  exit 1
fi

# Turn colors in this script off by setting the NO_COLOR variable in your
# environment to any value:
#
# $ NO_COLOR=1 test.sh
NO_COLOR=${NO_COLOR:-""}
if [ -z "$NO_COLOR" ]; then
  header=$'\e[1;33m'
  reset=$'\e[0m'
else
  header=''
  reset=''
fi

function header_text {
  echo "$header$*$reset"
}

rc=0
tmp_root=/tmp

kb_root_dir=$tmp_root/kubebuilder
kb_orig=$(pwd)

# Skip fetching and untaring the tools by setting the SKIP_FETCH_TOOLS variable
# in your environment to any value:
#
# $ SKIP_FETCH_TOOLS=1 ./test.sh
#
# If you skip fetching tools, this script will use the tools already on your
# machine, but rebuild the kubebuilder and kubebuilder-bin binaries.
SKIP_FETCH_TOOLS=${SKIP_FETCH_TOOLS:-""}

function prepare_staging_dir {
  header_text "preparing staging dir"

  if [ -z "$SKIP_FETCH_TOOLS" ]; then
    rm -rf "$kb_root_dir"
  else
    rm -f "$kb_root_dir/bin/kubebuilder"
  fi
}

# fetch k8s API gen tools and make it available under kb_root_dir/bin.
function fetch_tools {
  if [ -z "$SKIP_FETCH_TOOLS" ]; then
    fetch_kb_tools
  fi
}

function fetch_kb_tools {
  header_text "fetching kb tools"
  kb_tools_archive_name="kubebuilder-tools-$k8s_version-$goos-$goarch.tar.gz"
  kb_tools_download_url="https://storage.googleapis.com/kubebuilder-tools/$kb_tools_archive_name"

  kb_tools_archive_path="$tmp_root/$kb_tools_archive_name"
  if [ ! -f $kb_tools_archive_path ]; then
    curl -sL ${kb_tools_download_url} -o "$kb_tools_archive_path"
  fi
  tar -zvxf "$kb_tools_archive_path" -C "$tmp_root/"
}

function build_kb {
  header_text "building kubebuilder"

  if [ "$INJECT_KB_VERSION" = "unknown" ]; then
    opts=""
  else
    # Injects the version into the cmd/version.go file
    opts=-ldflags "-X sigs.k8s.io/kubebuilder/v2/cmd.kubeBuilderVersion=$INJECT_KB_VERSION"
  fi

  GO111MODULE=on go build $opts -o $kb_root_dir/bin/kubebuilder ./cmd
}

function install_kind {
  header_text "Checking for kind"
  if ! is_installed kind ; then
    header_text "Installing kind"
    KIND_DIR=$(mktemp -d)
    pushd $KIND_DIR
    GO111MODULE=on go get sigs.k8s.io/kind@v0.7.0
    popd
  fi
}

function is_installed {
  if command -v $1 &>/dev/null; then
    return 0
  fi
  return 1
}

function prepare_testdir_under_gopath {
  kb_test_dir=$kb_root_dir/test
  header_text "preparing test directory $kb_test_dir"
  rm -rf "$kb_test_dir" && mkdir -p "$kb_test_dir" && cd "$kb_test_dir"
  header_text "running kubebuilder commands in test directory $kb_test_dir"
}

function setup_envs {
  header_text "setting up env vars"

  # Setup env vars
  export PATH=$kb_root_dir/bin:$PATH
  export TEST_ASSET_KUBECTL=$kb_root_dir/bin/kubectl
  export TEST_ASSET_KUBE_APISERVER=$kb_root_dir/bin/kube-apiserver
  export TEST_ASSET_ETCD=$kb_root_dir/bin/etcd
  export TEST_DEP=$kb_root_dir/init_project
}

function cache_project {
  header_text "caching initialized projects"
  if [ -d "$TEST_DEP" ]; then
    rm -rf "$TEST_DEP"
  fi
  mkdir -p "$TEST_DEP"
  cp -r $PWD/* $TEST_DEP
}

function dump_project {
  header_text "restoring cached project"
  if [ -d "$TEST_DEP" ]; then
    cp -r $TEST_DEP/* .
  fi
}
