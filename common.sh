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
for p in ${GOPATH//:/ }; do
  if [[ $PWD/ = $p/* ]]; then
    go_workspace=$p
  fi
done

if [ -z $go_workspace ]; then
  echo 'Current directory is not in $GOPATH' >&2
  exit 1
fi

k8s_version=1.15.5
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
    rm -f "$kb_root_dir/kubebuilder/bin/kubebuilder"
  fi
}

# fetch k8s API gen tools and make it available under kb_root_dir/bin.
function fetch_tools {
  if [ -n "$SKIP_FETCH_TOOLS" ]; then
    return 0
  fi
  fetch_go_tools
  fetch_kb_tools
  IS_E2E=${E2E:-""}
  if [ -n "$IS_E2E" ]; then
    fetch_kind
  fi
}

function fetch_kb_tools {
  header_text "fetching tools"
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
    # TODO: what does this thing do.
    opts=-ldflags "-X sigs.k8s.io/kubebuilder/cmd/version.kubeBuilderVersion=$INJECT_KB_VERSION"
  fi

  GO111MODULE=on go build $opts -o $tmp_root/kubebuilder/bin/kubebuilder ./cmd
}

function fetch_kind {
  header_text "Checking for kind"
  if ! is_installed kind ; then
    header_text "Installing kind"
    KIND_DIR=$(mktemp -d)
    pushd $KIND_DIR
    GO111MODULE=on go get sigs.k8s.io/kind@v0.5.1
    popd
  fi
}

function fetch_go_tools {
  header_text "Checking for dep"
  export PATH=$(go env GOPATH)/src/github.com/golang/dep/bin:$PATH
  if ! is_installed dep ; then
    header_text "Installing dep"
    DEP_DIR=$(go env GOPATH)/src/github.com/golang/dep
    mkdir -p $DEP_DIR
    pushd $DEP_DIR
    git clone https://github.com/golang/dep.git .
    DEP_LATEST=$(git describe --abbrev=0 --tags)
    git checkout $DEP_LATEST
    mkdir bin
    go build -ldflags="-X main.version=$DEP_LATEST" -o bin/dep ./cmd/dep
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
  kb_test_dir=${go_workspace}/src/sigs.k8s.io/kubebuilder-test
  header_text "preparing test directory $kb_test_dir"
  rm -rf "$kb_test_dir" && mkdir -p "$kb_test_dir" && cd "$kb_test_dir"
  header_text "running kubebuilder commands in test directory $kb_test_dir"
}

function setup_envs {
  header_text "setting up env vars"

  # Setup env vars
  export PATH=$tmp_root/kubebuilder/bin:$PATH
  export TEST_ASSET_KUBECTL=$tmp_root/kubebuilder/bin/kubectl
  export TEST_ASSET_KUBE_APISERVER=$tmp_root/kubebuilder/bin/kube-apiserver
  export TEST_ASSET_ETCD=$tmp_root/kubebuilder/bin/etcd
  export TEST_DEP=$tmp_root/kubebuilder/init_project
  export KUBECONFIG="$(kind get kubeconfig-path --name="kind")"
}

# download_vendor_archive downloads vendor tarball for v1 projects. It skips the
# download if tarball exists.
function download_vendor_archive {
  archive_name="vendor.v1.tgz"
  archive_download_url="https://storage.googleapis.com/kubebuilder-vendor/$archive_name"
  archive_path="$tmp_root/$archive_name"
  header_text "checking the path $archive_path to download the $archive_name"
  if [ -f $archive_path ]; then
    header_text "removing file which exists"
    rm $archive_path
  fi
  header_text "downloading vendor archive from $archive_download_url"
  curl -sL ${archive_download_url} -o "$archive_path"
}

function restore_go_deps {
  header_text "restoring Go dependencies"
  download_vendor_archive
  tar -zxf $tmp_root/vendor.v1.tgz
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
    restore_go_deps
  fi
}
