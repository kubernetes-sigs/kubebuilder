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

source "$(dirname "$0")/../../test/common.sh"

build_kb

check_directory="$(dirname "$0")/../../docs/book/src/"

# Check docs directory first. If there are any uncommitted change, fail the test.
if [[ $(git status ${check_directory} --porcelain) ]]; then
  echo "Generate Docs test precondition failed!"
  echo "Please commit the change under docs directory before running the Generate Docs test"
  exit 1
fi

make generate-docs

# Check if there are any changes to files under testdata directory.
if [[ $(git status ${check_directory} --porcelain) ]]; then
  git status ${check_directory} --porcelain
  git diff ${check_directory}
  echo "Generate Docs failed!"
  echo "Please, if you have changed the scaffolding make sure you have run: make generate"
  exit 1
else
  echo "Generate Docs passed!"
fi
