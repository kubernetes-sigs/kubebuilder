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

# Validate accessibility of documentation notes.
# This ensures that:
# 1. All <aside> elements have role="note" for assistive tools
# 2. No heading tags (h1, h2, h3) are used inside <aside> elements
#    (they break the document outline for screen readers)

function check_aside_role {
  echo "Checking for <aside> elements without role=\"note\"..."
  if grep -rn '<aside[^>]*class=' docs/book/src --include="*.md" | grep -v 'role="note"'; then
    echo "ERROR: Found <aside> elements without role=\"note\""
    echo "Please add role=\"note\" to all aside elements for accessibility."
    return 1
  fi
  echo "All <aside> elements have role=\"note\""
  return 0
}

function check_headings_in_aside {
  echo "Checking for heading tags inside <aside> elements..."
  local files_with_issues=()

  # Find all markdown files and check for headings inside aside blocks
  while IFS= read -r file; do
    if awk '/<aside/,/<\/aside>/' "$file" | grep -iq '<h[123]>'; then
      files_with_issues+=("$file")
    fi
  done < <(find docs/book/src -name "*.md" -type f)

  if [ ${#files_with_issues[@]} -gt 0 ]; then
    echo "ERROR: Found heading tags inside <aside> elements in the following files:"
    printf '%s\n' "${files_with_issues[@]}"
    echo ""
    echo "Heading tags (h1, h2, h3) inside <aside> elements break the document outline"
    echo "for assistive tools like screen readers. Please use <p class=\"note-title\"> instead."
    echo ""
    echo "To fix automatically, run:"
    echo "  ./hack/docs/fix_note_accessibility.sh"
    return 1
  fi

  echo "No heading tags found inside <aside> elements"
  return 0
}

function validate_docs_accessibility {
  local failed=0

  check_aside_role || failed=1
  echo ""
  check_headings_in_aside || failed=1

  if [ $failed -eq 1 ]; then
    echo ""
    echo "Documentation accessibility validation FAILED"
    exit 1
  else
    echo ""
    echo "Documentation accessibility validation PASSED"
  fi
}

validate_docs_accessibility
