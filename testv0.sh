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

function generate_crd_resources {
  header_text "generating CRD resources and code"

  # Run the commands
  kubebuilder init repo --domain sample.kubernetes.io --project-version v0
  kubebuilder create resource --group insect --version v1beta1 --kind Bee

  header_text "editing generated files to simulate a user"
  sed -i -e '/type Bee struct/ i \
  // +kubebuilder:categories=foo,bar\
  // +kubebuilder:subresource:status\
  // +kubebuilder:subresource:scale
  ' pkg/apis/insect/v1beta1/bee_types.go

  sed -i -e '/type BeeController struct {/ i \
  // +kubebuilder:rbac:groups="",resources=pods,verbs=get;watch;list
  ' pkg/controller/bee/controller.go

  header_text "adding a map type to resource"
  sed -i -e '/type BeeSpec struct {/ a \
  Request map[string]string \`json:\"request,omitempty\"\`
  ' pkg/apis/insect/v1beta1/bee_types.go

  header_text "generating and testing CRD definition"
  kubebuilder create config --crds --output crd.yaml
  kubebuilder create config --controller-image myimage:v1 --name myextensionname --output install.yaml

  # Test for the expected generated CRD definition
  #
  # TODO: this is awkwardly inserted after the first resource created in this
  # test because the output order seems nondeterministic and it's preferable to
  # avoid introducing a new dependency like yq or complex parsing logic
  cat << EOF | diff crd.yaml -
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
  subresources:
    scale:
      specReplicasPath: ""
      statusReplicasPath: ""
    status: {}
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
          properties:
            request:
              type: object
          type: object
        status:
          type: object
  version: v1beta1
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: null
EOF

    cat << EOF | diff -B install.yaml -
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
  subresources:
    scale:
      specReplicasPath: ""
      statusReplicasPath: ""
    status: {}
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
          properties:
            request:
              type: object
          type: object
        status:
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


  kubebuilder create resource --group insect --version v1beta1 --kind Wasp
  kubebuilder create resource --group ant --version v1beta1 --kind Ant --controller=false
  kubebuilder create config --crds --output crd.yaml

  # Check for ordering of generated YAML
  # TODO: make this a more concise test in a follow-up
  cat << EOF | diff crd.yaml -
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
  subresources:
    scale:
      specReplicasPath: ""
      statusReplicasPath: ""
    status: {}
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
          properties:
            request:
              type: object
          type: object
        status:
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
  version: v1beta1
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: null
EOF
}

function test_crd_validation {
  header_text "testing crd validation"

  # Setup env vars
  export PATH=/tmp/kubebuilder/bin/:$PATH
  export TEST_ASSET_KUBECTL=/tmp/kubebuilder/bin/kubectl
  export TEST_ASSET_KUBE_APISERVER=/tmp/kubebuilder/bin/kube-apiserver
  export TEST_ASSET_ETCD=/tmp/kubebuilder/bin/etcd

  kubebuilder init repo --domain sample.kubernetes.io --project-version v0
  kubebuilder create resource --group got --version v1beta1 --kind House

  # Update crd
  sed -i -e '/type HouseSpec struct/ a \
    // +kubebuilder:validation:Maximum=100\
    // +kubebuilder:validation:ExclusiveMinimum=true\
    Power float32 \`json:"power,omitempty"\`\
    Bricks int32 \`json:"bricks,omitempty"\`\
    // +kubebuilder:validation:MaxLength=15\
    // +kubebuilder:validation:MinLength=1\
    Name string \`json:"name,omitempty"\`\
    // +kubebuilder:validation:MaxItems=500\
    // +kubebuilder:validation:MinItems=1\
    // +kubebuilder:validation:UniqueItems=false\
    Knights []string \`json:"knights,omitempty"\`\
    Winner bool \`json:"winner,omitempty"\`\
    // +kubebuilder:validation:Enum=Lion,Wolf,Dragon\
    Alias string \`json:"alias,omitempty"\`\
    // +kubebuilder:validation:Enum=1,2,3\
    Rank int \`json:"rank"\`\
    Comment []byte \`json:"comment,omitempty"\`\
  ' pkg/apis/got/v1beta1/house_types.go

  header_text "calling kubebuilder generate"
  kubebuilder generate
  header_text "generating and testing CRD..."
  kubebuilder create config --crds --output crd-validation.yaml
  diff crd-validation.yaml $kb_orig/test/data/resource/expected/crd-expected.yaml

  kubebuilder create config --controller-image myimage:v1 --name myextensionname --output install.yaml
  kubebuilder create controller --group got --version v1beta1 --kind House

  header_text "update controller"
  sed -i -e '/instance.Name = "instance-1"/ a \
        instance.Spec=HouseSpec{Power:89.5,Knights:[]string{"Jaime","Bronn","Gregor Clegane"}, Alias:"Lion", Name:"Lannister", Rank:1}
  ' ./pkg/apis/got/v1beta1/house_types_test.go
  sed -i -e '/instance.Name = "instance-1"/ a \
        instance.Spec=HouseSpec{Power:89.5,Knights:[]string{"Jaime","Bronn","Gregor Clegane"}, Alias:"Lion", Name:"Lannister", Rank:1}
  ' pkg/controller/house/controller_test.go
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
  diff docs/reference/includes "$kb_orig/test/data/docs/expected/includes"
  diff docs/reference/manifest.json "$kb_orig/test/data/docs/expected/manifest.json"
  diff docs/reference/config.yaml "$kb_orig/test/data/docs/expected/config.yaml"

  header_text "testing doc annotations"
  sed -i -e '/type Bee struct/ i \
  // +kubebuilder:doc:note=test notes message annotations\
  // +kubebuilder:doc:warning=test warnings message annotations
  ' pkg/apis/insect/v1beta1/bee_types.go

  kubebuilder docs --brodocs=false --cleanup=false
  diff docs/reference/config.yaml "$kb_orig/test/data/docs/expected/config-annotated.yaml"
}

function generate_controller {
  header_text "creating controller"
  kubebuilder create controller --group ant --version v1beta1 --kind Ant
}

function update_controller_test {
  # Update import
  sed -i -e '/"k8s.io\/client-go\/kubernetes\/typed\/apps\/v1beta2"/ a \
  "k8s.io/api/core/v1"
  ' ./pkg/controller/deployment/controller_test.go

  # Fill deployment instance
  sed -i -e '/instance.Name = "instance-1"/ a \
  instance.Spec.Template.Spec.Containers = []v1.Container{{Name: "name", Image: "someimage"}}\
  labels := map[string]string{"foo": "bar"}\
  instance.Spec.Template.ObjectMeta.Labels = labels\
  instance.Spec.Selector = \&metav1.LabelSelector{MatchLabels: labels}
  ' ./pkg/controller/deployment/controller_test.go
}

function generate_coretype_controller {
  header_text "generating controller for coretype Deployment"

  # Run the commands
  kubebuilder init repo --domain sample.kubernetes.io --controller-only --project-version v0
  kubebuilder create controller --group apps --version v1beta2 --kind Deployment --core-type

  # Fill the required fileds of Deployment object so that the Deployment instance can be successfully created
  update_controller_test
}

function generate_resource_with_coretype_controller {
  header_text "generating CRD resource as well as controller for coretype Deployment"

  # Run the commands
  kubebuilder init repo --domain sample.kubernetes.io --project-version v0
  kubebuilder create resource --group ant --version v1beta1 --kind Ant
  kubebuilder create controller --group apps --version v1beta2 --kind Deployment --core-type

  # Fill the required fields of Deployment object so that the Deployment instance can be successfully created
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

setup_envs

prepare_testdir_under_gopath

test_crd_validation
test_generated_controller

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
