#!/usr/bin/env bash

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

# Fix accessibility issues in documentation notes.
# This script:
# 1. Adds role="note" to all <aside> elements
# 2. Replaces <h1>, <h2>, <h3> tags inside <aside> with <p class="note-title">
#
# This maintains the exact same visual rendering while fixing the document
# structure for screen readers and other assistive tools.

set -euo pipefail

docs_dir="docs/book/src"

function fix_aside_role {
  echo "Adding role=\"note\" to <aside> elements..."
  local count=0

  # Find all markdown files with aside elements
  while IFS= read -r file; do
    # Use perl for in-place editing (compatible with both GNU and BSD sed)
    if perl -i -pe 's/<aside\s+class="([^"]+)"(?!\s+role=)>/<aside class="$1" role="note">/g' "$file"; then
      count=$((count + 1))
    fi
  done < <(grep -rl '<aside' "$docs_dir" --include="*.md" || true)

  echo "Updated $count files"
}

function fix_headings_in_aside {
  echo "Replacing heading tags inside <aside> elements..."
  local count=0

  # Find all markdown files
  while IFS= read -r file; do
    local tmpfile
    tmpfile=$(mktemp)
    local inside_aside=0
    local modified=0

    while IFS= read -r line; do
      # Check if we're entering an aside block
      if echo "$line" | grep -q '<aside'; then
        inside_aside=1
        echo "$line" >> "$tmpfile"
        continue
      fi

      # Check if we're exiting an aside block
      if echo "$line" | grep -q '</aside>'; then
        inside_aside=0
        echo "$line" >> "$tmpfile"
        continue
      fi

      # Replace heading tags if inside aside
      if [ $inside_aside -eq 1 ]; then
        # Handle inline headings (opening and closing on same line)
        if echo "$line" | grep -iq '<h[123]>.*</h[123]>'; then
          line=$(echo "$line" | sed -E 's|<[hH]([123])>(.*)</[hH][123]>|<p class="note-title">\2</p>|g')
          modified=1
        # Handle opening tags
        elif echo "$line" | grep -iq '<h[123]>'; then
          line=$(echo "$line" | sed -E 's|<[hH][123]>|<p class="note-title">|g')
          modified=1
        # Handle closing tags
        elif echo "$line" | grep -iq '</h[123]>'; then
          line=$(echo "$line" | sed -E 's|</[hH][123]>|</p>|g')
          modified=1
        fi
      fi

      echo "$line" >> "$tmpfile"
    done < "$file"

    if [ $modified -eq 1 ]; then
      mv "$tmpfile" "$file"
      count=$((count + 1))
    else
      rm "$tmpfile"
    fi
  done < <(find "$docs_dir" -name "*.md" -type f)

  echo "Fixed $count files"
}

echo "Fixing documentation accessibility issues..."
echo ""

fix_aside_role
echo ""
fix_headings_in_aside
echo ""
echo "Done! Please run 'make test-docs-accessibility' to verify."
