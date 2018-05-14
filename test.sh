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
TRACE=${TRACE:""}
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

k8s_version=1.10.1
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
kb_orig=$(pwd)

# Skip fetching and untaring the tools by setting the SKIP_FETCH_TOOLS variable
# in your environment to any value:
#
# $ SKIP_FETCH_TOOLS=1 ./test.sh
#
# If you skip fetching tools, this script will use the tools already on your
# machine, but rebuild the kubebuilder and kubebuilder-bin binaries.
SKIP_FETCH_TOOLS=${SKIP_FETCH_TOOLS:""}

function prepare_staging_dir {
  header_text "preparing staging dir"

  if [ -z "$SKIP_FETCH_TOOLS" ]; then
    rm -rf $kb_root_dir
  else
    rm -f $kb_root_dir/kubebuilder/bin/kubebuilder
    rm -f $kb_root_dir/kubebuilder/bin/kubebuilder-gen
    rm -f $kb_root_dir/kubebuilder/bin/vendor.tar.gz
  fi
}

# fetch k8s API gen tools and make it available under kb_root_dir/bin.
function fetch_tools {
  if [ -n "$SKIP_FETCH_TOOLS" ]; then
    return 0
  fi

  header_text "fetching tools"
  kb_tools_archive_name=kubebuilder-tools-$k8s_version-$goos-$goarch.tar.gz
  kb_tools_download_url="https://storage.googleapis.com/kubebuilder-tools/$kb_tools_archive_name"

  kb_tools_archive_path=$tmp_root/$kb_tools_archive_name
  if [ ! -f $kb_tools_archive_path ]; then
    curl -sL ${kb_tools_download_url} -o $kb_tools_archive_path
  fi
  tar -zvxf $kb_tools_archive_path -C $tmp_root/
}

function build_kb {
  header_text "building kubebuilder"

  if [ "$INJECT_KB_VERSION" = "unknown" ]; then
   go build -o $tmp_root/kubebuilder/bin/kubebuilder ./cmd/kubebuilder
  else
   go build -ldflags "-X github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/version.kubeBuilderVersion=$INJECT_KB_VERSION" -o $tmp_root/kubebuilder/bin/kubebuilder ./cmd/kubebuilder
  fi

  go build -o $tmp_root/kubebuilder/bin/kubebuilder-gen ./cmd/kubebuilder-gen
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
  sed -i -e "s|type Bee struct|// +kubebuilder:categories=foo,bar\ntype Bee struct|" pkg/apis/insect/v1beta1/bee_types.go
  sed -i -e "s|type BeeController struct {|// +kubebuilder:rbac:groups="",resources=pods,verbs=get;watch;list\ntype BeeController struct {|" pkg/controller/bee/controller.go

  header_text "generating and testing CRD definition"
  kubebuilder create config --crds --output crd.yaml
  kubebuilder create config --controller-image myimage:v1 --name myextensionname --output install.yaml

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
    kubebuilder.k8s.io: $INJECT_KB_VERSION
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

    cat << EOF > expected.yaml
apiVersion: v1
kind: Namespace
metadata:
  creationTimestamp: null
  labels:
    api: myextensionname
    kubebuilder.k8s.io: $INJECT_KB_VERSION
  name: myextensionname-system
spec: {}
status: {}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  creationTimestamp: null
  labels:
    api: myextensionname
    kubebuilder.k8s.io: $INJECT_KB_VERSION
  name: myextensionname-role
rules:
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - get
  - watch
  - list
- apiGroups:
  - insect.sample.kubernetes.io
  resources:
  - '*'
  verbs:
  - '*'
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  creationTimestamp: null
  labels:
    api: myextensionname
    kubebuilder.k8s.io: $INJECT_KB_VERSION
  name: myextensionname-rolebinding
  namespace: myextensionname-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: myextensionname-role
subjects:
- kind: ServiceAccount
  name: default
  namespace: myextensionname-system
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  labels:
    api: myextensionname
    kubebuilder.k8s.io: $INJECT_KB_VERSION
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
---
apiVersion: v1
kind: Service
metadata:
  creationTimestamp: null
  labels:
    api: myextensionname
    control-plane: controller-manager
    kubebuilder.k8s.io: $INJECT_KB_VERSION
  name: myextensionname-controller-manager-service
  namespace: myextensionname-system
spec:
  clusterIP: None
  selector:
    api: myextensionname
    control-plane: controller-manager
    kubebuilder.k8s.io: $INJECT_KB_VERSION
status:
  loadBalancer: {}
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  creationTimestamp: null
  labels:
    api: myextensionname
    control-plane: controller-manager
    kubebuilder.k8s.io: $INJECT_KB_VERSION
  name: myextensionname-controller-manager
  namespace: myextensionname-system
spec:
  replicas: 1
  selector:
    matchLabels:
      api: myextensionname
      control-plane: controller-manager
      kubebuilder.k8s.io: $INJECT_KB_VERSION
  serviceName: myextensionname-controller-manager-service
  template:
    metadata:
      creationTimestamp: null
      labels:
        api: myextensionname
        control-plane: controller-manager
        kubebuilder.k8s.io: $INJECT_KB_VERSION
    spec:
      containers:
      - args:
        - --install-crds=false
        command:
        - /root/controller-manager
        image: myimage:v1
        name: controller-manager
        resources:
          limits:
            cpu: 100m
            memory: 30Mi
          requests:
            cpu: 100m
            memory: 20Mi
      terminationGracePeriodSeconds: 10
  updateStrategy: {}
status:
  replicas: 0

EOF
  diff -B install.yaml expected.yaml


  kubebuilder create resource --group insect --version v1beta1 --kind Wasp
  kubebuilder create resource --group ant --version v1beta1 --kind Ant --controller=false
  kubebuilder create config --crds --output crd.yaml

  # Check for ordering of generated YAML
  # TODO: make this a more concise test in a follow-up
  cat << EOF > expected.yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  labels:
    api: ""
    kubebuilder.k8s.io: $INJECT_KB_VERSION
  name: ants.ant.sample.kubernetes.io
spec:
  group: ant.sample.kubernetes.io
  names:
    kind: Ant
    plural: ants
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
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  labels:
    api: ""
    kubebuilder.k8s.io: $INJECT_KB_VERSION
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
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  labels:
    api: ""
    kubebuilder.k8s.io: $INJECT_KB_VERSION
  name: wasps.insect.sample.kubernetes.io
spec:
  group: insect.sample.kubernetes.io
  names:
    kind: Wasp
    plural: wasps
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

function test_vendor_update {
 header_text "performing vendor update"
 kubebuilder update vendor
}

function test_docs {
  header_text "building docs"
  kubebuilder docs --docs-copyright "Hello" --title "World" --cleanup=false --brodocs=false
  diff docs/reference/includes $kb_orig/test/docs/expected/includes
  diff docs/reference/manifest.json $kb_orig/test/docs/expected/manifest.json
  diff docs/reference/config.yaml $kb_orig/test/docs/expected/config.yaml
}

function generate_controller {
  header_text "creating controller"
    kubebuilder create controller --group ant --version v1beta1 --kind Ant
}

function update_controller_test {
  # Update import
  sed -i 's!"k8s.io/client-go/kubernetes/typed/apps/v1beta2"!&\n    "k8s.io/api/core/v1"!' ./pkg/controller/deployment/controller_test.go

  # Fill deployment instance
  sed -i 's!instance.Name = "instance-1"!&\n    instance.Spec.Template.Spec.Containers = []v1.Container{{Name: "name", Image: "someimage"}}\n    labels := map[string]string{"foo": "bar"}\n    instance.Spec.Template.ObjectMeta.Labels = labels\n    instance.Spec.Selector = \&metav1.LabelSelector{MatchLabels: labels}!' ./pkg/controller/deployment/controller_test.go
}

function generate_coretype_controller {
    header_text "generating controller for coretype Deployment"

    # Run the commands
    kubebuilder init repo --domain sample.kubernetes.io --controller-only
    kubebuilder create controller --group apps --version v1beta2 --kind Deployment --core-type

  # Fill the required fileds of Deployment object so that the Deployment instance can be successfully created
    update_controller_test
}

function generate_resource_with_coretype_controller {
  header_text "generating CRD resource as well as controller for coretype Deployment"

  # Run the commands
  kubebuilder init repo --domain sample.kubernetes.io
  kubebuilder create resource --group ant --version v1beta1 --kind Ant
  kubebuilder create controller --group apps --version v1beta2 --kind Deployment --core-type

  # Fill the required fileds of Deployment object so that the Deployment instance can be successfully created
  update_controller_test
}

function test_plural_resource {
  header_text "generating CRD for plural resource"

  kubebuilder create resource --plural-kind=true --group testing --version v1beta1 --kind Metadata
  kubebuilder create resource --group testing --version v1beta1 --kind Postgress
}

prepare_staging_dir
fetch_tools
build_kb

prepare_testdir_under_gopath
generate_crd_resources
generate_controller
test_docs
test_generated_controller
test_vendor_update
# re-running controller tests post vendor update
test_generated_controller

prepare_testdir_under_gopath
generate_resource_with_coretype_controller
test_generated_controller

prepare_testdir_under_gopath
generate_coretype_controller
test_generated_controller

test_plural_resource

exit $rc
