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

k8s_version=1.16.4
goarch=amd64

if [[ "$OSTYPE" == "linux-gnu" ]]; then
  goos="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
  goos="darwin"
#elif [[ "$OS" == "Windows_NT" ]]; then
#  goos="windows"
else
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

tmp_root=/tmp
kb_root_dir=$tmp_root/kubebuilder

# Skip fetching and untaring the tools by setting the SKIP_FETCH_TOOLS variable
# in your environment to any value:
#
# $ SKIP_FETCH_TOOLS=1 ./test.sh
#
# If you skip fetching tools, this script will use the tools already on your
# machine, but rebuild the kubebuilder and kubebuilder-bin binaries.
SKIP_FETCH_TOOLS=${SKIP_FETCH_TOOLS:-""}

# Remove previously built binary and fetched tools if they need to be fetched again
function prepare_staging_dir {
  header_text "Preparing staging dir"

  if [ -z "$SKIP_FETCH_TOOLS" ]; then
    rm -rf "$kb_root_dir"
  else
    rm -f "$kb_root_dir/bin/kubebuilder"
  fi
}

# Build kubebuilder
function build_kb {
  header_text "Building kubebuilder"
  go build -o $kb_root_dir/bin/kubebuilder ./cmd
  kb=$kb_root_dir/bin/kubebuilder
}

# Fetch k8s API gen tools and make it available under kb_root_dir/bin.
function fetch_tools {
  if [ -z "$SKIP_FETCH_TOOLS" ]; then
    header_text "Fetching kb tools"
    kb_tools_archive_name="kubebuilder-tools-$k8s_version-$goos-$goarch.tar.gz"
    kb_tools_download_url="https://storage.googleapis.com/kubebuilder-tools/$kb_tools_archive_name"

    kb_tools_archive_path="$tmp_root/$kb_tools_archive_name"
    if [ ! -f $kb_tools_archive_path ]; then
      curl -sL ${kb_tools_download_url} -o "$kb_tools_archive_path"
    fi
    tar -zvxf "$kb_tools_archive_path" -C "$tmp_root/"
  fi

  export KUBEBUILDER_ASSETS=$kb_root_dir/bin/
}

# Installing kind in a temporal dir if no previously installed
function install_kind {
  header_text "Checking if kind is installed"
  if ! is_installed kind ; then
    header_text "Kind not found, installing kind"
    pushd $(mktemp -d)
    GO111MODULE=on go get sigs.k8s.io/kind@v0.7.0
    popd
  fi
}

# Check if a program is previously installed
function is_installed {
  if command -v $1 &>/dev/null; then
    return 0
  fi
  return 1
}
