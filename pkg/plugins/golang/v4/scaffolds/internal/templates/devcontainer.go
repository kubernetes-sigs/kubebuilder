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

const devContainerTemplate = `{
  "name": "Kubebuilder DevContainer",
  "image": "golang:1.25",
  "features": {
    "ghcr.io/devcontainers/features/docker-in-docker:2": {},
    "ghcr.io/devcontainers/features/git:1": {}
  },

  "runArgs": ["--network=host"],

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

  "onCreateCommand": "bash .devcontainer/post-install.sh"
}

`

const postInstallScript = `#!/bin/bash
set -x

curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-$(go env GOARCH)
chmod +x ./kind
mv ./kind /usr/local/bin/kind

BEGIN_MARKER="# BEGIN kind autocompletion"
END_MARKER="# END kind autocompletion"
if ! grep -q "$BEGIN_MARKER" "$BASHRC_FILE"; then
    echo ""
    echo "" >> "$BASHRC_FILE"
    echo "$BEGIN_MARKER" >> "$BASHRC_FILE"
    echo "# kind autocompletion" >> "$BASHRC_FILE"
    echo "if [ -f /usr/local/share/bash-completion/bash_completion ]; then" >> "$BASHRC_FILE"
    echo ". /usr/local/share/bash-completion/bash_completion" >> "$BASHRC_FILE"
    echo "fi" >> "$BASHRC_FILE"
    echo ". <(kind completion bash)" >> "$BASHRC_FILE"
    echo "$END_MARKER" >> "$BASHRC_FILE"
    echo "Kind autocompletion added to $BASHRC_FILE"
    echo ""
else
    echo "Kind autocompletion already exists in $BASHRC_FILE"
fi

curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/linux/$(go env GOARCH)
chmod +x kubebuilder
mv kubebuilder /usr/local/bin/

BEGIN_MARKER="# BEGIN kubebuilder autocompletion"
END_MARKER="# END kubebuilder autocompletion"
if ! grep -q "$BEGIN_MARKER" "$BASHRC_FILE"; then
    echo ""
    echo "" >> "$BASHRC_FILE"
    echo "$BEGIN_MARKER" >> "$BASHRC_FILE"
    echo "# kubebuilder autocompletion" >> "$BASHRC_FILE"
    echo "if [ -f /usr/local/share/bash-completion/bash_completion ]; then" >> "$BASHRC_FILE"
    echo ". /usr/local/share/bash-completion/bash_completion" >> "$BASHRC_FILE"
    echo "fi" >> "$BASHRC_FILE"
    echo ". <(kubebuilder completion bash)" >> "$BASHRC_FILE"
    echo "$END_MARKER" >> "$BASHRC_FILE"
    echo "Kubebuilder autocompletion added to $BASHRC_FILE"
    echo ""
else
    echo "Kubebuilder autocompletion already exists in $BASHRC_FILE"
fi

KUBECTL_VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt)
curl -LO "https://dl.k8s.io/release/$KUBECTL_VERSION/bin/linux/$(go env GOARCH)/kubectl"
chmod +x kubectl
mv kubectl /usr/local/bin/kubectl

BEGIN_MARKER="# BEGIN kubectl autocompletion"
END_MARKER="# END kubectl autocompletion"
if ! grep -q "$BEGIN_MARKER" "$BASHRC_FILE"; then
    echo ""
    echo "" >> "$BASHRC_FILE"
    echo "$BEGIN_MARKER" >> "$BASHRC_FILE"
    echo "# kubectl autocompletion" >> "$BASHRC_FILE"
    echo "if [ -f /usr/local/share/bash-completion/bash_completion ]; then" >> "$BASHRC_FILE"
    echo ". /usr/local/share/bash-completion/bash_completion" >> "$BASHRC_FILE"
    echo "fi" >> "$BASHRC_FILE"
    echo ". <(kubectl completion bash)" >> "$BASHRC_FILE"
    echo "$END_MARKER" >> "$BASHRC_FILE"
    echo "Kubectl autocompletion added to $BASHRC_FILE"
    echo ""
else
    echo "Kubectl autocompletion already exists in $BASHRC_FILE"
fi

BEGIN_MARKER="# BEGIN docker autocompletion"
END_MARKER="# END docker autocompletion"
if ! grep -q "$BEGIN_MARKER" "$BASHRC_FILE"; then
    echo ""
    echo "" >> "$BASHRC_FILE"
    echo "$BEGIN_MARKER" >> "$BASHRC_FILE"
    echo "# docker autocompletion" >> "$BASHRC_FILE"
    echo "if [ -f /usr/local/share/bash-completion/bash_completion ]; then" >> "$BASHRC_FILE"
    echo ". /usr/local/share/bash-completion/bash_completion" >> "$BASHRC_FILE"
    echo "fi" >> "$BASHRC_FILE"
    echo ". <(docker completion bash)" >> "$BASHRC_FILE"
    echo "$END_MARKER" >> "$BASHRC_FILE"
    echo "Docker autocompletion added to $BASHRC_FILE"
    echo ""
else
    echo "Docker autocompletion already exists in $BASHRC_FILE"
fi

docker network create -d=bridge --subnet=172.19.0.0/24 kind

kind version
kubebuilder version
docker --version
go version
kubectl version --client
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
