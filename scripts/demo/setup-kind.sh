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

set -e

CLUSTER_NAME="kubebuilder-demo"

echo "Setting up Kind cluster for Kubebuilder demo..."

# Check if Kind is installed
if ! command -v kind &> /dev/null; then
    echo "Kind is not installed. Installing Kind..."
    # Install Kind based on OS
    case "$(uname -s)" in
        Darwin)
            if command -v brew &> /dev/null; then
                brew install kind
            else
                echo "Please install Homebrew first or install Kind manually: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
                exit 1
            fi
            ;;
        Linux)
            # For AMD64 / x86_64
            [ $(uname -m) = x86_64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
            # For ARM64
            [ $(uname -m) = aarch64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-arm64
            chmod +x ./kind
            sudo mv ./kind /usr/local/bin/kind
            ;;
        *)
            echo "Unsupported OS. Please install Kind manually: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
            exit 1
            ;;
    esac
fi

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo "kubectl is not installed. Please install kubectl first."
    echo "Visit: https://kubernetes.io/docs/tasks/tools/install-kubectl/"
    exit 1
fi

# Check if cluster already exists
if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo "Kind cluster '${CLUSTER_NAME}' already exists."
    echo "Switching kubectl context to existing cluster..."
    kubectl cluster-info --context kind-${CLUSTER_NAME}
else
    echo "Creating Kind cluster '${CLUSTER_NAME}'..."
    
    # Create Kind config for the demo
    cat > /tmp/kind-config.yaml <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: ${CLUSTER_NAME}
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
EOF

    # Create the cluster
    kind create cluster --config=/tmp/kind-config.yaml --wait=300s
    
    # Clean up config file
    rm /tmp/kind-config.yaml
    
    echo "Kind cluster '${CLUSTER_NAME}' created successfully!"
fi

# Verify cluster is ready
echo "Verifying cluster is ready..."
kubectl cluster-info --context kind-${CLUSTER_NAME}
kubectl get nodes --context kind-${CLUSTER_NAME}

echo ""
echo "Kind cluster is ready for the Kubebuilder demo!"
echo "Cluster name: ${CLUSTER_NAME}"
echo "Context: kind-${CLUSTER_NAME}"
echo ""
echo "To delete the cluster when done:"
echo "   kind delete cluster --name ${CLUSTER_NAME}"
