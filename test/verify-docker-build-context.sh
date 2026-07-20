#!/bin/bash

# Copyright 2026 The Kubernetes Authors.
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

# Verifies the scaffolded .dockerignore: the builder stage must receive every Go source
# needed to build the manager, and none of the files that must not reach the image.
#
# Container tools disagree on whether they descend into an ignored directory to evaluate
# a re-include such as "!**/*.go": BuildKit does, the classic Docker builder and
# Podman/buildah do not. Run this against each of them.
#
# Usage: CONTAINER_TOOL=docker ./test/verify-docker-build-context.sh <project-dir>

set -o errexit
set -o nounset
set -o pipefail

CONTAINER_TOOL="${CONTAINER_TOOL:-docker}"
PROJECT_DIR="${1:?usage: $0 <project-dir>}"

cd "${PROJECT_DIR}"

IMG="dockerignore-context-probe:$$"
trap '${CONTAINER_TOOL} rmi -f "${IMG}" >/dev/null 2>&1 || true' EXIT

# Stop at the builder stage: it still has a shell, and it is the stage that consumes
# the build context. The final image is distroless and only holds the binary.
echo "Building the builder stage with ${CONTAINER_TOOL} ..."
"${CONTAINER_TOOL}" build --target builder -t "${IMG}" .

context="$("${CONTAINER_TOOL}" run --rm "${IMG}" \
  sh -c 'cd /workspace && find . -type f | sed "s|^\./||" | sort')"

echo "Files the builder stage received:"
echo "${context}" | sed 's/^/  /'

fail=0

# Every non-test Go source under the scaffolded source dirs has to be there, or the
# build either breaks now or breaks later when a controller starts importing it.
required="$(find ./cmd ./api ./internal -type f -name '*.go' ! -name '*_test.go' 2>/dev/null \
  | sed 's|^\./||' | sort || true)"
required="$(printf '%s\ngo.mod\ngo.sum\n' "${required}" | grep -v '^$' | sort)"

missing="$(comm -23 <(echo "${required}") <(echo "${context}"))"
if [[ -n "${missing}" ]]; then
  echo "ERROR: these files are missing from the build context:"
  echo "${missing}" | sed 's/^/  /'
  fail=1
fi

# Nothing that belongs to the repo rather than the binary may reach the context.
forbidden="$(echo "${context}" | grep -E \
  '^(config/|dist/|bin/|hack/|grafana/|\.git/|Makefile$|PROJECT$|Dockerfile$|.*\.md$)|_test\.go$' || true)"
if [[ -n "${forbidden}" ]]; then
  echo "ERROR: these files must not be in the build context:"
  echo "${forbidden}" | sed 's/^/  /'
  fail=1
fi

if [[ "${fail}" -ne 0 ]]; then
  exit 1
fi

echo "OK: ${CONTAINER_TOOL} build context contains the manager sources and nothing else."
