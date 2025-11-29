/*
Copyright 2022 The Kubernetes Authors.

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

var _ machinery.Template = &Makefile{}

// Makefile scaffolds a file that defines project management CLI commands
type Makefile struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// Image is controller manager image name
	Image string
	// BoilerplatePath is the path to the boilerplate file
	BoilerplatePath string
	// Controller tools version to use in the project
	ControllerToolsVersion string
	// Kustomize version to use in the project
	KustomizeVersion string
	// golangci-lint version to use in the project
	GolangciLintVersion string
	// ControllerRuntimeVersion version to be used to download the envtest setup script
	ControllerRuntimeVersion string
	// EnvtestVersion store the name of the verions to be used to install setup-envtest
	EnvtestVersion string
}

// SetTemplateDefaults implements machinery.Template
func (f *Makefile) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = "Makefile"
	}

	f.TemplateBody = makefileTemplate

	f.IfExistsAction = machinery.Error

	if f.Image == "" {
		f.Image = "controller:latest"
	}

	// TODO: Current workaround for setup-envtest compatibility
	// Due to past instances where controller-runtime maintainers released
	// versions without corresponding branches, directly relying on branches
	// poses a risk of breaking the Kubebuilder chain. Such practices may
	// change over time, potentially leading to compatibility issues. This
	// approach, although not ideal, remains the best solution for ensuring
	// compatibility with controller-runtime releases as of now. For more
	// details on the quest for a more robust solution, refer to the issue
	// raised in the controller-runtime repository: https://github.com/kubernetes-sigs/controller-runtime/issues/2744
	if f.EnvtestVersion == "" {
		f.EnvtestVersion = "latest"
	}
	return nil
}

//nolint:lll
const makefileTemplate = `# Image URL to use all building/pushing image targets
IMG ?= {{ .Image }}

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	"$(CONTROLLER_GEN)" rbac:roleName=manager-role crd webhook paths="$(CONTROLLER_GEN_PATHS)" output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	{{ if .BoilerplatePath -}}
	"$(CONTROLLER_GEN)" object:headerFile={{printf "%q" .BoilerplatePath}} paths="$(CONTROLLER_GEN_PATHS)"
	{{- else -}}
	"$(CONTROLLER_GEN)" object paths="$(CONTROLLER_GEN_PATHS)"
	{{- end }}

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet setup-envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell "$(ENVTEST)" use $(ENVTEST_K8S_VERSION) --bin-dir "$(LOCALBIN)" -p path)" go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

# TODO(user): To use a different vendor for e2e tests, modify the setup under 'tests/e2e'.
# The default setup assumes Kind is pre-installed and builds/loads the Manager Docker image locally.
# CertManager is installed by default; skip with:
# - CERT_MANAGER_INSTALL_SKIP=true
KIND_CLUSTER ?= {{ .ProjectName }}-test-e2e

.PHONY: setup-test-e2e
setup-test-e2e: ## Set up a Kind cluster for e2e tests if it does not exist
	@command -v $(KIND) >/dev/null 2>&1 || { \
		echo "Kind is not installed. Please install Kind manually."; \
		exit 1; \
	}
	@case "$$($(KIND) get clusters)" in \
		*"$(KIND_CLUSTER)"*) \
			echo "Kind cluster '$(KIND_CLUSTER)' already exists. Skipping creation." ;; \
		*) \
			echo "Creating Kind cluster '$(KIND_CLUSTER)'..."; \
			$(KIND) create cluster --name $(KIND_CLUSTER) ;; \
	esac

.PHONY: test-e2e
test-e2e: setup-test-e2e manifests generate fmt vet ## Run the e2e tests. Expected an isolated environment using Kind.
	KIND=$(KIND) KIND_CLUSTER=$(KIND_CLUSTER) go test -tags=e2e ./test/e2e/ -v -ginkgo.v
	$(MAKE) cleanup-test-e2e

.PHONY: cleanup-test-e2e
cleanup-test-e2e: ## Tear down the Kind cluster used for e2e tests
	@$(KIND) delete cluster --name $(KIND_CLUSTER)

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter
	"$(GOLANGCI_LINT)" run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	"$(GOLANGCI_LINT)" run --fix

.PHONY: lint-config
lint-config: golangci-lint ## Verify golangci-lint linter configuration
	"$(GOLANGCI_LINT)" config verify

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name {{ .ProjectName }}-builder
	$(CONTAINER_TOOL) buildx use {{ .ProjectName }}-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm {{ .ProjectName }}-builder
	rm Dockerfile.cross

.PHONY: build-installer
build-installer: manifests generate kustomize ## Generate a consolidated YAML with CRDs and deployment.
	mkdir -p dist
	cd config/manager && "$(KUSTOMIZE)" edit set image controller=${IMG}
	"$(KUSTOMIZE)" build config/default > dist/install.yaml

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	@out="$$( "$(KUSTOMIZE)" build config/crd 2>/dev/null || true )"; \
	if [ -n "$$out" ]; then echo "$$out" | "$(KUBECTL)" apply -f -; else echo "No CRDs to install; skipping."; fi

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	@out="$$( "$(KUSTOMIZE)" build config/crd 2>/dev/null || true )"; \
	if [ -n "$$out" ]; then echo "$$out" | "$(KUBECTL)" delete --ignore-not-found=$(ignore-not-found) -f -; else echo "No CRDs to delete; skipping."; fi

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && "$(KUSTOMIZE)" edit set image controller=${IMG}
	"$(KUSTOMIZE)" build config/default | "$(KUBECTL)" apply -f -

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	"$(KUSTOMIZE)" build config/default | "$(KUBECTL)" delete --ignore-not-found=$(ignore-not-found) -f -

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(CURDIR)/bin

# WINDOWS_CYGWIN_FIX: Convert Cygwin/Git Bash paths to Windows paths for Go tools
# This is necessary because:
# - Cygwin make uses /cygdrive/c/... paths (CYGWIN_NT)
# - Git Bash (MINGW64) uses /c/... paths (MINGW64_NT) but make might use /cygdrive/c/...
# - Windows Go expects C:/... paths
# We handle both formats with two sed expressions
define convert-to-windows-path
$(shell echo "$(1)" | sed -e 's|^/cygdrive/\([a-z]\)|\1:|' -e 's|^/\([a-z]\)/|\1:/|' 2>/dev/null || echo "$(1)")
endef

# WINDOWS_CYGWIN_FIX: Convert Windows path to MINGW/Cygwin path for shell commands
# In MINGW64, we need /d/path format, not /cygdrive/d/path
define convert-to-mingw-path
$(shell echo "$(1)" | sed -e 's|^/cygdrive/\([a-z]\)|\1:|' -e 's|^\([a-z]\):|/\1|' 2>/dev/null || echo "$(1)")
endef

# WINDOWS_CYGWIN_FIX: Detect if we're in MINGW/Cygwin environment
UNAME_S := $(shell uname -s 2>/dev/null || echo "")
IS_MINGW := $(shell echo "$(UNAME_S)" | grep -qi mingw && echo "yes" || echo "")
IS_CYGWIN := $(shell echo "$(UNAME_S)" | grep -qi cygwin && echo "yes" || echo "")

LOCALBIN_FOR_GO := $(call convert-to-windows-path,$(LOCALBIN))

# WINDOWS_CYGWIN_FIX: For MINGW64, convert paths for shell commands
ifeq ($(IS_MINGW),yes)
LOCALBIN_FOR_SHELL := $(call convert-to-mingw-path,$(LOCALBIN))
else
LOCALBIN_FOR_SHELL := $(LOCALBIN)
endif

$(LOCALBIN):
	mkdir -p "$(LOCALBIN)"

## Tool Binaries
KUBECTL ?= kubectl
KIND ?= kind
# WINDOWS_CYGWIN_FIX: Add .exe extension on Windows and use correct path format
ifeq ($(OS),Windows_NT)
KUSTOMIZE ?= $(LOCALBIN_FOR_SHELL)/kustomize.exe
CONTROLLER_GEN ?= $(LOCALBIN_FOR_SHELL)/controller-gen.exe
ENVTEST ?= $(LOCALBIN_FOR_SHELL)/setup-envtest.exe
GOLANGCI_LINT = $(LOCALBIN_FOR_SHELL)/golangci-lint.exe
else ifeq ($(IS_MINGW),yes)
KUSTOMIZE ?= $(LOCALBIN_FOR_SHELL)/kustomize.exe
CONTROLLER_GEN ?= $(LOCALBIN_FOR_SHELL)/controller-gen.exe
ENVTEST ?= $(LOCALBIN_FOR_SHELL)/setup-envtest.exe
GOLANGCI_LINT = $(LOCALBIN_FOR_SHELL)/golangci-lint.exe
else ifeq ($(IS_CYGWIN),yes)
KUSTOMIZE ?= $(LOCALBIN)/kustomize.exe
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen.exe
ENVTEST ?= $(LOCALBIN)/setup-envtest.exe
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint.exe
else
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
endif

# WINDOWS_CYGWIN_FIX: Get Go module name for controller-gen paths
# On Windows/Cygwin, controller-gen cannot resolve relative paths like "./..." correctly
# because it's a Windows binary running in a Cygwin environment. We use the module name instead.
GO_MODULE := $(shell go list -m 2>/dev/null || echo "")
CONTROLLER_GEN_PATHS := $(if $(GO_MODULE),$(GO_MODULE)/...,./...)

## Tool Versions
KUSTOMIZE_VERSION ?= {{ .KustomizeVersion }}
CONTROLLER_TOOLS_VERSION ?= {{ .ControllerToolsVersion }}

#ENVTEST_VERSION is the version of controller-runtime release branch to fetch the envtest setup script (i.e. release-0.20)
ENVTEST_VERSION ?= $(shell v='$(call gomodver,sigs.k8s.io/controller-runtime)'; \
  [ -n "$$v" ] || { echo "Set ENVTEST_VERSION manually (controller-runtime replace has no tag)" >&2; exit 1; }; \
  printf '%s\n' "$$v" | sed -E 's/^v?([0-9]+)\.([0-9]+).*/release-\1.\2/')

#ENVTEST_K8S_VERSION is the version of Kubernetes to use for setting up ENVTEST binaries (i.e. 1.31)
ENVTEST_K8S_VERSION ?= $(shell v='$(call gomodver,k8s.io/api)'; \
  [ -n "$$v" ] || { echo "Set ENVTEST_K8S_VERSION manually (k8s.io/api replace has no tag)" >&2; exit 1; }; \
  printf '%s\n' "$$v" | sed -E 's/^v?[0-9]+\.([0-9]+).*/1.\1/')

GOLANGCI_LINT_VERSION ?= {{ .GolangciLintVersion }}
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	@$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	@$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: setup-envtest
setup-envtest: envtest ## Download the binaries required for ENVTEST in the local bin directory.
	@echo "Setting up envtest binaries for Kubernetes version $(ENVTEST_K8S_VERSION)..."
	@"$(ENVTEST)" use $(ENVTEST_K8S_VERSION) --bin-dir "$(LOCALBIN)" -p path || { \
		echo "Error: Failed to set up envtest binaries for version $(ENVTEST_K8S_VERSION)."; \
		exit 1; \
	}

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	@$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	@$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (may include .exe extension in MINGW/Cygwin)
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || [ -f "$(1)-$(3).exe" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f "$(1)" "$(1).exe" "$(1)-$(3)" "$(1)-$(3).exe" ;\
GOBIN="$(LOCALBIN_FOR_GO)" go install $${package} ;\
binary_name=$$(basename "$(1)" .exe) ;\
bin_dir=$$(dirname "$(1)") ;\
if [ -f "$${bin_dir}/$${binary_name}.exe" ]; then \
	mv "$${bin_dir}/$${binary_name}.exe" "$${bin_dir}/$${binary_name}-$(3).exe" ;\
	cp "$${bin_dir}/$${binary_name}-$(3).exe" "$${bin_dir}/$${binary_name}.exe" ;\
elif [ -f "$${bin_dir}/$${binary_name}" ]; then \
	mv "$${bin_dir}/$${binary_name}" "$${bin_dir}/$${binary_name}-$(3)" ;\
	ln -sf "$${binary_name}-$(3)" "$${bin_dir}/$${binary_name}" 2>/dev/null || cp "$${bin_dir}/$${binary_name}-$(3)" "$${bin_dir}/$${binary_name}" ;\
fi ;\
}
endef

define gomodver
$(shell go list -m -f '{{"{{"}}if .Replace{{"}}"}}{{"{{"}}.Replace.Version{{"}}"}}{{"{{"}}else{{"}}"}}{{"{{"}}.Version{{"}}"}}{{"{{"}}end{{"}}"}}' $(1) 2>/dev/null)
endef
`
