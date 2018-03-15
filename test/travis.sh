#!/usr/bin/env bash

set -x -e

test/scripts/build_kubebuilder.sh

#go build ./cmd/...
#go build ./pkg/...
#go test ./cmd/...
#go test ./pkg/...

# Create the working directory to test the repo setup
export GOPATH=/tmp/go
mkdir -p $GOPATH/src/github.com/kubernetes-sigs/kubebuilder-test/
cd $GOPATH/src/github.com/kubernetes-sigs/kubebuilder-test/

# Run the commands
/tmp/kubebuilder/bin/kubebuilder init repo --domain sample.kubernetes.io
/tmp/kubebuilder/bin/kubebuilder create resource --group insect --version v1beta1 --kind Bee
#/tmp/kubebuilder/bin/kubebuilder create resource --group insect --version v1beta1 --kind Wasp

export TEST_ASSET_KUBECTL=/tmp/kubebuilder/bin/kubectl
export TEST_ASSET_KUBE_APISERVER=/tmp/kubebuilder/bin/kube-apiserver
export TEST_ASSET_ETCD=/tmp/kubebuilder/bin/etcd

# Verify the controller-manager builds and the tests pass
go install github.com/kubernetes-sigs/kubebuilder-test/cmd/controller-manager
go build ./cmd/...
go build ./pkg/...
go test ./cmd/...
go test ./pkg/...
