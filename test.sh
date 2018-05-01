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
set -x

# Make sure, we run in the root of the repo and
# therefore run the tests on all packages
base_dir="$( cd "$(dirname "$0")/" && pwd )"
cd "$base_dir" || {
  echo "Cannot cd to '$base_dir'. Aborting." >&2
  exit 1
}

k8s_version=1.10
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
NO_COLOR=${NO_COLOR:""}
header=$'\e[1;33m'
reset=$'\e[0m'
function header_text {
  if [ -z "$NO_COLOR" ]; then
    echo "$header${@}$reset"
  else
    echo ${@}
  fi
}

rc=0
tmp_root=/tmp

kb_root_dir=$tmp_root/kubebuilder
kb_vendor_dir=$tmp_root/vendor

function prepare_staging_dir {
  header_text "preparing staging dir"
  rm -rf $kb_root_dir
}

# fetch k8s API gen tools and make it available under kb_root_dir/bin.
function fetch_tools {
  header_text "fetching tools"
  kb_tools_archive_name=kubebuilder-tools-$k8s_version-$goos-$goarch.tar.gz
  kb_tools_download_url="https://storage.googleapis.com/kubebuilder-tools/$kb_tools_archive_name"

  kb_tools_archive_path=$tmp_root/$kb_tools_archive_name
  if [ ! -f $kb_tools_archive_path ]; then
    curl -sL ${kb_tools_download_url} -o $kb_tools_archive_path
  fi
  tar -zxf $kb_tools_archive_path -C $tmp_root/
}

function build_kb {
  header_text "building kubebuilder"
  go build -o $tmp_root/kubebuilder/bin/kubebuilder ./cmd/kubebuilder
  go build -o $tmp_root/kubebuilder/bin/kubebuilder-gen ./cmd/kubebuilder-gen
}

function prepare_vendor_deps {
  header_text "preparing vendor dependencies"
  # TODO(droot): clean up this function
  rm -rf $kb_vendor_dir && rm -f $tmp_root/Gopkg.toml && rm -f $tmp_root/Gopkg.lock
  mkdir -p $kb_vendor_dir/github.com/kubernetes-sigs/kubebuilder/pkg/ || echo ""
  cp -r pkg/* $kb_vendor_dir/github.com/kubernetes-sigs/kubebuilder/pkg/
  cp LICENSE $kb_vendor_dir/github.com/kubernetes-sigs/kubebuilder/LICENSE
  cp Gopkg.toml Gopkg.lock $tmp_root/
  cp -a vendor/* $kb_vendor_dir/
  cd $tmp_root 
  sed -i "s/KUBEBUILDER_VERSION/"${VERSION-master}"/" Gopkg.toml
  tar -czf $kb_root_dir/bin/vendor.tar.gz vendor/ Gopkg.lock  Gopkg.toml
}

function prepare_testdir_under_gopath {
  kb_test_dir=$GOPATH/src/github.com/kubernetes-sigs/kubebuilder-test
  header_text "preparing test directory $kb_test_dir"
  rm -rf $kb_test_dir && mkdir -p $kb_test_dir && cd $kb_test_dir
  header_text "running kubebuilder commands in test directory $kb_test_dir"
}

function generate_crd_resources {
  header_text "generating CRD resources and code"
  
  # Setup env vars
  export PATH=/tmp/kubebuilder/bin/:$PATH
  export TEST_ASSET_KUBECTL=/tmp/kubebuilder/bin/kubectl
  export TEST_ASSET_KUBE_APISERVER=/tmp/kubebuilder/bin/kube-apiserver
  export TEST_ASSET_ETCD=/tmp/kubebuilder/bin/etcd

  # Run the commands
  kubebuilder init repo --domain sample.kubernetes.io
  kubebuilder create resource --group insect --version v1beta1 --kind Bee

  header_text "editing generated files to simulate a user"
  sed -i pkg/apis/insect/v1beta1/bee_types.go -e "s|type Bee struct|// +kubebuilder:categories=foo,bar\ntype Bee struct|"

  header_text "generating and testing CRD definition"
  kubebuilder create config --crds --output crd.yaml

  # Test for the expected generated CRD definition
  #
  # TODO: this is awkwardly inserted after the first resource created in this
  # test because the output order seems nondeterministic and it's preferable to
  # avoid introducing a new dependency like yq or complex parsing logic
  cat << EOF > expected.yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  labels:
    api: ""
    kubebuilder.k8s.io: unknown
  name: bees.insect.sample.kubernetes.io
spec:
  group: insect.sample.kubernetes.io
  names:
    categories:
    - foo
    - bar
    kind: Bee
    plural: bees
  scope: Namespaced
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object
        spec:
          type: object
        status:
          type: object
      type: object
  version: v1beta1
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: null
EOF
  diff crd.yaml expected.yaml

  kubebuilder create resource --group insect --version v1beta1 --kind Wasp
}

function test_generated_controller {
  header_text "building generated code"
  # Verify the controller-manager builds and the tests pass
  go build ./cmd/...
  go build ./pkg/...

  header_text "testing generated code"
  go test -v ./cmd/...
  go test -v ./pkg/...
}

prepare_staging_dir
fetch_tools
build_kb
prepare_vendor_deps
prepare_testdir_under_gopath

generate_crd_resources
test_generated_controller
exit $rc
