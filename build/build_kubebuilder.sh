#!/bin/bash
#
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

# Google Container Builder automatically checks out all the code under the /workspace directory,
# but we actually want it to under the correct expected package in the GOPATH (/go)
# - Create the directory to host the code that matches the expected GOPATH package locations
# - Use /go as the default GOPATH because this is what the image uses
# - Link our current directory (containing the source code) to the package location in the GOPATH
mkdir -p /go/src/github.com/kubernetes-sigs
ln -s $(pwd) /go/src/github.com/kubernetes-sigs/kubebuilder

# Create the output directory for the binaries we will build
# Make sure CGO is 0 so the binaries are statically compiled and linux which is necessary for cross compiling go
export CGO=0
export DEST=/workspace/_output/kubebuilder/bin/
mkdir -p $DEST || echo ""

# Explicitly set the values of the variables in package "X" using ldflag so that they are statically compiled into
# the "version" command
export X=github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/version
go build -o $DEST/kubebuilder \
 -ldflags "-X $X.kubeBuilderVersion=$VERSION -X $X.goos=$GOOS -X $X.goarch=$GOARCH -X $X.kubernetesVendorVersion=$KUBERNETES_VERSION" \
 github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder

# Also build the kubebuilder-gen command
go build -o $DEST/kubebuilder-gen github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder-gen
