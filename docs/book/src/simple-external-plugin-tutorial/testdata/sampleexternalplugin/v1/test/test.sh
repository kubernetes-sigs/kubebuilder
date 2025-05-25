#!/bin/bash

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
kubebuilder init --plugins sampleexternalplugin/v1 --domain sample.domain.com
kubebuilder create api --plugins sampleexternalplugin/v1 --number 2 --group samplegroup --version v1 --kind SampleKind
