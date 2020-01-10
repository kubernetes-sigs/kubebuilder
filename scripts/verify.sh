#!/usr/bin/env bash

#  Copyright 2018 The Kubernetes Authors.
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

set -e

source ./common.sh

header_text "running go vet"
go vet ${MOD_OPT} ../...

header_text "running go fmt"
go fmt ${MOD_OPT} ../...

header_text "running golangci-lint"
cd .. # To go to the root of the project
golangci-lint run --disable-all \
    --deadline 5m \
    --enable=misspell \
    --enable=structcheck \
    --enable=deadcode \
    --enable=errcheck \
    --enable=varcheck \
    --enable=unparam \
    --enable=ineffassign \
    --enable=nakedret \
    --enable=interfacer \
    --enable=misspell \
    --enable=dupl \
    --enable=goimports \
    --enable=gocyclo \


##todo(camilamacedo86): The following checks requires fixes in the code
# --enable=golint
# --enable=gocyclo
# --enable=lll
# --enable=goconst
# --enable=gosec
# --enable=maligned

GO111MODULES=on go list -mod=readonly ./...
