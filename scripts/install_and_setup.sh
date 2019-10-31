#!/usr/bin/env bash

# Copyright 2019 The Kubernetes Authors.
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

pushd "$( dirname "${BASH_SOURCE[0]}" )" 
DIR=$(pwd)
popd
source "$DIR/../common.sh"

K8S_VERSION=$1

fetch_kind

# You can use --image flag to specify the cluster version you want, e.g --image=kindest/node:v1.13.6, the supported version are listed at https://hub.docker.com/r/kindest/node/tags
kind create cluster --config "$DIR/../test/kind-config.yaml" --image=kindest/node:$K8S_VERSION
