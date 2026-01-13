#!/bin/bash

set -e

# Copyright 2021 The Kubernetes Authors.
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

echo "Setting up test environment..."

# Create a directory named testplugin for running kubebuilder commands
mkdir -p testdata/testplugin
cd testdata/testplugin
rm -rf *

# Run Kubebuilder commands inside the testplugin directory
kubebuilder init --plugins go/v4 --domain sample.domain.com --repo sample.domain.com/test-operator
kubebuilder edit --plugins sampleexternalplugin/v1

# Ensure Prometheus assets were scaffolded
test -f config/prometheus/prometheus.yaml
test -f config/prometheus/kustomization.yaml
test -f config/default/kustomization_prometheus_patch.yaml

# Clean up test files only on success (set -e ensures we exit on failure)
echo "All tests passed!"
cd ../..
rm -rf testdata/testplugin
