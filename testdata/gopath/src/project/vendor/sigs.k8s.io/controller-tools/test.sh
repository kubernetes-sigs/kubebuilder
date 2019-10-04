#!/usr/bin/env bash

#  Copyright 2018 The Kubernetes Authors.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

set -e

# Enable tracing in this script off by setting the TRACE variable in your
# environment to any value:
#
# $ TRACE=1 test.sh
TRACE=${TRACE:-""}
if [ -n "$TRACE" ]; then
  set -x
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

k8s_version=1.11.0
# keeping older version around to reproduce any issue (just in case)
#k8s_version=1.10.1
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

tmp_root=/tmp
kb_root_dir=$tmp_root/kubebuilder

function header_text {
  echo "$header$*$reset"
}

# Skip fetching and untaring the tools by setting the SKIP_FETCH_TOOLS variable
# in your environment to any value:
#
# $ SKIP_FETCH_TOOLS=1 ./test.sh
#
# If you skip fetching tools, this script will use the tools already on your
# machine, but rebuild the kubebuilder and kubebuilder-bin binaries.
SKIP_FETCH_TOOLS=${SKIP_FETCH_TOOLS:-""}

# fetch k8s API gen tools and make it available under kb_root_dir/bin.
function fetch_kb_tools {
  if [ -n "$SKIP_FETCH_TOOLS" ]; then
    return 0
  fi

  header_text "fetching tools"
  kb_tools_archive_name="kubebuilder-tools-$k8s_version-$goos-$goarch.tar.gz"
  kb_tools_download_url="https://storage.googleapis.com/kubebuilder-tools/$kb_tools_archive_name"

  kb_tools_archive_path="$tmp_root/$kb_tools_archive_name"
  if [ ! -f $kb_tools_archive_path ]; then
    curl -sL ${kb_tools_download_url} -o "$kb_tools_archive_path"
  fi
  tar -zvxf "$kb_tools_archive_path" -C "$tmp_root/"
}

function setup_envs {
  header_text "setting up env vars"

  # Setup env vars
  export TEST_ASSET_KUBECTL=$kb_root_dir/bin/kubectl
  export TEST_ASSET_KUBE_APISERVER=$kb_root_dir/bin/kube-apiserver
  export TEST_ASSET_ETCD=$kb_root_dir/bin/etcd
}

header_text "using tools"

which gometalinter.v2

# fetch the testing binaries - e.g. apiserver and etcd
fetch_kb_tools

# setup testing env
setup_envs

header_text "running go vet"

go vet ./pkg/... ./cmd/...

# go get is broken for golint.  re-enable this once it is fixed.
header_text "running golint"

golint -set_exit_status ./pkg/...

header_text "running gometalinter.v2"

gometalinter.v2 --disable-all \
    --deadline 5m \
    --enable=misspell \
    --enable=structcheck \
    --enable=golint \
    --enable=deadcode \
    --enable=goimports \
    --enable=errcheck \
    --enable=varcheck \
    --enable=goconst \
    --enable=unparam \
    --enable=ineffassign \
    --enable=nakedret \
    --enable=interfacer \
    --enable=misspell \
    --enable=gocyclo \
    --skip=parse \
    ./pkg/... ./cmd/...
# enable this after fixing linting error
#    --enable=gosec \

header_text "running go test"

go test ./pkg/... ./cmd/... -parallel 4
