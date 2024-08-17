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

# Not every exact cluster version has an equal tools version, and visa versa.
# This function returns the exact tools version for a k8s version based on its minor.
function convert_to_tools_ver {
  local k8s_ver=${1:?"k8s version must be set to arg 1"}
  local maj_min=$(echo $k8s_ver | grep -oE '^[0-9]+\.[0-9]+')
  case $maj_min in
  # 1.14-1.19 work with the 1.19 server bins and kubectl.
  "1.14"|"1.15"|"1.16"|"1.17"|"1.18"|"1.19") echo "1.19.2";;
  # Tests in 1.20 and 1.21 with their counterpart version's apiserver.
  "1.20"|"1.21") echo "1.21.5";;
  "1.22") echo "1.22.1";;
  "1.23") echo "1.23.3";;
  "1.24") echo "1.24.1";;
  "1.25") echo "1.25.0";;
  "1.26") echo "1.26.0";;
  "1.27") echo "1.27.1";;
  "1.28") echo "1.28.3";;
  "1.29") echo "1.29.0";;
  "1.30") echo "1.30.0";;
  "1.31") echo "1.31.0";;
  *)
    echo "k8s version $k8s_ver not supported"
    exit 1
  esac
}

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

export KIND_K8S_VERSION="${KIND_K8S_VERSION:-"v1.31.0"}"
tools_k8s_version=$(convert_to_tools_ver "${KIND_K8S_VERSION#v*}")
kind_version=0.22.0
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

    # TODO: Current workaround for setup-envtest compatibility
    # Due to past instances where controller-runtime maintainers released
    # versions without corresponding branches, directly relying on branches
    # poses a risk of breaking the Kubebuilder chain. Such practices may
    # change over time, potentially leading to compatibility issues. This
    # approach, although not ideal, remains the best solution for ensuring
    # compatibility with controller-runtime releases as of now. For more
    # details on the quest for a more robust solution, refer to the issue
    # raised in the controller-runtime repository: https://github.com/kubernetes-sigs/controller-runtime/issues/2744
    go install sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.19
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

function listPkgDirs() {
	go list -f '{{.Dir}}' ./... | grep -v generated
}

#Lists all go files
function listFiles() {
	# pipeline is much faster than for loop
	listPkgDirs | xargs -I {} find {} \( -name '*.go' -o -name '*.sh' \)  | grep -v generated
}
