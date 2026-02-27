#!/usr/bin/env bash

#  Copyright 2023 The Kubernetes Authors.
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

#
# Makefile with some common workflow for dev, build and test
#
export GOPROXY?=https://proxy.golang.org/

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

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
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

.PHONY: build
build: ## Build the project locally
	go build --trimpath -o bin/kubebuilder

.PHONY: install
install: build ## Build and install the binary with the current source code. Use it to test your changes locally.
	rm -f $(GOBIN)/kubebuilder
	cp ./bin/kubebuilder $(GOBIN)/kubebuilder

##@ Development

.PHONY: generate
generate: generate-testdata generate-docs ## Update/generate all mock data. You should run this commands to update the mock data after your changes.
	go mod tidy
	make remove-spaces

.PHONY: remove-spaces
remove-spaces:
	@echo "Removing trailing spaces"
	@bash -c ' \
		if sed --version 2>&1 | grep -q "GNU"; then \
			find . -type f -name "*.md" -exec sed -i "s/[[:space:]]*$$//" {} + || true; \
		else \
			find . -type f -name "*.md" -exec sed -i "" "s/[[:space:]]*$$//" {} + || true; \
		fi'

.PHONY: generate-testdata
generate-testdata: ## Update/generate the testdata in $GOPATH/src/sigs.k8s.io/kubebuilder
	chmod -R +w testdata/
	rm -rf testdata/
	./test/testdata/generate.sh

.PHONY: generate-docs
generate-docs: ## Update/generate the docs
	./hack/docs/generate.sh

.PHONY: generate-charts
generate-charts: build ## Re-generate the helm chart testdata and docs samples
	rm -rf testdata/project-v4-with-plugins/dist/chart
	rm -rf docs/book/src/getting-started/testdata/project/dist/chart
	rm -rf docs/book/src/cronjob-tutorial/testdata/project/dist/chart
	rm -rf docs/book/src/multiversion-tutorial/testdata/project/dist/chart

	# Generate helm charts from kustomize manifests using v2-alpha plugin
	(cd testdata/project-v4-with-plugins && make build-installer && ../../bin/kubebuilder edit --plugins=helm/v2-alpha)
	(cd docs/book/src/getting-started/testdata/project && make build-installer && ../../../../../../bin/kubebuilder edit --plugins=helm/v2-alpha)
	(cd docs/book/src/cronjob-tutorial/testdata/project && make build-installer && ../../../../../../bin/kubebuilder edit --plugins=helm/v2-alpha)
	(cd docs/book/src/multiversion-tutorial/testdata/project && make build-installer && ../../../../../../bin/kubebuilder edit --plugins=helm/v2-alpha)

.PHONY: check-docs
check-docs: ## Run the script to ensure that the docs are updated
	./hack/docs/check.sh

.PHONY: lint
lint: golangci-lint yamllint check-sample-permissions ## Run golangci-lint linter, yamllint & sample permissions check
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

.PHONY: lint-config
lint-config: golangci-lint ## Verify golangci-lint linter configuration
	$(GOLANGCI_LINT) config verify

# Lint all YAML: testdata files (yamllint-yaml) + Helm-rendered charts (yamllint-helm).
# Repo YAML uses .yamllint; Helm output uses .yamllint-helm.
YAMLLINT_FILES := $(shell find testdata -name '*.yaml' ! -path 'testdata/.helm-rendered.yaml' \( ! -path 'testdata/*/dist/*' -o -path 'testdata/*/dist/chart/Chart.yaml' -o -path 'testdata/*/dist/chart/values.yaml' \) 2>/dev/null)
HELM_CHARTS := $(shell find testdata docs/book -type d -path '*/dist/chart' 2>/dev/null)

.PHONY: yamllint yamllint-yaml yamllint-helm
yamllint: yamllint-yaml yamllint-helm

yamllint-yaml:
	@docker run --rm $$(tty -s && echo "-it" || echo) -v $(PWD):/data -w /data cytopia/yamllint:latest $(YAMLLINT_FILES) -c .yamllint --no-warnings

yamllint-helm:
	@for chart in $(HELM_CHARTS); do \
	  helm template release $$chart --namespace=release-system 2>/dev/null | \
	  docker run --rm -i -v $(PWD):/data -w /data cytopia/yamllint:latest -c .yamllint-helm --no-warnings - || (echo "yamllint-helm: $$chart failed"; exit 1); \
	done

# All Kubebuilder-generated samples (go/v4, kustomize, helm use machinery defaults 0755/0644).
SAMPLE_ROOTS := testdata \
	docs/book/src/getting-started/testdata \
	docs/book/src/cronjob-tutorial/testdata \
	docs/book/src/multiversion-tutorial/testdata

.PHONY: check-sample-permissions
check-sample-permissions: ## Fail if any file/dir under testdata or docs samples has wrong permissions (expect 0644/0755). bin/ excluded.
	@for d in $(SAMPLE_ROOTS); do \
		test -d "$$d" || continue; \
		bad=$$(find "$$d" -path '*/bin' -prune -o \( \( -type f ! -perm 0644 \) -o \( -type d ! -perm 0755 \) \) -print 2>/dev/null); \
		if [ -n "$$bad" ]; then echo "Invalid permissions under $$d (expect 0644/0755):"; echo "$$bad"; exit 1; fi; \
	done

.PHONY: golangci-lint
golangci-lint:
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,${GOLANGCI_LINT_VERSION})

.PHONY: apidiff
apidiff: go-apidiff ## Run the go-apidiff to verify any API differences compared with origin/master
	$(GO_APIDIFF) master --compare-imports --print-compatible --repo-path=.

.PHONY: go-apidiff
go-apidiff:
	$(call go-install-tool,$(GO_APIDIFF),github.com/joelanford/go-apidiff,$(GO_APIDIFF_VERSION))

##@ Tests

.PHONY: test
test: test-unit test-integration test-testdata test-book test-license test-gomod ## Run the unit and integration tests (used in the CI)

.PHONY: test-unit
TEST_PKGS := ./pkg/... ./test/e2e/utils/...
test-unit: ## Run the unit tests
	go test -race $(TEST_PKGS)

.PHONY: test-integration
test-integration: install ## Run the integration tests (requires kubebuilder binary in PATH)
	go test -race -tags=integration -timeout 30m $(TEST_PKGS)

.PHONY: test-coverage
test-coverage: ## Run unit and integration tests with coverage report
	- rm -rf *.out  # Remove all coverage files if exists
	go test -race -failfast -tags=integration -timeout 30m \
		-coverprofile=coverage-all.out \
		-coverpkg="\
./pkg/cli/...,\
./pkg/config/...,\
./pkg/internal/...,\
./pkg/machinery/...,\
./pkg/model/...,\
./pkg/plugin/...,\
./pkg/plugins/golang,\
./pkg/plugins/golang/deploy-image/v1alpha1,\
./pkg/plugins/golang/v4,\
./pkg/plugins/external/...,\
./pkg/plugins/common/kustomize/v2,\
./pkg/plugins/optional/autoupdate/v1alpha,\
./pkg/plugins/optional/grafana/...,\
./pkg/plugins/optional/helm/v2alpha/..." \
		$(TEST_PKGS)

.PHONY: check-testdata
check-testdata: ## Run the script to ensure that the testdata is updated
	./test/testdata/check.sh

.PHONY: test-testdata
test-testdata: ## Run the tests of the testdata directory
	./test/testdata/test.sh

.PHONY: test-e2e-local
test-e2e-local: ## Run the end-to-end tests locally
	## To keep the same kind cluster between test runs, use `SKIP_KIND_CLEANUP=1 make test-e2e-local`
	./test/e2e/local.sh

.PHONY: test-e2e-ci
test-e2e-ci: ## Run the end-to-end tests (used in the CI)`
	./test/e2e/ci.sh

.PHONY: test-book
test-book: ## Run the cronjob tutorial's unit tests to make sure we don't break it
	cd ./docs/book/src/cronjob-tutorial/testdata/project && make test
	cd ./docs/book/src/multiversion-tutorial/testdata/project && make test
	cd ./docs/book/src/getting-started/testdata/project && make test

.PHONY: test-license
test-license:  ## Run the license check
	./test/check-license.sh

.PHONY: test-gomod
test-gomod:  ## Run the Go module compatibility check
	go run ./hack/test/check_go_module.go

.PHONY: test-external-plugin
test-external-plugin: install  ## Run tests for external plugin
	make -C docs/book/src/simple-external-plugin-tutorial/testdata/sampleexternalplugin/v1 install
	make -C docs/book/src/simple-external-plugin-tutorial/testdata/sampleexternalplugin/v1 test-plugin

.PHONY: test-spaces
test-spaces:  ## Run the trailing spaces check
	./test/check_spaces.sh

## TODO: Remove me when go/v4 plugin be removed
## Deprecated
.PHONY: test-legacy
test-legacy:  ## Run the tests to validate legacy path for webhooks
	rm -rf  ./testdata/**legacy**/
	./test/testdata/legacy-webhook-path.sh

.PHONY: install-helm
install-helm: ## Install the latest version of Helm locally
	@curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-4 | bash

.PHONY: helm-lint
helm-lint: install-helm ## Lint the Helm chart in testdata
	helm lint testdata/project-v4-with-plugins/dist/chart

## Tool Binaries
GO_APIDIFF ?= $(LOCALBIN)/go-apidiff
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

## Tool Versions
GO_APIDIFF_VERSION ?= v0.8.3
GOLANGCI_LINT_VERSION ?= v2.8.0

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] && [ "$$(readlink -- "$(1)" 2>/dev/null)" = "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $$(realpath $(1)-$(3)) $(1)
endef
