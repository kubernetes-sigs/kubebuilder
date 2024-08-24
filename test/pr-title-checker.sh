#!/bin/bash

# Copyright 2024 The Kubernetes Authors.
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

# Define regex patterns
WIP_REGEX="^\W?WIP\W"
TAG_REGEX="^\[[[:alnum:]\._-]*\]"
PR_TITLE="$1"

# Trim WIP and tags from title
trimmed_title=$(echo "$PR_TITLE" | sed -E "s/$WIP_REGEX//" | sed -E "s/$TAG_REGEX//" | xargs)

# Normalize common emojis in text form to actual emojis
trimmed_title=$(echo "$trimmed_title" | sed -E "s/:warning:/⚠/g")
trimmed_title=$(echo "$trimmed_title" | sed -E "s/:sparkles:/✨/g")
trimmed_title=$(echo "$trimmed_title" | sed -E "s/:bug:/🐛/g")
trimmed_title=$(echo "$trimmed_title" | sed -E "s/:book:/📖/g")
trimmed_title=$(echo "$trimmed_title" | sed -E "s/:rocket:/🚀/g")
trimmed_title=$(echo "$trimmed_title" | sed -E "s/:seedling:/🌱/g")

# Check PR type prefix
if [[ "$trimmed_title" =~ ^⚠ ]] || [[ "$trimmed_title" =~ ^✨ ]] || [[ "$trimmed_title" =~ ^🐛 ]] || [[ "$trimmed_title" =~ ^📖 ]] || [[ "$trimmed_title" =~ ^🚀 ]] || [[ "$trimmed_title" =~ ^🌱 ]]; then
    echo "PR title is valid: $trimmed_title"
    exit 0
else
    echo "Error: No matching PR type indicator found in title."
    echo "You need to have one of these as the prefix of your PR title:"
    echo "- Breaking change: ⚠ (:warning:)"
    echo "- Non-breaking feature: ✨ (:sparkles:)"
    echo "- Patch fix: 🐛 (:bug:)"
    echo "- Docs: 📖 (:book:)"
    echo "- Release: 🚀 (:rocket:)"
    echo "- Infra/Tests/Other: 🌱 (:seedling:)"
    exit 1
fi
