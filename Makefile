#!/usr/bin/env bash

#  Copyright 2019 The Kubernetes Authors.
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

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
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

LD_FLAGS=-ldflags " \
    -X main.kubeBuilderVersion=$(shell git describe --tags --dirty --broken) \
    -X main.goos=$(shell go env GOOS) \
    -X main.goarch=$(shell go env GOARCH) \
    -X main.gitCommit=$(shell git rev-parse HEAD) \
    -X main.buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ') \
    "
.PHONY: build
build: ## Build the project locally
	go build $(LD_FLAGS) -o bin/kubebuilder ./cmd

.PHONY: install
install: build ## Build and install the binary with the current source code. Use it to test your changes locally.
	cp ./bin/kubebuilder $(GOBIN)/kubebuilder

##@ Development

.PHONY: generate
generate: generate-testdata ## Update/generate all mock data. You should run this commands to update the mock data after your changes.
	go mod tidy

.PHONY: generate-testdata
generate-testdata: ## Update/generate the testdata in $GOPATH/src/sigs.k8s.io/kubebuilder
	./test/testdata/generate.sh

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
golangci-lint:
	@[ -f $(GOLANGCI_LINT) ] || { \
	set -e ;\
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell dirname $(GOLANGCI_LINT)) v1.37.1 ;\
	}

.PHONY: apidiff
apidiff: go-apidiff ## Run the go-apidiff to verify any API differences compared with origin/master
	$(GO_APIDIFF) master --compare-imports --print-compatible --repo-path=.

GO_APIDIFF = $(shell pwd)/bin/go-apidiff
go-apidiff:
	@[ -f $(GO_APIDIFF) ] || { \
	cd tools && go build -tags=tools -o $(GO_APIDIFF) github.com/joelanford/go-apidiff ;\
	}

##@ Tests

.PHONY: test
test: test-unit test-integration test-testdata test-book ## Run the unit and integration tests (used in the CI)

.PHONY: test-unit
test-unit: ## Run the unit tests
	go test -race -v ./pkg/...

.PHONY: test-coverage
test-coverage: ## Run unit tests creating the output to report coverage
	- rm -rf *.out  # Remove all coverage files if exists
	go test -race -failfast -tags=integration -coverprofile=coverage-all.out -coverpkg="./pkg/cli/...,./pkg/config/...,./pkg/internal/...,./pkg/machinery/...,./pkg/model/...,./pkg/plugin/...,./pkg/plugins/golang" ./pkg/...

.PHONY: test-integration
test-integration: ## Run the integration tests
	./test/integration.sh

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
