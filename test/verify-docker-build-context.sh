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

# Checks that the scaffolded .dockerignore sends exactly the Go sources that build the
# manager to the image build context: every non-test *.go in the project, wherever it
# lives, plus go.mod and go.sum, and nothing else.
#
# Docker (BuildKit) is the supported tool. Podman and Buildah do not evaluate the
# !**/*.go re-include (containers/buildah#6417) and fail this check by design; the
# limitation and its workaround are documented in the FAQ.
#
# Usage: ./test/verify-docker-build-context.sh <project-dir>

set -o errexit
set -o nounset
set -o pipefail

# Sort by bytes on the host so the ordering matches the container's C-locale sort,
# on both Linux and macOS.
export LC_ALL=C

# The !**/*.go re-include needs BuildKit, the default builder since Docker 23.
export DOCKER_BUILDKIT=1

CONTAINER_TOOL="${CONTAINER_TOOL:-docker}"
PROJECT_DIR="${1:?usage: $0 <project-dir>}"

if ! command -v "${CONTAINER_TOOL}" >/dev/null 2>&1; then
  echo "ERROR: '${CONTAINER_TOOL}' is not installed. Install Docker or Podman," >&2
  echo "       or set CONTAINER_TOOL to a container tool that is on PATH." >&2
  exit 1
fi

cd "${PROJECT_DIR}"

# The files the build context must hold, derived from the project itself: every
# non-test Go source, no matter the directory, plus the module files.
# go.sum is only present once modules are resolved, so include it only if it exists.
expected="$( { find . -type f -name '*.go' ! -name '*_test.go' | sed 's|^\./||'
               ls go.mod go.sum 2>/dev/null || true; } | sort )"

IMG="dockerignore-context-probe:$$"
trap '${CONTAINER_TOOL} rmi -f "${IMG}" >/dev/null 2>&1 || true' EXIT

# Copy the build context into a throwaway image so we can list what the .dockerignore
# let through, without running the real (slow) manager build.
"${CONTAINER_TOOL}" build -q -t "${IMG}" -f - . >/dev/null <<'EOF'
FROM busybox
WORKDIR /context
COPY . .
EOF

actual="$("${CONTAINER_TOOL}" run --rm "${IMG}" \
  sh -c 'cd /context && find . -type f | sed "s|^\./||" | sort')"

if ! diff <(echo "${expected}") <(echo "${actual}"); then
  echo "ERROR: ${CONTAINER_TOOL} build context does not match the manager sources."
  echo "       '<' expected, '>' present in the context."
  exit 1
fi

echo "OK: ${CONTAINER_TOOL} build context holds exactly the manager sources."
