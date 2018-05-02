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

mkdir -p /workspace/vendor/github.com/kubernetes-sigs/kubebuilder/pkg/ || echo ""
cp -r /workspace/pkg/* /workspace/vendor/github.com/kubernetes-sigs/kubebuilder/pkg/
cp /workspace/LICENSE /workspace/vendor/github.com/kubernetes-sigs/kubebuilder/LICENSE

export DEST=/workspace/_output/kubebuilder/bin/
mkdir -p $DEST || echo ""
tar -czvf $DEST/vendor.tar.gz vendor/ Gopkg.lock  Gopkg.toml
