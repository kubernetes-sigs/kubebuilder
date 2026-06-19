/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package helpers

import (
	"strings"
)

// IssueTitleTmpl is the title template for the GitHub issue.
const IssueTitleTmpl = "[Action Required] Upgrade the Scaffold: %[2]s -> %[1]s"

// IssueBodyTmpl is used when no conflicts are detected during the merge.
//
//nolint:lll
const IssueBodyTmpl = `## Description

Upgrade your project to use the latest scaffold changes introduced in Kubebuilder [%[1]s](https://github.com/kubernetes-sigs/kubebuilder/releases/tag/%[1]s).

See the release notes from [%[3]s](https://github.com/kubernetes-sigs/kubebuilder/releases/tag/%[3]s) to [%[1]s](https://github.com/kubernetes-sigs/kubebuilder/releases/tag/%[1]s) for details about the changes included in this upgrade.

## What to do

A scheduled workflow already attempted this upgrade and created the branch %[4]s to help you in this process.

Create a Pull Request using the URL below to review the changes:
%[2]s

## Next steps

**Verify the changes**
- Build the project
- Run tests
- Confirm everything still works

:book: **More info:** https://kubebuilder.io/reference/commands/alpha_update
`

// IssueBodyTmplWithConflicts is used when conflicts are detected during the merge.
//
//nolint:lll
const IssueBodyTmplWithConflicts = `## Description

Upgrade your project to use the latest scaffold changes introduced in Kubebuilder [%[1]s](https://github.com/kubernetes-sigs/kubebuilder/releases/tag/%[1]s).

See the release notes from [%[3]s](https://github.com/kubernetes-sigs/kubebuilder/releases/tag/%[3]s) to [%[1]s](https://github.com/kubernetes-sigs/kubebuilder/releases/tag/%[1]s) for details about the changes included in this upgrade.

## What to do

A scheduled workflow already attempted this upgrade and created the branch (%[4]s) to help you in this process.

:warning: **Conflicts were detected during the merge.**

Create a Pull Request using the URL below to review the changes and resolve conflicts manually:
%[2]s

## Next steps

### 1. Resolve conflicts
After fixing conflicts, run:
~~~bash
make manifests generate fmt vet lint-fix
~~~

### 2. Optional: work on a new branch
To apply the update in a clean branch, run:
~~~bash
kubebuilder alpha update --output-branch my-fix-branch
~~~

This will create a new branch (my-fix-branch) with the update applied.
Resolve conflicts there, complete the merge locally, and push the branch.

### 3. Verify the changes
- Build the project
- Run tests
- Confirm everything still works

:book: **More info:** https://kubebuilder.io/reference/commands/alpha_update
`

// FirstURL is a helper to grab the first URL-looking token from gh stdout
func FirstURL(s string) string {
	for f := range strings.FieldsSeq(s) {
		if strings.HasPrefix(f, "http://") || strings.HasPrefix(f, "https://") {
			// trim common trailing punctuation
			return strings.TrimRight(f, ").,")
		}
	}
	return ""
}

// IssueNumberFromURL returns the last path segment (…/issues/<n> ⇒ <n>).
func IssueNumberFromURL(u string) string {
	u = strings.TrimSuffix(u, "/")
	if i := strings.LastIndex(u, "/"); i >= 0 && i+1 < len(u) {
		return u[i+1:]
	}
	return ""
}
