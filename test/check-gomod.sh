#!/bin/bash

# Copyright 2025 The Kubernetes Authors.
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

echo "Checking for Go module compatibility (invalid filename characters)..."

# Go modules don't allow certain characters in file paths when creating archives
# Reference: https://go.dev/ref/mod#zip-files
# Invalid characters include: *, ?, [, ], \, and others that are shell wildcards or special characters

# Get all tracked files in git (this will include all files that would be part of the go module)
allfiles=$(git ls-files)
invalidFiles=""

for file in $allfiles; do
  # Check if the filename (not the full path) contains invalid characters
  filename=$(basename "$file")
  
  # Check for wildcard characters that are invalid in Go module archives
  # Note: We check for *, ?, [, ], and \ (backslash)
  if echo "$filename" | grep -qE '[*?[]|\\'; then
    invalidFiles="${invalidFiles}\n  ${file} (contains invalid character in filename: ${filename})"
  fi
done

if [ -n "${invalidFiles}" ]; then
  echo -e "Go module compatibility check failed. The following files contain invalid characters:"
  echo -e "${invalidFiles}"
  echo ""
  echo "Go modules cannot include files with wildcard characters (*, ?, [, ], \\) in their names."
  echo "This will cause 'go get' and 'go mod download' to fail."
  echo "Please rename these files to use only alphanumeric characters, hyphens, underscores, and dots."
  exit 255
fi

echo "Go module compatibility check passed."

