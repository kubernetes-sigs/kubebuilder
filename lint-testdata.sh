#!/usr/bin/env bash

# Copyright 2023 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GOLANGCI_LINT="$(pwd)/bin/golangci-lint"

args="$1"

runOpts="./..."
if [[ "$args" == "fix" ]]; then
  runOpts="./... --fix"
fi

# only check go/v4 testdata
FILES=`ls testdata | grep "v4" | xargs`

for project in $FILES; do
  ( cd "testdata/$project/" && pwd && $GOLANGCI_LINT run $runOpts)
done

