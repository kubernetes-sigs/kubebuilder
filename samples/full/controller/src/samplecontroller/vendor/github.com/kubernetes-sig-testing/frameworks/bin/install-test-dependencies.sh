#!/usr/bin/env bash

set -e
set -u

go get -u github.com/golang/lint/golint
go get -u golang.org/x/tools/cmd/goimports
go get -u github.com/onsi/ginkgo/ginkgo
