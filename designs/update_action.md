| Authors         | Creation Date | Status      | Extra |
|-----------------|---------------|-------------|-------|
| @camilamacedo86 | 2024-11-07    | Implementable | - |
| @vitorfloriano  |               | Implementable | - |

# Proposal: Automating Operator Maintenance: Driving Better Results with Less Overhead

## Introduction

Code-generation tools like **Kubebuilder** and **Operator-SDK** have revolutionized cloud-native application development by providing scalable, community-driven frameworks. These tools simplify complexity, accelerate development, and enable developers to create tailored solutions while avoiding common pitfalls, establishing a strong foundation for innovation.

However, as these tools evolve to keep up with ecosystem changes and new features, projects risk becoming outdated. Manual updates are time-consuming, error-prone, and create challenges in maintaining security, adopting advancements, and staying aligned with modern standards.

This project proposes an **automated solution for Kubebuilder**, with potential applications for similar tools or those built on its foundation. By streamlining maintenance, projects remain modern, secure, and adaptable, fostering growth and innovation across the ecosystem. The automation lets developers focus on what matters most: **building great solutions**.


## Problem Statement

Kubebuilder is widely used for developing Kubernetes operators, providing a standardized scaffold. However, as the ecosystem evolves, keeping projects up-to-date presents challenges due to:

- **Manual re-scaffolding processes**: These are time-intensive and error-prone.
- **Increased risk of outdated configurations**: Leads to security vulnerabilities and incompatibility with modern practices.

## Proposed Solution

This proposal introduces a **workflow-based tool** (such as a GitHub Action) that automates updates for Kubebuilder projects. Whenever a new version of Kubebuilder is released, the tool initiates a workflow that:

1. **Detects the new release**.
2. **Generates an updated scaffold**.
3. **Performs a three-way merge to retain customizations**.
4. **Creates a pull request (PR) summarizing the updates** for review and merging.

## Example Usage

### GitHub Actions Workflow:

1. A user creates a project with Kubebuilder `v4.4.3`.
2. When Kubebuilder `v4.5.0` is released, a **pull request** is automatically created.
3. The PR includes scaffold updates while preserving the user’s customizations, allowing easy review and merging.

### Local Tool Usage:

1. A user creates a project with Kubebuilder `v4.4.3`
2. When Kubebuilder `v4.5.0` is released, they run `kubebuilder alpha update` which calls `kubebuilder alpha generate` behind the scenes
3. The tool updates the scaffold and preserves customizations for review and application.
4. In case of conflicts, the tool allows users to resolve them before push a pull request with the changes.

### Handling Merge Conflicts

**Local Tool Usage**:

If conflicts cannot be resolved automatically, developers can manually address
them before completing the update.

**GitHub Actions Workflow**:

If conflicts arise during the merge, the action will create a pull request and
the conflicst will be highlighted in the PR. Developers can then review and resolve
them. The PR will contains the default markers:

**Example**

```go
<<<<<<< HEAD
	_ = logf.FromContext(ctx)
=======
log := log.FromContext(ctx)
>>>>>>> original
```

## Open Questions

### 1. Do we need to create branches to perform the three-way merge,or can we use local temporary directories?

> While temporary directories are sufficient for simple three-way merges, branches are better suited for complex scenarios.
> They provide history tracking, support collaboration, integrate with CI/CD workflows, and offer more advanced
> conflict resolution through Git’s merge command. For these reasons, it seems more appropriate to use branches to ensure
> flexibility and maintainability in the merging process.

> Furthermore, branches allows a better resolution strategy,
> since allows us to use `kubebuilder alpha generate` command to-rescaffold the projects
> using the same name directory and provide a better history for the PRs
> allowing users to see the changes and have better insights for conflicts
> resolution.

### 2. What Git configuration options can facilitate the three-way merge?

Several Git configuration options can improve the three-way merge process:

```bash
# Show all three versions (base, current, and updated) during conflicts
git config --global merge.conflictStyle diff3

# Enable "reuse recorded resolution" to remember and reuse previous conflict resolutions
git config --global rerere.enabled true

# Increase the rename detection limit to better handle renamed or moved files
git config --global merge.renameLimit 999999
```

These configurations enhance the merging process by improving conflict visibility,
reusing resolutions, and providing better file handling, making three-way
merges more efficient and developer-friendly.

### 3. If we change Git configurations, can we isolate these changes to avoid affecting the local developer environment when the tool runs locally?

It seems that changes can be made using the `-c` flag, which applies the
configuration only for the duration of a specific Git command. This ensures
that the local developer environment remains unaffected.

For example:

```
git -c merge.conflictStyle=diff3 -c rerere.enabled=true merge
```

### 4. How can we minimize and resolve conflicts effectively during merges?

- **Enable Git Features:**
    - Use `git config --global rerere.enabled true` to reuse previous conflict resolutions.
    - Configure custom merge drivers for specific file types (e.g., `git config --global merge.&lt;driver&gt;.name "Custom Merge Driver"`).

- **Encourage Standardization:**
    - Adopt a standardized scaffold layout to minimize divergence and reduce conflicts.

- **Apply Frequent Updates:**
    - Regularly update projects to avoid significant drift between the scaffold and customizations.

These strategies help minimize conflicts and simplify their resolution during merges.

### 5. How to create the PR with the changes for projects that are monorepos?
That means the result of Kubebuilder is not defined in the root dir and might be in other paths.

We can define an `--output` directory and a configuration for the GitHub Action where
users will define where in their repo the path for the Kubebuilder project is.
However, this might be out of scope for the initial version.

### 6. How could AI help us solve conflicts? Are there any available solutions?

While AI tools like GitHub Copilot can assist in code generation and provide suggestions,
however, it might be risky be 100% dependent on AI for conflict resolution, especially in complex scenarios.
Therefore, we might want to use AI as a complementary tool rather than a primary solution.

AI can help by:
- Providing suggestions for resolving conflicts based on context.
- Analyzing code patterns to suggest potential resolutions.
- Offering explanations for conflicts and suggesting best practices.
- Assisting in summarizing changes.

## Summary

### Workflow Example:

1. A developer creates a project with Kubebuilder `v4.4`.
2. The tooling uses the release of Kubebuilder `v4.5`.
3. The tool:
   - Regenerates the original base source code for `v4.4` using the `clientVersion` in the `PROJECT` file.
   - Generates the base source code for `v4.5`
4. A three-way merge integrates the changes into the developer’s project while retaining custom code.
5. The changes now can be packaged into a pull request, summarizing updates and conflicts for the developer’s review.

### Steps:

The proposed implementation involves the following steps:

1. **Version Tracking**:
   - Record the `clientVersion` (initial Kubebuilder version) in the `PROJECT` file.
   - Use this version as a baseline for updates.
   - Available in the `PROJECT` file, from [v4.6.0](https://github.com/kubernetes-sigs/kubebuilder/releases/tag/v4.6.0) release onwards.

2. **Scaffold Generation**:
    - Generate the **original scaffold** using the recorded version.
    - Generate the **updated scaffold** using the latest Kubebuilder release.

3. **Three-Way Merge**:
   - Ensure git is configured to handle three-way merges.
   - Merge the original scaffold, updated scaffold, and the user’s customized project.
   - Preserve custom code during the merge.

4. **(For Actions) - Pull Request Creation**:
   - Open a pull request summarizing changes, including details on conflict resolution.
   - Schedule updates weekly or provide an on-demand option.

#### Example Workflow

The following example code illustrates the proposed idea but has not been evaluated.
This is an early, incomplete draft intended to demonstrate the approach and basic concept.

We may want to develop a dedicated command-line tool, such as `kubebuilder alpha update`,
to handle tasks like downloading binaries, merging, and updating the scaffold. In this approach,
the GitHub Action would simply invoke this tool to manage the update process and open the
Pull Request, rather than performing each step directly within the Action itself.

```yaml
name: Workflow Auto-Update

permissions:
  contents: write
  pull-requests: write

on:
  workflow_dispatch:
  schedule:
    - cron: "0 0 * * 1"    # Every Monday 00:00 UTC

jobs:
  alpha-update:
    runs-on: ubuntu-latest

    steps:
      # 1) Checkout the repository with full history
      - name: Checkout repository
        uses: actions/checkout@v4
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          fetch-depth: 0

      # 2) Install the latest stable Go toolchain
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: 'stable'

      # 3) Install Kubebuilder CLI
      - name: Install Kubebuilder
        run: |
          curl -L -o kubebuilder "https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)"
          chmod +x kubebuilder
          sudo mv kubebuilder /usr/local/bin/

      # 4) Extract Kubebuilder version (e.g., v4.6.0) for branch/title/body
      - name: Get Kubebuilder version
        id: kb
        shell: bash
        run: |
          RAW="$(kubebuilder version 2>/dev/null || true)"
          VERSION="$(printf "%s" "$RAW" | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1)"
          echo "version=${VERSION:-vunknown}" >> "$GITHUB_OUTPUT"

      # 5) Run kubebuilder alpha update
      - name: Run kubebuilder alpha update
        run: |
          kubebuilder alpha update --force

      # 6) Restore workflow files so the update doesn't overwrite CI config
      - name: Restore workflows directory
        run: |
          git restore --source=main --staged --worktree .github/workflows
          git add .github/workflows
          git commit --amend --no-edit || true

      # 7) Push to a versioned branch; create PR if missing, otherwise it just updates
      - name: Push branch and create/update PR
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        shell: bash
        run: |
          set -euo pipefail
          VERSION="${{ steps.kb.outputs.version }}"
          PR_BRANCH="kubebuilder-update-to-${VERSION}"

          # Create or update the branch and push
          git checkout -B "$PR_BRANCH"
          git push -u origin "$PR_BRANCH" --force

          PR_TITLE="chore: update scaffolding to Kubebuilder ${VERSION}"
          PR_BODY=$'Automated update of Kubebuilder project scaffolding to '"${VERSION}"$'.\n\nMore info: https://github.com/kubernetes-sigs/kubebuilder/releases\n\n :warning: If conflicts arise, resolve them and run:\n```bash\nmake manifests generate fmt vet lint-fix\n```'

          # Try to create the PR; ignore error only if it already exists
          if ! gh pr create \
            --title "${PR_TITLE}" \
            --body "${PR_BODY}" \
            --base main \
            --head "$PR_BRANCH"
          then
            EXISTING="$(gh pr list --state open --head "$PR_BRANCH" --json number --jq '.[0].number' || true)"
            if [ -n "${EXISTING}" ]; then
              echo "PR #${EXISTING} already exists for ${PR_BRANCH}, branch updated."
            else
              echo "Failed to create PR for ${PR_BRANCH} and no open PR found."
              exit 1
            fi
          fi
```

## Motivation

A significant challenge faced by Kubebuilder users is keeping their projects up-to-date with the latest
scaffolds while preserving customizations. The manual processes required for updates are time-consuming,
error-prone, and often discourage users from adopting new versions, leading to outdated and insecure projects.

The primary motivation for this proposal is to simplify and automate the process of maintaining Kubebuilder
projects. By providing a streamlined workflow for updates, this solution ensures that users can keep
their projects aligned with modern standards while retaining their customizations.

### Goals

- **Automate Updates**: Detect and apply scaffold updates while preserving customizations.
- **Simplify Updates**: Generate pull requests for easy review and merging.
- **Provide Local Tooling**: Allow developers to run updates locally with preserved customizations.
- **Keep Projects Current**: Ensure alignment with the latest scaffold improvements.
- **Minimize Disruptions**: Enable scheduled or on-demand updates.

### Non-Goals

- **Automating conflict resolution for heavily customized projects**.
- **Automatically merging updates without developer review**.
- **Supporting monorepo project layouts or handling repositories that contain more than just the Kubebuilder-generated code**.

## Proposal

### User Stories

- **As a Kubebuilder maintainer**, I want to help users keep their projects updated with minimal effort, ensuring they adhere to best practices and maintain alignment with project standards.
- **As a user of Kubebuilder**, I want my project to stay up-to-date with the latest scaffold best practices while preserving customizations.
- **As a user of Kubebuilder**, I want an easy way to apply updates across multiple repositories, saving time on manual updates.
- **As a user of Kubebuilder**, I want to ensure my codebases remain secure and maintainable without excessive manual effort.

### Implementation Details/Notes/Constraints

- Introduce a new [Kubebuilder Plugin](https://book.kubebuilder.io/plugins/plugins) that scaffolds the
  **GitHub Action** based on the POC. This plugin will be released as an **alpha feature**,
  allowing users to opt-in for automated updates.

- The plugin should be added by default in the Golang projects build with Kubebuilder, so new
  projects can benefit from the automated updates without additional configuration. While it will not be escaffolded
  by default in tools which extend Kubebuilder such as the Operator-SDK, where the alpha generate and update
  features cannot be ported or extended.

- Documentation should be provided to guide users on how to enable and use the new plugin as the new alpha command

- The alpha command update should
  - provide help and examples of usage
  - allow users to specify the version of Kubebuilder they want to update to or from to
  - allow users to specify the path of the project they want to update
  - allow users to specify the output directory where the updated scaffold should be generated
  - re-use the existing `kubebuilder alpha generate` command to generate the updated scaffold

- The `kubebuilder alpha update` command should be covered with e2e tests to ensure it works as expected
  and that the generated scaffold is valid and can be built.

## Risks and Mitigations
- **Risk**: Frequent conflicts may make the process cumbersome.
    - *Mitigation*: Provide clear conflict summaries and leverage GitHub preview tools.
- **Risk**: High maintenance overhead.
    - *Mitigation*: Build a dedicated command-line tool (`kubebuilder alpha update`) to streamline updates and minimize complexity.

## Proof of Concept

The feasibility of re-scaffolding projects has been demonstrated by the
`kubebuilder alpha generate` command.

**Command Example:**

```bash
kubebuilder alpha generate
```

For more details, refer to the [Alpha Generate Documentation](https://kubebuilder.io/reference/rescaffold).

This command allows users to manually re-scaffold a project, to allow users add their code on top.
It confirms the technical capability of regenerating and updating scaffolds effectively.

This proposal builds upon this foundation by automating the process. The proposed tool would extend this functionality
to automatically update projects with new scaffold versions, preserving customizations.

The three-way merge approach is a common strategy for integrating changes from multiple sources.
It is widely used in version control systems to combine changes from a common ancestor with two sets of modifications.
In the context of this proposal, the three-way merge would combine the original scaffold, the updated scaffold, and the user’s custom code
seems to be very promising.

### POC Implementation using 3-way merge:

Following some POCs done to demonstrate the three-way merge approach
where a project was escaffolded with Kubebuilder `v4.5.0` or `v4.5.2`
and then updated to `v4.6.0`

```shell
## The following options were passed when merging UPGRADE:

git config --global merge.yaml.name "Custom YAML merge"
git config --global merge.yaml.driver "yaml-merge %O %A %B"
git config merge.conflictStyle diff3
git config rerere.enabled true
git config merge.renameLimit 999999
Here are the steps taken:

## On main:

git checkout -b ancestor
Clean up the ancestor and commit

rm -fr *
git add .
git commit -m "clean up ancestor"

## Bring back the PROJECT file, re-scaffold with v4.5.0, and commit

git checkout main -- PROJECT
kubebuilder alpha generate
git add .
git commit -m "alpha generate on ancestor with 4.5.0"
## Then proceed to create the original (ours) branch, bring back the code on main, add and commit:

git checkout -b original
git checkout main -- .
git add .
git commit -m "add code back in original"

## Then create the upgrade branch (theirs), run kubebuilder alpha generate with v4.6.0 add and commit:

git checkout ancestor
git checkout -b upgrade
kubebuilder alpha generate
git add .
git commit -m "alpha generate on upgrade with 4.6.0"

## So now we have the ancestor, the original, and the upgrade branches all set, we can create a branch to commit the merge with the conflict markers:

git checkout original
git checkout -b merge
git merge upgrade
git add .
git commit -m "Merge with upgrade with conflict markers"
## Now that we have performed the three way merge and commited the conflict markers, we can open a PR against main.
```

As the script:

```bash
#!/bin/bash

set -euo pipefail

# CONFIG — change as needed
REPO_PATH="$HOME/go/src/github/camilamacedo86/wordpress-operator"
KUBEBUILDER_SRC="$HOME/go/src/sigs.k8s.io/kubebuilder"
PROJECT_FILE="PROJECT"

echo "📦 Kubebuilder 3-way merge upgrade (v4.5.0 → v4.6.0)"
echo "📂 Working in: $REPO_PATH"
echo "🧪 Kubebuilder source: $KUBEBUILDER_SRC"

cd "$REPO_PATH"

# Step 1: Create ancestor branch and clean it up
echo "🌱 Creating 'ancestor' branch"
git checkout -b ancestor main

echo "🧼 Cleaning all files and folders (including dotfiles), except .git and PROJECT"
find . -mindepth 1 -maxdepth 1 ! -name '.git' ! -name 'PROJECT' -exec rm -rf {} +

git add -A
git commit -m "Clean ancestor branch"

# Step 2: Install Kubebuilder v4.5.0 and regenerate scaffold
echo "⬇️ Installing Kubebuilder v4.5.0"
cd "$KUBEBUILDER_SRC"
git checkout upstream/release-4.5
make install
kubebuilder version

cd "$REPO_PATH"
echo "📂 Restoring PROJECT file"
git checkout main -- "$PROJECT_FILE"
kubebuilder alpha generate
make manifests generate fmt vet lint-fix
git add -A
git commit -m "alpha generate on ancestor with v4.5.0"

# Step 3: Create original branch with user's code
echo "📦 Creating 'original' branch with user code"
git checkout -b original
git checkout main -- .
git add -A
git commit -m "Add project code into original"

# Step 4: Install Kubebuilder v4.6.0 and scaffold upgrade
echo "⬆️ Installing Kubebuilder v4.6.0"
cd "$KUBEBUILDER_SRC"
git checkout upstream/release-4.6
make install
kubebuilder version

cd "$REPO_PATH"
echo "🌿 Creating 'upgrade' branch from ancestor"
git checkout ancestor
git checkout -b upgrade
echo "🧼 Cleaning all files and folders (including dotfiles), except .git and PROJECT"
find . -mindepth 1 -maxdepth 1 ! -name '.git' ! -name 'PROJECT' -exec rm -rf {} +

kubebuilder alpha generate
make manifests generate fmt vet lint-fix
git add -A
git commit -m "alpha generate on upgrade with v4.6.0"

# Step 5: Merge original into upgrade and preserve conflicts
echo "🔀 Creating 'merge' branch from upgrade and merging original"
git checkout upgrade
git checkout -b merge

# Do a non-interactive merge and commit manually
echo "🤖 Running non-interactive merge..."
set +e
git merge --no-edit --no-commit original
MERGE_EXIT_CODE=$?
set -e

# Stage everything and commit with an appropriate message
if [ $MERGE_EXIT_CODE -ne 0 ]; then
  # Manually the alpha generate should out put the info so the person can fix it
  echo "⚠️ Conflicts occurred."
  echo "You will need to fix the conflicts manually and run the following commands:"
  echo "make manifests generate fmt vet lint-fix"
  echo "⚠️ Conflicts occurred. Keeping conflict markers and committing them."
  git add -A
  git commit -m "upgrade has conflicts to be solved"
else
  echo "Merge successful with no conflicts. Running commands"
  make manifests generate fmt vet lint-fix

  echo "✅ Merge successful with no conflicts."
  git add -A
  git commit -m "upgrade worked without conflicts"
fi

echo ""
echo "📍 You are now on the 'merge' branch."
echo "📤 Push with: git push -u origin merge"
echo "🔁 Then open a PR to 'main' on GitHub."
echo ""
```

## Drawbacks

- **Frequent Conflicts:** Automated updates may often result in conflicts, making the process cumbersome for users.
- **Complex Resolutions:** If conflicts are hard to review and resolve, users may find the solution impractical.
- **Maintenance Overhead:** The implementation could become too complex for maintainers to develop and support effectively.

## Alternatives

- **Manual Update Workflow**: Continue with manual updates where users regenerate
and merge changes independently, though this is time-consuming and error-prone.
- **Use alpha generate command**: Continue with partially automated updates provided
by the alpha generate command.
- **Dependabot Integration**: Leverage Dependabot for dependency updates, though this
doesn’t fully support scaffold updates and could lead to incomplete upgrades.
