#!/usr/bin/env bash

# Copyright 2018 The Kubernetes Authors.
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

set -euo pipefail

source "$(dirname "$0")/common.sh"

header_text "Running e2e tests that do not require a cluster"

build_kb
fetch_tools

pushd . >/dev/null

header_text "Running Alpha Update Command E2E tests"
go test "$(dirname "$0")/e2e/alphaupdate" ${flags:-} -timeout 30m

popd >/dev/null
