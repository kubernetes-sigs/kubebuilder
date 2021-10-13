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

tools_k8s_version=1.19.2
kind_version=0.11.1
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

# Certain tools are installed to GOBIN.
export PATH="$(go env GOPATH)/bin:${PATH}"

# Kubebuilder's bin path should be the last added to PATH such that it is preferred.
tmp_root=/tmp
kb_root_dir=$tmp_root/kubebuilder
mkdir -p "$kb_root_dir"
export PATH="${kb_root_dir}/bin:${PATH}"

# Skip fetching and untaring the tools by setting the SKIP_FETCH_TOOLS variable
# in your environment to any value:
#
# $ SKIP_FETCH_TOOLS=1 ./test.sh
#
# If you skip fetching tools, this script will use the tools already on your
# machine, but rebuild the kubebuilder and kubebuilder-bin binaries.
SKIP_FETCH_TOOLS=${SKIP_FETCH_TOOLS:-""}

# Build kubebuilder
function build_kb {
  header_text "Building kubebuilder"

  go build -o "${kb_root_dir}/bin/kubebuilder" ./cmd
  kb="${kb_root_dir}/bin/kubebuilder"
}

# Fetch k8s API tools and manage them globally with setup-envtest.
function fetch_tools {
  if ! is_installed setup-envtest; then
    header_text "Installing setup-envtest to $(go env GOPATH)/bin"

    go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
  fi

  if [ -z "$SKIP_FETCH_TOOLS" ]; then
    header_text "Installing e2e tools with setup-envtest"

    setup-envtest use $tools_k8s_version
  fi

  # Export KUBEBUILDER_ASSETS.
  eval $(setup-envtest use -i -p env $tools_k8s_version)
  # Downloaded tools should be used instead of counterparts present in the environment.
  export PATH="${KUBEBUILDER_ASSETS}:${PATH}"
}

# Installing kind in a temporal dir if no previously installed to GOBIN.
function install_kind {
  if ! is_installed kind ; then
    header_text "Installing kind to $(go env GOPATH)/bin"

    go install sigs.k8s.io/kind@v$kind_version
  fi
}

# Check if a program is previously installed
function is_installed {
  if command -v $1 &>/dev/null; then
    return 0
  fi
  return 1
}

