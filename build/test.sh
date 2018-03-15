#!/usr/bin/env bash

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

# Verify the controller-manager builds and the tests pass
go build ./cmd/...
go build ./pkg/...
go test ./cmd/...
go test ./pkg/...

fi