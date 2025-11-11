| Authors | Creation Date | Status | Extra |
|---|---:|---|---|
| vitorfloriano (drafted with Copilot) | 2025-11-11 | Draft | Implements: docs generation, patch overlays, snapshot testing |

# Golden Snapshot Patches: Improving Docs Maintainability

This proposal introduces "Golden Snapshot Patches" — a lightweight, auditable workflow for generating, customizing, including, and testing tutorial code snippets in the Kubebuilder book.

## Example

This section demonstrates how a real contributor would use Golden Snapshot Patches in practice. The goal is to show commands, file layout, and the expected test/CI feedback so reviewers can evaluate the end-user experience.

Scenario: You (contributor/maintainer/docs writer) add a new code snippet to the getting-started tutorial that should come from the runnable sample project, but you also need a small customization in the sample for the example to read well.

1. Authoring the Markdown tutorial page

- File: docs/book/src/getting-started/intro.md

````markdown
# Getting started

This section shows how to define the CronJob spec.

```go
{{#include testdata/project/api/v1/cronjob_types.go:42:58}}
```

Explain the CronJob fields and how they are used.
````
Notes:
- The _include_ uses mdBook's native line-range include: path: start:end.
- The page is pure Markdown; no Go string literals or inlined HTML.

2. Small tutorial-specific sample customization (as a patch)

- Directory: docs/book/src/getting-started/testdata/project.patches/
- File: 01-cronjob_types.patch

Example unified-diff patch (kept small and focused):

```diff
--- api/v1/cronjob_types.go
+++ api/v1/cronjob_types.go
@@ lines @@ type CronJobSpec struct {
+	// DefaultSuspend indicates the CronJob's default suspend.
+	// This field is added for the tutorial to show defaulting behavior.
+	// +optional
+	DefaultSuspend *bool `json:"defaultSuspend,omitempty"`
 	Schedule string `json:"schedule"`
 	// ... other fields
 }
```

Rationale:
- Patches are tiny, reviewable diffs. They document intent in the PR and are visible in GitHub's unified diff viewers.

3. Apply patches after generation (local dev)

Regeneration + apply step (Makefile target or ad-hoc):

```bash
# regenerate runnable sample, then apply patches
make samples-generate    # runs generate_samples.go -> populates testdata/project/
./hack/docs/scripts/apply_patches.sh docs/book/src/getting-started/testdata/project \
    docs/book/src/getting-started/testdata/project.patches
```

apply_patches.sh (conceptual):

```bash
#!/usr/bin/env bash
set -euo pipefail
project_dir="$1"
patch_dir="$2"
cd "$project_dir"
for p in "${patch_dir}"/*.patch; do
  echo "Applying $p"
  if git apply --index "$p"; then
    echo "applied $p with git apply"
  else
    patch -p1 < "$p" || { echo "Failed to apply $p"; exit 1; }
  fi
done
```

4. Render mdBook to produce actual markdown output (what users will see)

```toml
# This can be achieve by simply adding this option to book.toml
[output.markdown]
```

5. Snapshot test (local)

Run the tests that compare the rendered output for a page against the golden:

```bash
# Normal test (fails if rendered output != golden)
go test ./tests/docs -run TestDocumentationSnapshots -v

# If the diffs are intentional, regenerate goldens:
UPDATE_SNAPSHOTS=1 go test ./tests/docs -run TestDocumentationSnapshots -v
# or:
make update-golden-files
```

Expected test failure output (example):

```
--- FAIL: docs/book/src/getting-started/intro.md
    snapshot_test.go:67: Snapshot mismatch for getting-started/intro.md:
    --- expected
    +++ actual
    @@ -3,7 +3,8 @@
     ```go
    -type CronJobSpec struct {
    -    Schedule string `json:"schedule"`
    -}
    +type CronJobSpec struct {
    +    DefaultSuspend *bool `json:"defaultSuspend,omitempty"`
    +    Schedule string `json:"schedule"`
    +}
     ```
```

This diff tells you:
- The snippet extracted by `{{#include}}` changed because the patched sample added DefaultSuspend.
- If the change is intentional, run `UPDATE_SNAPSHOTS=1` to accept & update the golden; otherwise, fix the patch or the `{{#include}}` range.

6. Updating goldens (intentional change)

If the change is intended:

```bash
# regenerate golden artifacts from rendered output (Makefile wraps this)
make update-golden-files
git add docs/book/src/getting-started/intro.md.golden
git commit -m "docs(getting-started): update golden after intentional sample change"
git push
```

7. CI behavior

Docs CI job runs:

1. `make samples-generate` (regenerate samples)
2. `apply_patches` (apply overlays)
3. `go test ./tests/docs` (snapshot assertions)

- If snapshot test fails, CI job logs the diff and fails the PR job.
- Reviewers see both the source Markdown and the changed golden in the PR when the contributor updates goldens; they can inspect the patch diffs to review sample customizations.

8. UX considerations captured in this example

- Contributors edit Markdown only; no escaping or string constants required.
- Contributors keep the sample runnable; small, reviewable patches express tutorial needs.
- Snapshot test diffs are the contract: they make rendered-output changes visible and auditable.
- The update flow (`UPDATE_SNAPSHOTS=1`) is explicit and intentionally gated.

---

This Example demonstrates the full workflow in which a contributor writes Markdown, generation produces samples, patches mutate the sample, mdBook renders the page, tests detect drift, and updating goldens is explicit. The flow is incremental and reviewable, minimizing surprises for maintainers and contributors.

## Open Questions

The proposed workflow raise a few questions on how to implement certains aspects, like the following:

1. How should we patch the samples: `git apply` vs `patch`?
2. How granular should the golden be: entire book vs per-page vs per-section vs per-codeblock?
3. Should we enforce golden updates in the contributor workflow?
4. Should generated samples be ephemeral by default?

## Summary

Golden Snapshot Patches is a lightweight, practical redesign of the Kubebuilder documentation generation workflow that shifts the source of truth for tutorial prose to Markdown, keeps sample projects fully generated and runnable, and makes tutorial-specific changes explicit and reviewable using standard unified-diff patches.

Rather than embedding prose inside generated Go files and relying on a custom literate extractor, this approach applies small, focused `.patch` overlays to freshly-generated sample projects and uses mdBook's native `{{#include filename:lines}}` syntax to pull snippets into Markdown. The rendered pages are validated by simple snapshot tests that compare mdBook Markdown output to checked-in golden files; mismatches produce readable diffs that surface unintended drift or deliberate changes for reviewers.

This change directly addresses recurring problems observed in prior review cycles—fragile string constants, invisible whitespace/formatting regressions, brittle custom parsing, and tautological tests—by

1. keeping prose where writers expect it
2. keeping runnable code separate and testable
3. making customizations human-readable (patch diffs)
4. validating the actual user-facing output via per-page goldens.

The design is deliberately incremental and low-infrastructure: patches are standard `git diff`/`patch` files, snapshot tests are simple file comparisons, and Make targets/scripts orchestrate generation, patch application, rendering, and test execution.

The end result is a more maintainable, auditable workflow that yields clearer PRs, faster reviews, and stronger guarantees that documentation examples continue to match runnable code.

## Motivation

The current documentation generation workflow mixes concerns in ways that make writing, reviewing, and maintaining tutorials difficult:

- Tutorial prose is frequently embedded inside generated source (Go string constants or block comments) so writers must author documentation inside code files. This leads to invisible whitespace/formatting bugs, escaping pain, and poor editing ergonomics for writers.
- To render tutorials, the project relies on custom extraction logic (for example, literate.go) and many Insert/Replace operations. That logic is complex, brittle, and hard to reason about. It also causes a lot of overhead and make it very hard to onboard new contributors/maintainers.
- Many customization steps for tutorial samples are implemented as scattered code edits in generator helpers, which makes intent hard to see in PR diffs and increases the cognitive burden on reviewers and maintainers.

This proposal addresses these concrete, recurring problems by moving to a simple, auditable workflow that separates responsibilities and validates the actual user-visible output.

### Why this matters to users

- Docs authors can write and edit prose in plain Markdown, which improves productivity and reduces errors.
- Engineers keep sample code runnable and testable; included snippets are taken from working code, increasing trust for readers.
- Reviewers get clear, small diffs (patch overlays for sample customizations and golden diffs for rendered output) so they can verify both intent and result easily.
- CI detects unintended drift between generated samples and documentation early via snapshot tests, preventing silent breakage.

### Goals

- Make tutorial prose the canonical source (Markdown), not embedded code strings.
- Preserve automatic sample generation while making tutorial-specific edits explicit and reviewable.
- Validate the *rendered* documentation (what users see) with per-page snapshot tests and human-readable diffs.
- Use standard, well-understood formats and tools (unified-diff patches, mdBook includes, golden files) to minimize new infra and lower maintenance burden.
- Provide a clear, documented developer workflow (regenerate → apply patches → render → test → update goldens if intentional).

### Non-Goals

- Introducing a heavy-weight preprocessor to inject markers into generated samples (this would increase coupling to the generator).
- Replacing mdBook or changing rendering semantics; the design uses mdBook’s native include capability.
- Forcing all conceptual examples into runnable samples; docs-only fragments remain an acceptable and documented option.

By focusing on these motivations and goals, Golden Snapshot Patches aims to reduce the brittleness and review overhead that currently plague documentation maintenance while preserving runnable examples and improving reviewer confidence through explicit, testable artifacts.

## Proposal

This section specifies the concrete implementation plan and the exact developer workflow required to implement Golden Snapshot Patches. It describes how generated samples are produced, how tutorial customizations are expressed and applied, how mdBook includes are rendered, how snapshot tests run, and how CI enforces correctness.

Design principles
- Keep prose in Markdown and use mdBook's native include syntax for snippets.
- Keep generated samples runnable and produced by the existing generation tooling.
- Express tutorial-specific changes as small, human-readable `.patch` overlay files applied after generation.
- Validate the actual rendered page with per-page golden files; provide an explicit update mode for intentional changes.
- Use standard, well-understood tools (git/patch, mdBook, unified-diff) to minimize new infrastructure.

Core components
1. Generated sample project
   - Location: docs/book/src/<tutorial>/testdata/project/
   - Produced by the existing generator (generate_samples.go / hack/docs workflow).
   - Should be a normal filesystem tree; optionally initialize a git repo locally to allow `git apply` usage.

2. Patch overlays
   - Location: docs/book/src/<tutorial>/testdata/project.patches/
   - Format: unified-diff `.patch` files (produced by `git diff` or manually authored).
   - Naming convention: `<nn>-short-description.patch` (numeric prefix to define deterministic ordering).
   - Each patch must be small, focused, and accompanied by a short README explaining intent and any ordering constraints.
   - Apply method: prefer `git apply --index` (keeps index consistent) and fall back to `patch -p1` if needed.
   - Failure behavior: patch application failure is a hard error in generation — present the failed patch, the hunk context, and the generated tree path for debugging.

3. Apply patches step
   - Implement a script `hack/docs/scripts/apply_patches.sh` (small, portable shell) or a tiny Go helper.
   - Called by the generation flow immediately after samples are scaffolded (modify generate_samples.go or the wrapper script to invoke apply_patches).
   - Script responsibilities:
     - Ensure the project root is correct.
     - Iterate patches in lexical order (numeric prefix enforces intended order).
     - Try `git apply --index` and check for success; if it fails, run `patch -p1` as a fallback.
     - If a patch fails, print the failing patch, failing hunk, and example manual fix instructions and exit non-zero.

4. Including snippets in Markdown
   - Writers use mdBook `{{#include path/to/file.go:start:end}}` for line-range inclusion.
   - Prefer line ranges for simple extraction; for fragile sections, use docs-only fragments (see below).
   - Keep prose and include directives in Markdown pages only.

5. Docs-only fragments
   - Location: docs/book/src/<tutorial>/fragments/
   - Use when a snippet demonstrates an alternate approach or concept that should NOT be applied to the runnable sample.
   - Include these via mdBook `{{#include fragments/foo.go}}`.
   - Mark these clearly in prose as "docs-only".

6. Rendering and golden extraction
   - Build mdBook in markdown mode to get deterministic textual output: `mdbook build docs/book --output-format markdown`.
   - Extract the rendered page corresponding to each source `.md` file (the mdBook output directory contains per-page files).
   - Store checked-in golden snapshots next to source pages: `<page>.md.golden`.
   - For readability in PRs, golden files should be Markdown.

7. Snapshot tests
   - Implement `tests/docs/snapshot_test.go` (Go test) that:
     - Discovers `.md` files containing `{{#include`.
     - Ensures generation and patch application were already run (or runs `make samples-generate` in test setup if desired).
     - Builds mdBook (or reads prebuilt mdBook output) and extracts the per-page output.
     - Compares actual output to corresponding `.md.golden` using `cmp.Diff` to produce a readable unified diff.
     - Supports update mode: when `UPDATE_SNAPSHOTS=1` environment variable is set, the test writes actual output to `.md.golden` and succeeds.
   - Test failure messages must be actionable:
     - Show file path, diff, and exact command to update goldens (e.g., `UPDATE_SNAPSHOTS=1 make update-golden-files`).

8. Make targets & scripts
   - `make samples-generate` - run generate_samples.go to scaffold samples.
   - `make docs-generate` - build mdBook (`docs-generate` should depend on `samples-generate` and `apply_patches`).
   - `make update-golden-files` - build mdBook and overwrite `.md.golden` for pages discovered by tests.
   - `make test-docs` - run snapshot tests (fast path when mdBook is already built).
   - `hack/docs/scripts/check-golden-sync.sh` - quick check to ensure `.md` and `.md.golden` are touched together (used by pre-commit or CI gating).

9. CI integration
   - Create a docs-specific CI job that runs:
     1. `make samples-generate`
     2. `apply_patches`
     3. `make docs-generate`
     4. `go test ./tests/docs` (snapshot tests)
   - The job must fail with the diffs emitted if snapshots diverge.
   - The PR author must either update `.md.golden` or adjust generation/patches/includes to fix unintended changes.
   - Optionally gate PR merges with `check-golden-sync` to ensure that changed `.md` files always update their goldens.

10. Migration approach
    - Pilot one tutorial (cronjob or getting-started).
    - Convert existing `UpdateTutorial()` Insert/Replace calls into representative `.patch` files in `testdata/project.patches/`.
    - Adjust `generate_samples.go` for that tutorial to call `apply_patches`.
    - Add `.md.golden` for pages that use include directives.
    - Add snapshot tests and ensure CI passes.
    - Iterate and migrate remaining tutorials incrementally; remove legacy extraction logic once all migrated.

Error handling & UX
- When patch apply fails: fail generation with error messages including patch filename, hunk context, and path to generated project.
- When snapshot test fails: fail test and print diff + suggestion to update goldens.
- Provide a developer `make` shorthand to run full flow locally and a shorter path for iterative work (e.g., `make samples-generate && mdbook serve`).

Security & privacy
- Patches and goldens are developer-authored content; treat them like any other source file.
- CI should run in an isolated environment with cached toolchains to avoid network flakiness; generation should be deterministic when possible.

Extensibility
- The patch application helper can later accept a `--dry-run` mode to preview patches without mutating the generated project.
- For teams that prefer a git-backed generated tree, generator can initialize a temporary git repo to allow `git apply` and easier diagnostics.

### Repository layout (proposed vs current)

Below are two trees:

1. The proposed, opinionated layout for the Golden Snapshot Patches workflow with short annotations about what belongs in each directory.
2. A focused slice of the current repository layout showing where the existing docs-generation files and related artifacts live today (only paths relevant to this proposal are included).

Use the proposed layout as a guide when migrating tutorials (pilot → full migration). The current layout shows the existing locations you will change or reference while migrating.

---

## Proposed architecture (annotated)

```
.
├── book.toml
├── src                               <-- canonical tutorial Markdown
│   ├── cronjob-tutorial
│   │   ├── intro.md
│   │   └── ...
│   ├── getting-started
│   └── multiversion-tutorial
├── samples                           <-- generated runnable sample
│   ├── cronjob-tutorial-sample
│   ├── getting-started-sample
│   └── multiversion-tutorial-sample
├── patches                          <-- tutorial-specific overlays
│   ├── cronjob-tutorial-patches
│   ├── getting-started-patches
│   └── multiversion-tutorial-patches
├── testdata                          <-- .md.golden snapshots
│   ├── cronjob-tutorial-golden
│   ├── getting-started-golden
│   └── multiversion-tutorial-golden
├── hack                              <-- orchestration scripts
│   ├── apply_patches.sh
│   ├── update-golden-files.sh
│   ├── check-golden-sync.sh
│   └── ...
├── tests
│   └── snapshot_test.go         <-- Go test comparing mdBook to golden
├── themes
│   └── css
└── utils
    ├── scripts
    └── preprocessors
```

---

## Current repository (focused slice relevant to this proposal)

This is a simplified view showing where the current docs generation and related files live today. Only files and directories relevant to the docs/sample generation, literate extraction, and the docs book are shown.

```
.
├── docs
│   └── book
│       ├── book.toml
│       ├── src                      <-- current tutorial markdown
│       │   ├── cronjob-tutorial
│       │   ├── getting-started
│       │   └── multiversion-tutorial
│       ├── theme
│       └── utils
│           └── preprocessors
│               └── literate.go     <-- current code extractor
├── hack
│   └── docs                <-- orchestration scripts and generators
│       ├── check.sh
│       ├── generate.sh
│       ├── generate_samples.go
│       ├── internal
│       │   ├── cronjob-tutorial
│       │   ├── getting-started
│       │   └── multiversion-tutorial
│       └── utils           <-- helper functions used by generators
├── pkg
│   └── plugin
│       └── util            <-- pluginutil used by generators
```

Annotations / why these matter:
- `hack/docs/internal/*` is where the current approach implements many `InsertCode` / `ReplaceInFile` operations as Go string constants. Moving these customizations into `patches/` makes intent and diffs visible instead of hidden in Go constants.
- `generate_samples.go` currently runs the whole generation → UpdateTutorial → CodeGen flow. The proposed change updates this flow to call `apply_patches.sh` after generation and before mdBook extraction.
- `docs/book/utils/litgo/literate.go` is the existing custom parser that extracts comments from Go source and interleaves them with code. The proposal aims to stop using it (for migrated tutorials) in favor of direct mdBook includes and patches.
- `docs/book/src/*` is already the home for tutorial Markdown; the proposal keeps prose here and uses mdBook `{{#include}}` to pull snippets from patched samples.
- `check.sh` and `generate.sh` are existing CI helpers — they will map to the new `make` targets (or be adjusted) to integrate patch application and golden update steps.

---

### User Stories

- As a docs writer, I can write tutorial prose in plain Markdown and use mdBook include directives, so that I can author docs without escaping embedded code or fighting Go string constants.

- As a developer maintaining samples, I can keep generated samples runnable and express tutorial-specific edits as explicit `.patch` files, so that the sample remains testable and the intent of each customization is visible in PR diffs.

- As a reviewer, I can inspect a small set of patch files and the diff between actual and golden render output, so that I can quickly verify both the sample customizations and the user-facing documentation change.

- As a CI engineer, I can run a deterministic pipeline that regenerates samples, applies overlays, renders mdBook, and runs snapshot tests, so that unintended documentation drift is caught automatically.

- As a contributor making an intentional doc or sample change, I can run `UPDATE_SNAPSHOTS=1 make update-golden-files` to update goldens, so that accepting deliberate changes is explicit and auditable.

- As a maintainer, I can migrate tutorials incrementally (pilot → broader rollout), so that the effort is low-risk and reversible while removing brittle extraction code over time.

- As a reader of the documentation, I can trust that code snippets are taken from runnable samples (or are clearly labeled docs-only fragments), so that the examples are more likely to work when followed.

- As a plugin/extension author, I can provide a patch or fragment to illustrate plugin-specific behavior, so that the book can show variations without coupling them into the core runnable sample.

This proposal yields a reproducible, reviewable, and testable docs generation flow that prevents silent regressions, surfaces intended changes clearly, and keeps the editorial and engineering responsibilities neatly separated.

### Implementation Details/Notes/Constraints [optional]

<!-- What are the caveats to the implementation? What are some important details that
didn't come across above. Go in to as much detail as necessary here. This might
be a good place to talk about core concepts and how they relate. -->

### Risks and Mitigations

<!-- What are the risks of this proposal and how do we mitigate. Think broadly. For
example, consider both security and how this will impact the larger Operator Framework
ecosystem.

How will security be reviewed and by whom? How will UX be reviewed and by whom?

Consider including folks that also work outside your immediate sub-project. -->

### Proof of Concept [optional]

<!-- A demo showcasing a prototype of your design can be extremely useful to the
community when reviewing your proposal. There are many services that enable
you to record and share demos. Most OLM features can be showcased from the
command line, making [https://asciinema.org](https://asciinema.org) an
excellent option to [record](https://asciinema.org/docs/usage) and
[embed](https://asciinema.org/docs/embedding) your demo.

Be sure to include:
- An embedded recording of the prototype in action.
- A link to the repository hosting the changes that the prototype introduces. -->

## Drawbacks

<!-- The idea is to find the best form of an argument why this enhancement should _not_ be implemented. -->

## Alternatives

<!-- Similar to the `Drawbacks` section the `Alternatives` section is used to
highlight and record other possible approaches to delivering the value proposed
by an enhancement. -->
