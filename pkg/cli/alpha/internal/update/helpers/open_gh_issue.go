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
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

// Curated-diff budgets (fixed; no env vars)
const (
	selectedDiffTotalCap     = 96 << 10 // 96 KiB total across all files
	selectedDiffLinesPerFile = 120      // default +/- lines per file
	selectedDiffLinesGoMod   = 240      // allow more for go.mod
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

// AiPRPrompt is the prompt to `gh models run`.
//
//nolint:lll
const AiPRPrompt = `You are a senior Go/K8s engineer. Produce a concise, reviewer-friendly **Pull Request summary** for a Kubebuilder project upgrade.
Style rules:
- Use **simple, plain English** (like Kubebuilder docs).
- Avoid jargon or long sentences.
- Focus on clarity and readability for new contributors.

Rules (follow strictly):
- Do NOT output angle-bracket placeholders like <OUTPUT_BRANCH>; use the real value from the context.
- Do NOT guess versions. Only mention an exact version (e.g., controller-runtime v0.21.0)
  if that exact version string appears in the provided diffs/context (e.g., go.mod).
- When talking about dependencies:
  - **Only** name modules that changed on **non-indirect** ` + "`require`" + ` lines in **go.mod** (i.e., lines **without** "// indirect").
  - You may also name explicit tool versions found in **Makefile** or **Dockerfile** (e.g., controller-tools, golangci-lint, Go toolchain).
  - **Never** name modules that appear only with "// indirect" or only in **go.sum** or generated files.
  - If you cannot name any direct modules safely, write simply: "dependencies updated" (no module names).
- Output exactly one overview and one reviewed-changes table. No duplicates.
- Valid Markdown only. No ">>>", no meta commentary.
- Start with this exact sentence, substituting real values:
  "This is a Kubebuilder scaffold update from %s to %s on branch %s."
  If a Compare PR URL is provided in the context header, append it **in parentheses** at the end of that sentence as a Markdown link, e.g., " (see [compare PR](URL))".
- A "conflict" means the file currently contains Git merge markers (<<<<<<<, =======, >>>>>>>) and requires manual resolution. If no conflicts are provided in the context, omit the conflicts section entirely.
- Conflicts section: ONLY add if there are conflicts. Do NOT invent conflicts.
- Do NOT invent changes; use only what is in the context.

Required sections (Markdown, EXACT wording/case):

## ( :robot: AI generate ) Scaffold Changes Overview
Start with one short sentence: "This is a Kubebuilder scaffold update from <FROM> to <TO> on branch <OUTPUT_BRANCH>." (with the optional compare link in parentheses at the end).
Then list 4–6 concise bullet highlights (e.g., Go/tooling bumps, controller-runtime/k8s.io deps, security hardening like readOnlyRootFilesystem, error handling improvements).
Then list **only the most important 6–10 bullet points** (never more than 10 items total in this section).
If there are many changes, summarize and cluster them (e.g., "several small Go tooling bumps") instead of listing everything.

### ( :robot: AI generate ) Reviewed Changes
Add a collapsible block:
<details>
<summary>Show a summary per file</summary>

| File | Description |
| ---- | ----------- |
| … | … |

</details>

Build the table using ONLY the "Changed files" lists provided in the context. Do not invent files.
It is OK if some files also appear in the Conflicts section.
If there are many GENERATED files, you may **group them** using a glob with a count (e.g., ` + "`config/crd/bases/*.yaml (12 files)`" + `) instead of listing each one.

**ONLY** if the context includes conflict files; add ANOTHER collapsible block titled **Conflicts Summary**:

<details>
<summary>Conflicts Summary</summary>

| File | Description |
| ---- | ----------- |
| … | … |

</details>

A "conflict" means the file currently contains Git merge markers (<<<<<<<, =======, >>>>>>>) and requires manual resolution. If no conflicts are provided in the context, omit this section.

List each conflicted file with a brief suggestion. For GENERATED files:
- api/**/zz_generated.*.go: "Do not edit by hand; run: make generate"
- config/crd/bases/*.yaml: "Fix types in api/*_types.go; then run: make manifests"
- config/rbac/*.yaml: "Fix markers in controllers/webhooks; then run: make manifests"
- dist/install.yaml: "Fix conflicts; then run: make build-installer"`

// listConflictFiles uses the unified conflict detection from conflict.go
func listConflictFiles() (src []string, gen []string) {
	conflicts := FindConflictFiles()
	return conflicts.SourceFiles, conflicts.GeneratedFiles
}

func bulletList(items []string) string {
	if len(items) == 0 {
		return "<none>"
	}
	return "- " + strings.Join(items, "\n- ")
}

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

// listChangedFiles returns files changed between base..head, split into SOURCE and GENERATED.
func listChangedFiles(base, head string) (src []string, gen []string) {
	cmd := exec.Command("git", "diff", "--name-only", "-M", "--diff-filter=ACMRTD", base+".."+head)
	out, err := cmd.Output()
	if err != nil {
		return nil, nil // best-effort
	}
	for p := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if isGeneratedKB(p) {
			gen = append(gen, p)
		} else {
			src = append(src, p)
		}
	}
	sort.Strings(src)
	sort.Strings(gen)
	return src, gen
}

// BuildFullPrompet builds the AI context and writes it to a temp file.
// It returns the absolute filepath to pass via --input-file/--file.
func BuildFullPrompet(
	fromVersion, toVersion, baseBranch, outBranch, compareURL, releaseURL string,
) string {
	changedSrc, changedGen := listChangedFiles(baseBranch, outBranch)
	conflictSrc, conflictGen := listConflictFiles()

	var ctx strings.Builder

	fmt.Fprintf(&ctx, "Kubebuilder upgrade: %s -> %s\n", fromVersion, toVersion)
	fmt.Fprintf(&ctx, "Compare PR URL: %s\n", compareURL)
	fmt.Fprintf(&ctx, "Release notes: %s\n\n", releaseURL)
	ctx.WriteString("\n")

	// List changed files so the AI can build the Reviewed Changes table.
	if len(changedSrc) > 0 {
		fmt.Fprintf(&ctx, "\nChanged [SOURCE] files:\n%s\n", bulletList(changedSrc))
	}
	if len(changedGen) > 0 {
		fmt.Fprintf(&ctx, "\nChanged [GENERATED] files:\n%s\n", bulletList(changedGen))
	}
	// List conflicts for extra context (will be empty if none)
	if len(conflictSrc) > 0 {
		fmt.Fprintf(&ctx, "\nConflicted [SOURCE] files:\n%s\n", bulletList(conflictSrc))
	}
	if len(conflictGen) > 0 {
		fmt.Fprintf(&ctx, "\nConflicted [GENERATED] files:\n%s\n", bulletList(conflictGen))
	}

	// Concise, curated diffs for important SOURCE files only
	if len(changedSrc) > 0 {
		ctx.WriteString("## Selected diffs\n")
		// Per-file cap is ignored for go.mod (it uses its own higher cap).
		const perFileLineCap = selectedDiffLinesPerFile
		// total cap is fixed inside concatSelectedDiffs (selectedDiffTotalCap).
		ctx.WriteString(concatSelectedDiffs(strings.TrimSpace(baseBranch),
			strings.TrimSpace(outBranch), changedSrc, perFileLineCap, selectedDiffTotalCap))
		ctx.WriteString("\n")
	}

	return ctx.String()
}

// Never include these in curated diffs.
func excludedFromDiff(p string) bool {
	return isGeneratedKB(p) ||
		strings.HasSuffix(p, ".md") ||
		p == "PROJECT" ||
		p == "go.sum" ||
		strings.HasPrefix(p, "grafana/") ||
		strings.HasPrefix(p, "config/crd/bases/") ||
		strings.HasPrefix(p, "hack/") ||
		strings.HasPrefix(p, "bin/") ||
		strings.HasPrefix(p, "vendor/") ||
		strings.HasSuffix(p, ".log")
}

// Only files that matter for KB review context (after exclusions).
func importantFile(p string) bool {
	if excludedFromDiff(p) {
		return false
	}

	// Critical Kubebuilder files
	//nolint:goconst
	if p == "go.mod" || p == "Makefile" || p == "Dockerfile" {
		return true
	}

	// Core source code
	if strings.HasPrefix(p, "cmd/") ||
		strings.HasPrefix(p, "controllers/") ||
		strings.HasPrefix(p, "internal/controller/") ||
		strings.HasPrefix(p, "internal/webhook/") ||
		(strings.HasPrefix(p, "api/") && strings.HasSuffix(p, "_types.go")) {
		return true
	}

	// Test files (important for breaking changes)
	if strings.HasPrefix(p, "test/") && (strings.HasSuffix(p, "_test.go") ||
		strings.HasSuffix(p, ".go")) {
		return true
	}

	// Important config files (not generated)
	if strings.HasPrefix(p, "config/") {
		// Include kustomization files and important config
		if strings.HasSuffix(p, "kustomization.yaml") ||
			strings.HasPrefix(p, "config/default/") ||
			strings.HasPrefix(p, "config/manager/") ||
			strings.HasPrefix(p, "config/webhook/") ||
			strings.HasPrefix(p, "config/certmanager/") ||
			strings.HasPrefix(p, "config/prometheus/") ||
			strings.HasPrefix(p, "config/network-policy/") {
			return true
		}
	}

	return false
}

// Priority: lower number = earlier.
// 0: go.mod (dependencies)
// 1: Makefile (build automation)
// 2: Dockerfile (container images)
// 3: Core code (cmd/, controllers/, api/*_types.go, internal/)
// 4: Critical config (config/default, config/manager)
// 5: Webhook & security config (config/webhook, config/certmanager)
// 6: Other config (config/*)
// 7: Tests
// 9: fallback
func filePriority(p string) int {
	switch {
	case p == "go.mod":
		return 0
	case p == "Makefile":
		return 1
	case p == "Dockerfile":
		return 2
	case strings.HasPrefix(p, "cmd/"),
		strings.HasPrefix(p, "controllers/"),
		strings.HasPrefix(p, "internal/controller/"),
		strings.HasPrefix(p, "internal/webhook/"),
		(strings.HasPrefix(p, "api/") && strings.HasSuffix(p, "_types.go")):
		return 3
	case strings.HasPrefix(p, "config/default/"),
		strings.HasPrefix(p, "config/manager/"),
		p == "config/default/kustomization.yaml",
		p == "config/manager/kustomization.yaml":
		return 4
	case strings.HasPrefix(p, "config/webhook/"),
		strings.HasPrefix(p, "config/certmanager/"),
		strings.HasPrefix(p, "config/prometheus/"),
		strings.HasPrefix(p, "config/network-policy/"):
		return 5
	case strings.HasPrefix(p, "config/"):
		return 6
	case strings.HasPrefix(p, "test/"):
		return 7
	default:
		return 9
	}
}

//nolint:lll
var (
	reFlags       = regexp.MustCompile(`(?i)--(leader-elect|metrics-bind-address|health-probe-bind-address|\bzap|secure-port|bind-address)`)
	reGo          = regexp.MustCompile(`(?i)^(?:\+|\-)\s*(package|import|type|func|const|var|//\+kubebuilder:|//go:(?:build|generate)|return|if\s+err|log\.|fmt\.|errors?\.|client\.|ctrl\.|manager|scheme|requeue|context\.|SetupWithManager|Reconcile|reconcile\.Result)`)
	reYAMLKey     = regexp.MustCompile(`(?i)(apiVersion:|kind:|metadata:|name:|namespace:|image:|command:|args:|env:|resources:|limits:|requests:|ports:|securityContext:|readOnlyRootFilesystem|runAsNonRoot|seccompProfile|allowPrivilegeEscalation|capabilities|livenessProbe|readinessProbe|namePrefix:|commonLabels:|bases:|patches:|replicas:)`)
	reDocker      = regexp.MustCompile(`(?i)^(?:\+|\-)\s*(FROM|ARG|ENV|RUN|ENTRYPOINT|CMD|COPY|ADD|USER|WORKDIR)\b`)
	reMakeLine    = regexp.MustCompile(`(?i)^(?:\+|\-)\s*([A-Z0-9_]+)\s*[:?+]?=\s*|^(?:\+|\-)\s*(manifests|generate|fmt|vet|lint-fix|docker-build|test|install|uninstall|deploy|undeploy|build-installer|controller-gen|kustomize)\b`)
	reKubebuilder = regexp.MustCompile(`(?i)^(?:\+|\-)\s*(\/\/\+kubebuilder:|kubebuilder\s+(init|create|edit)|controller-runtime|sigs\.k8s\.io|k8s\.io\/api|k8s\.io\/apimachinery)`)
)

// keepGoModLine returns true for +/- go.mod lines we want to retain.
// Keep: module/go/toolchain, replace, require lines without "// indirect", and block delimiters.
func keepGoModLine(s string) bool {
	if len(s) == 0 || (s[0] != '+' && s[0] != '-') {
		return false
	}
	t := strings.TrimSpace(s[1:]) // strip +/- then trim
	switch {
	case strings.HasPrefix(t, "module "):
		return true
	case strings.HasPrefix(t, "go "):
		return true
	case strings.HasPrefix(t, "toolchain "):
		return true
	case strings.HasPrefix(t, "replace "):
		return true
	case strings.HasPrefix(t, "require ") && !strings.Contains(t, "// indirect"):
		return true
	case t == "require (" || t == ")": // keep block delimiters for readability
		return true
	default:
		return false
	}
}

// Decide if a +/- line is interesting based on the file path.
func interestingLine(path, line string) bool {
	if len(line) == 0 || (line[0] != '+' && line[0] != '-') {
		return false
	}
	switch {
	case strings.HasSuffix(path, ".go"):
		return reGo.MatchString(line) || reKubebuilder.MatchString(line)
	case strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml"):
		return reYAMLKey.MatchString(line) || reFlags.MatchString(line) || reKubebuilder.MatchString(line)
	case path == "Makefile":
		return reMakeLine.MatchString(line) || reKubebuilder.MatchString(line)
	case path == "Dockerfile":
		return reDocker.MatchString(line)
	case strings.HasSuffix(path, "kustomization.yaml"):
		// Kustomization files are critical for Kubebuilder config
		return true
	default:
		// Unknown text files: keep Kubebuilder-related lines and obvious flag changes
		return reFlags.MatchString(line) || reKubebuilder.MatchString(line)
	}
}

// Curated unified=0 diff: keep hunk headers + filtered +/- lines.
// For go.mod keep only direct requires and key headers (still capped).
func selectedDiff(base, head, path string, maxLines int) string {
	cmd := exec.Command("git", "diff", "--no-color", "-w", "--unified=0", base+".."+head, "--", path)
	out, _ := cmd.Output()
	if len(out) == 0 {
		return ""
	}

	sc := bufio.NewScanner(bytes.NewReader(out))
	lines := 0
	var b strings.Builder

	if path == "go.mod" {
		for sc.Scan() {
			s := sc.Text()
			if strings.HasPrefix(s, "@@") {
				b.WriteString(s + "\n")
				continue
			}
			if keepGoModLine(s) {
				b.WriteString(s + "\n")
				lines++
				if lines >= maxLines {
					break
				}
			}
		}
		return strings.TrimSpace(b.String())
	}

	for sc.Scan() {
		s := sc.Text()
		if strings.HasPrefix(s, "@@") {
			b.WriteString(s + "\n")
			continue
		}
		if len(s) == 0 || (s[0] != '+' && s[0] != '-') {
			continue
		}
		if interestingLine(path, s) {
			b.WriteString(s + "\n")
			lines++
			if lines >= maxLines {
				break
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func concatSelectedDiffs(base, head string, files []string, perFileLineCap, totalByteCap int) string {
	var b strings.Builder

	// Global budget: prefer the passed-in cap if >0, else default.
	remaining := totalByteCap
	if remaining <= 0 {
		remaining = selectedDiffTotalCap
	}

	// Filter and prioritize candidates
	candidates := make([]string, 0, len(files))
	for _, p := range files {
		if importantFile(p) {
			candidates = append(candidates, p)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		pi, pj := filePriority(candidates[i]), filePriority(candidates[j])
		if pi != pj {
			return pi < pj
		}
		return candidates[i] < candidates[j] // stable alphabetical within same priority
	})

	// Emit diffs until the global budget is hit
	for _, p := range candidates {
		// Per-file line budget: use param if >0, else default; ensure go.mod gets at least its larger cap.
		perCap := perFileLineCap
		if perCap <= 0 {
			perCap = selectedDiffLinesPerFile
		}
		if p == "go.mod" && perCap < selectedDiffLinesGoMod {
			perCap = selectedDiffLinesGoMod
		}

		diff := selectedDiff(base, head, p, perCap)
		if diff == "" {
			continue
		}

		section := "----- BEGIN SELECTED DIFF " + p + " -----\n" + diff + "\n----- END SELECTED DIFF " + p + " -----\n\n"
		if len(section) > remaining {
			if remaining <= 0 {
				b.WriteString("\n... [global diff budget reached] ...\n")
				break
			}
			// Trim last section to fit remaining budget
			cut := min(remaining, len(section))
			b.WriteString(section[:cut])
			b.WriteString("\n... [global diff budget reached] ...\n")
			break
		}

		b.WriteString(section)
		remaining -= len(section)
	}

	out := strings.TrimSpace(b.String())
	if out == "" {
		return "<no selected diffs produced>"
	}
	return out
}
