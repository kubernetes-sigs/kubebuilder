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

all: build test

.PHONY: build test

build:
	go build -o bin/kubebuilder ./cmd

install: build
	cp ./bin/kubebuilder $(shell go env GOPATH)/bin/kubebuilder

generate:
	GO111MODULE=on ./generated_golden.sh

test:
	go test -v ./cmd/... ./pkg/...

vendor:
	@GO111MODULE=on go mod tidy
	@GO111MODULE=on go mod download
	@GO111MODULE=on go mod vendor
.PHONY: vendor
