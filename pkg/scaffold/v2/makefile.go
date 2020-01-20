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
	// Controller tools version to use in the project
	ControllerToolsVersion string
}

// GetInput implements input.File
func (f *Makefile) GetInput() (input.Input, error) {
	if f.Path == "" {
		f.Path = "Makefile"
	}
	if f.Image == "" {
		f.Image = "controller:latest"
	}
	f.TemplateBody = makefileTemplate
	f.Input.IfExistsAction = input.Error
	return f.Input, nil
}

// nolint:lll
const makefileTemplate = `
# Image URL to use all building/pushing image targets
IMG ?= {{ .Image }}
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

all: manager

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile={{printf "%q" .BoilerplatePath}} paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
  # Get CONTROLLER_GEN from one of the possibly locations, ordered by priority
  @CONTROLLER_GEN=$(shell PATH=$$PATH:$$GOPATH/bin:$$GOBIN:$$GOPATH/bin:$$HOME/go/bin which controller-gen)
ifeq (, $(CONTROLLER_GEN))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@{{.ControllerToolsVersion}} ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
  @CONTROLLER_GEN=$(shell PATH=$$PATH:$$GOPATH/bin:$$GOBIN:$$GOPATH/bin:$$HOME/go/bin which controller-gen)
endif
`
