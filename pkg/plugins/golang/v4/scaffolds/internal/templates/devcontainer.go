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

echo "Installing Kubebuilder development tools..."

# Verify running as root (required for installing to /usr/local/bin and /etc)
if [ "$(id -u)" -ne 0 ]; then
  echo "ERROR: This script must be run as root"
  exit 1
fi

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

BASH_COMPLETIONS_DIR="/usr/share/bash-completion/completions"

# Enable bash-completion in .bashrc
if ! grep -q "source /usr/share/bash-completion/bash_completion" ~/.bashrc 2>/dev/null; then
  echo 'source /usr/share/bash-completion/bash_completion' >> ~/.bashrc
  echo "Added bash-completion to .bashrc"
fi

# Install kind
if ! command -v kind &> /dev/null; then
  TMP_KIND=$(mktemp)
  curl -Lo "${TMP_KIND}" "https://kind.sigs.k8s.io/dl/latest/kind-linux-${ARCH}"
  chmod +x "${TMP_KIND}"
  mv "${TMP_KIND}" /usr/local/bin/kind
fi
kind completion bash > "${BASH_COMPLETIONS_DIR}/kind" 2>/dev/null || true

# Install kubebuilder
if ! command -v kubebuilder &> /dev/null; then
  TMP_KB=$(mktemp)
  curl -L -o "${TMP_KB}" "https://go.kubebuilder.io/dl/latest/linux/${ARCH}"
  chmod +x "${TMP_KB}"
  mv "${TMP_KB}" /usr/local/bin/kubebuilder
fi
kubebuilder completion bash > "${BASH_COMPLETIONS_DIR}/kubebuilder" 2>/dev/null || true

# Install kubectl
if ! command -v kubectl &> /dev/null; then
  KUBECTL_VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt)
  TMP_KUBECTL=$(mktemp)
  curl -Lo "${TMP_KUBECTL}" "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${ARCH}/kubectl"
  chmod +x "${TMP_KUBECTL}"
  mv "${TMP_KUBECTL}" /usr/local/bin/kubectl
fi
kubectl completion bash > "${BASH_COMPLETIONS_DIR}/kubectl" 2>/dev/null || true

# Docker completion
docker completion bash > "${BASH_COMPLETIONS_DIR}/docker" 2>/dev/null || true

# Wait for Docker to be ready
for i in {1..30}; do
  if docker info >/dev/null 2>&1; then
    break
  fi
  if [ $i -eq 30 ]; then
    echo "WARNING: Docker not ready after 30s"
  fi
  sleep 1
done

# Create kind network, ignore errors if exists or conflicts
docker network inspect kind >/dev/null 2>&1 || docker network create kind || true

# Verify installations
echo "Installed versions:"
kind version
kubebuilder version
kubectl version --client
docker --version
go version

echo "DevContainer ready!"
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
