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

cp -r /workspace/_output/kubebuilder /tmp/kubebuilder/

# Tests won't work on darwin
if [ $GOOS = "linux" ]; then

export GOPATH=/go
mkdir -p $GOPATH/src/github.com/kubernetes-sigs/kubebuilder-test/
cd $GOPATH/src/github.com/kubernetes-sigs/kubebuilder-test/

# Setup env vars
export PATH=$PATH:/tmp/kubebuilder/bin/
export TEST_ASSET_KUBECTL=/tmp/kubebuilder/bin/kubectl
export TEST_ASSET_KUBE_APISERVER=/tmp/kubebuilder/bin/kube-apiserver
export TEST_ASSET_ETCD=/tmp/kubebuilder/bin/etcd

# Run the commands
kubebuilder init repo --domain sample.kubernetes.io
kubebuilder create resource --group insect --version v1beta1 --kind Bee
kubebuilder create resource --group insect --version v1beta1 --kind Wasp
kubebuilder create controller --group apps --version v1beta2 --kind Deployment --core-type

# Verify the controller-manager builds and the tests pass
go build ./cmd/...
go build ./pkg/...
go test ./cmd/...
go test ./pkg/...

fi