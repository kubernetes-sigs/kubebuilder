/*
Copyright 2019 The Kubernetes Authors.

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

package v2

import (
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

var _ input.File = &Makefile{}

// Makefile scaffolds the Makefile
type Makefile struct {
	input.Input
	// Image is controller manager image name
	Image string
}

// GetInput implements input.File
func (c *Makefile) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = "Makefile"
	}
	if c.Image == "" {
		c.Image = "controller:latest"
	}
	c.TemplateBody = makefileTemplate
	c.Input.IfExistsAction = input.Error
	return c.Input, nil
}

var makefileTemplate = `
# Image URL to use all building/pushing image targets
IMG ?= {{ .Image }}
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
test: generate lint manifests
	go test ./api/... ./controllers/... -coverprofile cover.out

# Build manager binary
manager: generate lint
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate lint
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crd/bases

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -f config/crd/bases
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run golangci-lint against code
lint: $(GOPATH)/bin/golangci-lint
	golangci-lint run -E gofmt ./...

$(GOPATH)/bin/golangci-lint:
	go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths=./api/...

# Build the docker image
docker-build: test
	docker build . -t ${IMG}
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/manager_image_patch.yaml

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.0-beta.3
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
`
