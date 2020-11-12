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

# The help will print out all targets with their descriptions organized bellow their categories. The categories are represented by `##@` and the target descriptions by `##`.
# The awk commands is responsable to read the entire set of makefiles included in this invocation, looking for lines of the file as xyz: ## something, and then pretty-format the target and help. Then, if there's a line with ##@ something, that gets pretty-printed as a category.
# More info over the usage of ANSI control characters for terminal formatting: https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info over awk command: http://linuxcommand.org/lc3_adv_awk.php
.PHONY: help
help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

LD_FLAGS=-ldflags " \
    -X sigs.k8s.io/kubebuilder/v2/cmd.kubeBuilderVersion=$(shell git descripe --tags --dirty --broken) \
    -X sigs.k8s.io/kubebuilder/v2/cmd.goos=$(shell go env GOOS) \
    -X sigs.k8s.io/kubebuilder/v2/cmd.goarch=$(shell go env GOARCH) \
    -X sigs.k8s.io/kubebuilder/v2/cmd.gitCommit=$(shell git rev-parse HEAD) \
    -X sigs.k8s.io/kubebuilder/v2/cmd.buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ') \
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
	./generate_testdata.sh

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
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell dirname $(GOLANGCI_LINT)) v1.29.0 ;\
	}

##@ Tests

.PHONY: go-test
go-test: ## Run the unit test
	go test -race -v ./cmd/... ./pkg/... ./plugins/...

.PHONY: test
test: ## Run the unit tests (used in the CI)
	./test.sh

.PHONY: test-coverage
test-coverage:  ## Run coveralls
	# remove all coverage files if exists
	- rm -rf *.out
	# run the go tests and gen the file coverage-all used to do the integration with coverrals.io
	go test -race -failfast -tags=integration -coverprofile=coverage-all.out ./cmd/... ./pkg/... ./plugins/...

.PHONY: test-e2e-local
test-e2e-local: ## It will run the script to install kind and run e2e tests
	## To keep the same kind cluster between test runs, use `SKIP_KIND_CLEANUP=1 make test-e2e-local`
	./test_e2e_local.sh

.PHONY: check-testdata
check-testdata: ## Run the script to ensure that the testdata is updated
	./check_testdata.sh
