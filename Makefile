#
# Makefile with some common workflow for dev, build and test
#

all: build test

.PHONY: build test

build:
	go build -o bin/kubebuilder ./cmd

test:
	go test -v ./cmd/... ./pkg/...

