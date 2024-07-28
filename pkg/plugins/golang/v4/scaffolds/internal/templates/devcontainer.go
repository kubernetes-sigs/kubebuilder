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
  "image": "golang:1.22",
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

curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-amd64
chmod +x ./kind
mv ./kind /usr/local/bin/kind

curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/linux/amd64
chmod +x kubebuilder
mv kubebuilder /usr/local/bin/

KUBECTL_VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt)
curl -LO "https://dl.k8s.io/release/$KUBECTL_VERSION/bin/linux/amd64/kubectl"
chmod +x kubectl
mv kubectl /usr/local/bin/kubectl

docker network create -d=bridge --subnet=172.19.0.0/24 kind

kind version
kubebuilder version
docker --version
go version
kubectl version --client
`

var _ machinery.Template = &DevContainer{}
var _ machinery.Template = &DevContainerPostInstallScript{}

// DevCotaniner scaffoldds a `devcontainer.json` configurations file for
// creating Kubebuilder & Kind based DevContainer.
type DevContainer struct {
	machinery.TemplateMixin
}

type DevContainerPostInstallScript struct {
	machinery.TemplateMixin
}

func (f *DevContainer) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = ".devcontainer/devcontainer.json"
	}

	f.TemplateBody = devContainerTemplate

	return nil
}

func (f *DevContainerPostInstallScript) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = ".devcontainer/post-install.sh"
	}

	f.TemplateBody = postInstallScript

	return nil
}
