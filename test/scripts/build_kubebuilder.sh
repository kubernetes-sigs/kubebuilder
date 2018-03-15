#!/usr/bin/env bash

set -x -e

# Build binaries
export GOBIN=/tmp/kubebuilder/bin/
go install github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder-gen
go install github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder

export VENDOR_KB=/tmp/vendorbin/vendor/github.com/kubernetes-sigs/kubebuilder

# Build vendor tar
mkdir -p $VENDOR_KB/pkg/ || echo ""
cp -r vendor/* /tmp/vendorbin/vendor/
cp -r pkg/* $VENDOR_KB/pkg/
cp LICENSE $VENDOR_KB/LICENSE
cp Gopkg.lock /tmp/vendorbin
cp Gopkg.toml /tmp/vendorbin

# Copy the vendor tar to the installation directory
export DEST=/tmp/kubebuilder/bin/
mkdir -p $DEST || echo ""
cd /tmp/vendorbin
tar -czvf $DEST/vendor.tar.gz vendor/ Gopkg.lock  Gopkg.toml
