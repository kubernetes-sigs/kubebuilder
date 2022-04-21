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

# This script runs goreleaser using the build/.goreleaser.yml config.
# While it can be run locally, it is intended to be run by cloudbuild
# in the goreleaser/goreleaser image.

set -e

# If running in Github actions: this should be set to "github.base_ref".
: ${1?"the first argument must be set to a commit-ish reference"}

# Patterns to ignore.
declare -a DOC_PATTERNS
DOC_PATTERNS=(
  "(\.md)"
  "(\.MD)"
  "(\.png)"
  "(\.pdf)"
  "(netlify\.toml)"
  "(OWNERS)"
  "(OWNERS_ALIASES)"
  "(LICENSE)"
  "(docs/)"
)

if ! git diff --name-only $1 | grep -qvE "$(IFS="|"; echo "${DOC_PATTERNS[*]}")"; then
  echo "true"
  exit 0
fi
