#!/usr/bin/env bash
# Copyright 2019 The Kubernetes Authors.
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

set -o errexit
set -o nounset
set -o pipefail

source common.sh

fetch_tools
build_kb

setup_envs

check_directory=testdata

# Check testdata directory first. If there are any uncommitted change, fail the test.
if [[ `git status ${check_directory} --porcelain` ]]; then
  header_text "Golden test precondition failed!"
  header_text "Please commit the change under testdata directory before running the golden test"
  exit 1
fi

./generate_golden.sh

# Check if there are any changes to files under testdata directory.
if [[ `git status ${check_directory} --porcelain` ]]; then
  header_text "git status ${check_directory} --porcelain"
  git status ${check_directory} --porcelain
  header_text "git diff ${check_directory}"
  git diff ${check_directory}
  header_text "Golden test failed!"
  header_text "Please make sure you have run ./generate_golden.sh if you have changed the scaffolding"
  exit 1
else
  header_text "Golden test passed!"
fi
