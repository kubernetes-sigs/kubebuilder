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
CONTROLLER_GEN_BIN_PATH := $(shell which controller-gen)

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

.PHONY: build
build: ## Build the project locally
	go build -o bin/kubebuilder ./cmd

.PHONY: install
install: ## Build and install the binary with the current source code. Use it to test your changes locally.
	make build
	cp ./bin/kubebuilder $(shell go env GOPATH)/bin/kubebuilder

##@ Development

.PHONY: generate
generate: ## Update/generate all mock data. You should run this commands to update the mock data after your changes.
    # If exists, rremove the controller-gen installed locally. It ensures that the version which will be used
    # is the version defined in the Makefile scaffolded.
	- rm -rf $(CONTROLLER_GEN_BIN_PATH)
	make generate-testdata
	go mod tidy

.PHONY: generate-testdata
generate-testdata: ## Update/generate the testdata in $GOPATH/src/sigs.k8s.io/kubebuilder
	GO111MODULE=on ./generate_testdata.sh

.PHONY: generate-vendor
generate-vendor: ## (Deprecated) Update/generate the vendor by using the path $GOPATH/src/sigs.k8s.io/kubebuilder-test
	GO111MODULE=off ./generate_vendor.sh

.PHONY: lint
lint: ## Run code lint checks
	./scripts/verify.sh

##@ Tests

.PHONY: go-test
go-test: ## Run the go tests ($ go test -v ./cmd/... ./pkg/...)
	go test -v ./cmd/... ./pkg/...

.PHONY: test
test: ## Run the unit tests (used in the CI)
	./test.sh

.PHONY: test-coverage
test-coverage:  ## Run coveralls
	# remove all coverage files if exists
	- rm -rf *.out
	# run the go tests and gen the file coverage-all used to do the integration with coverrals.io
	go test -failfast -tags=integration -coverprofile=coverage-all.out -covermode=count ./pkg/... ./cmd/...

.PHONY: check-testdata
check-testdata: ## Run the script to ensure that the testdata is updated
	./check_testdata.sh
