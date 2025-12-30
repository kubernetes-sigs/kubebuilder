#!/usr/bin/env bash

#  Copyright 2025 The Kubernetes Authors.
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

# This script sets up a Kind cluster for the Kubebuilder demo
# It reuses the e2e test infrastructure for consistency

set -e

SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_ROOT}/../.." && pwd)"

# Source the e2e test infrastructure
source "${PROJECT_ROOT}/test/common.sh"
source "${PROJECT_ROOT}/test/e2e/setup.sh"

echo "Setting up Kind cluster for Kubebuilder demo..."

# Set the cluster name for the demo
export KIND_CLUSTER="kubebuilder-demo"

# Build kubebuilder, fetch tools, and install kind if needed
build_kb
fetch_tools
install_kind

# Create the cluster using the e2e setup function
create_cluster ${KIND_K8S_VERSION}

echo ""
echo "Kind cluster is ready for the Kubebuilder demo!"
echo "Cluster name: ${KIND_CLUSTER}"
echo "Context: kind-${KIND_CLUSTER}"
echo ""
echo "To delete the cluster when done:"
echo "   make clean-demo"
