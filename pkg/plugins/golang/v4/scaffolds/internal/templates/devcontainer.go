/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package templates

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// devContainerTemplate defines the devcontainer.json configuration
// Works with VS Code, GitHub Codespaces, and other devcontainer-compatible tools
//
// Configuration choices:
//   - moby: false - Uses Docker CE instead of Moby, fixes DinD issues in Codespaces
//   - dockerDefaultAddressPool - Prevents subnet conflicts in shared/cloud environments
//   - --privileged - Required for Docker daemon to run inside container (DinD)
//   - --init - Properly handles zombie processes and signal forwarding
//   - GO111MODULE=on - Ensures Go modules work consistently
//   - Runs as root (golang:1.25 default) - no sudo needed in post-install script
const devContainerTemplate = `{
  "name": "Kubebuilder DevContainer",
  "image": "golang:1.25",
  "features": {
    "ghcr.io/devcontainers/features/docker-in-docker:2": {
      "moby": false,
      "dockerDefaultAddressPool": "base=172.30.0.0/16,size=24"
    },
    "ghcr.io/devcontainers/features/git:1": {},
    "ghcr.io/devcontainers/features/common-utils:2": {
      "upgradePackages": true
    }
  },

  "runArgs": ["--privileged", "--init"],

  "customizations": {
    "vscode": {
      "settings": {
        "terminal.integrated.shell.linux": "/bin/bash"
      },
      "extensions": [
        "ms-kubernetes-tools.vscode-kubernetes-tools",
        "ms-azuretools.vscode-docker"
      ]
    }
  },

  "remoteEnv": {
    "GO111MODULE": "on"
  },

  "onCreateCommand": "bash .devcontainer/post-install.sh"
}

`

const postInstallScript = `#!/bin/bash
set -euo pipefail

echo "===================================="
echo "Kubebuilder DevContainer Setup"
echo "===================================="

# Verify running as root (required for installing to /usr/local/bin and /etc)
if [ "$(id -u)" -ne 0 ]; then
  echo "ERROR: This script must be run as root"
  exit 1
fi

echo ""
echo "Detecting system architecture..."
# Detect architecture using uname
MACHINE=$(uname -m)
case "${MACHINE}" in
  x86_64)
    ARCH="amd64"
    ;;
  aarch64|arm64)
    ARCH="arm64"
    ;;
  *)
    echo "WARNING: Unsupported architecture ${MACHINE}, defaulting to amd64"
    ARCH="amd64"
    ;;
esac
echo "Architecture: ${ARCH}"

echo ""
echo "------------------------------------"
echo "Setting up bash completion..."
echo "------------------------------------"

BASH_COMPLETIONS_DIR="/usr/share/bash-completion/completions"

# Enable bash-completion in root's .bashrc (devcontainer runs as root)
if ! grep -q "source /usr/share/bash-completion/bash_completion" ~/.bashrc 2>/dev/null; then
  echo 'source /usr/share/bash-completion/bash_completion' >> ~/.bashrc
  echo "Added bash-completion to .bashrc"
fi

echo ""
echo "------------------------------------"
echo "Installing development tools..."
echo "------------------------------------"

# Install kind
if ! command -v kind &> /dev/null; then
  echo "Installing kind..."
  curl -Lo /usr/local/bin/kind "https://kind.sigs.k8s.io/dl/latest/kind-linux-${ARCH}"
  chmod +x /usr/local/bin/kind
  echo "kind installed successfully"
fi

# Generate kind bash completion
if command -v kind &> /dev/null; then
  if kind completion bash > "${BASH_COMPLETIONS_DIR}/kind" 2>/dev/null; then
    echo "kind completion installed"
  else
    echo "WARNING: Failed to generate kind completion"
  fi
fi

# Install kubebuilder
if ! command -v kubebuilder &> /dev/null; then
  echo "Installing kubebuilder..."
  curl -Lo /usr/local/bin/kubebuilder "https://go.kubebuilder.io/dl/latest/linux/${ARCH}"
  chmod +x /usr/local/bin/kubebuilder
  echo "kubebuilder installed successfully"
fi

# Generate kubebuilder bash completion
if command -v kubebuilder &> /dev/null; then
  if kubebuilder completion bash > "${BASH_COMPLETIONS_DIR}/kubebuilder" 2>/dev/null; then
    echo "kubebuilder completion installed"
  else
    echo "WARNING: Failed to generate kubebuilder completion"
  fi
fi

# Install kubectl
if ! command -v kubectl &> /dev/null; then
  echo "Installing kubectl..."
  KUBECTL_VERSION=$(curl -Ls https://dl.k8s.io/release/stable.txt)
  curl -Lo /usr/local/bin/kubectl "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${ARCH}/kubectl"
  chmod +x /usr/local/bin/kubectl
  echo "kubectl installed successfully"
fi

# Generate kubectl bash completion
if command -v kubectl &> /dev/null; then
  if kubectl completion bash > "${BASH_COMPLETIONS_DIR}/kubectl" 2>/dev/null; then
    echo "kubectl completion installed"
  else
    echo "WARNING: Failed to generate kubectl completion"
  fi
fi

# Generate Docker bash completion
if command -v docker &> /dev/null; then
  if docker completion bash > "${BASH_COMPLETIONS_DIR}/docker" 2>/dev/null; then
    echo "docker completion installed"
  else
    echo "WARNING: Failed to generate docker completion"
  fi
fi

echo ""
echo "------------------------------------"
echo "Configuring Docker environment..."
echo "------------------------------------"

# Wait for Docker to be ready
echo "Waiting for Docker to be ready..."
for i in {1..30}; do
  if docker info >/dev/null 2>&1; then
    echo "Docker is ready"
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "WARNING: Docker not ready after 30s"
  fi
  sleep 1
done

# Create kind network (ignore if already exists)
if ! docker network inspect kind >/dev/null 2>&1; then
  if docker network create kind >/dev/null 2>&1; then
    echo "Created kind network"
  else
    echo "WARNING: Failed to create kind network (may already exist)"
  fi
fi

echo ""
echo "------------------------------------"
echo "Verifying installations..."
echo "------------------------------------"
kind version
kubebuilder version
kubectl version --client
docker --version
go version

echo ""
echo "===================================="
echo "DevContainer ready!"
echo "===================================="
echo "All development tools installed successfully."
echo "You can now start building Kubernetes operators."
`

var (
	_ machinery.Template = &DevContainer{}
	_ machinery.Template = &DevContainerPostInstallScript{}
)

// DevContainer scaffoldds a `devcontainer.json` configurations file for creating Kubebuilder & Kind based DevContainer.
type DevContainer struct {
	machinery.TemplateMixin
}

// DevContainerPostInstallScript defines the scaffold that will be done with the post install script
type DevContainerPostInstallScript struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults set defaults for this template
func (f *DevContainer) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = ".devcontainer/devcontainer.json"
	}

	f.TemplateBody = devContainerTemplate

	return nil
}

// SetTemplateDefaults set the defaults of this template
func (f *DevContainerPostInstallScript) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = ".devcontainer/post-install.sh"
	}

	f.TemplateBody = postInstallScript

	return nil
}
