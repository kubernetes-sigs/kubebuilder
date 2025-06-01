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
> since allows us use kubebuilder alpha generate to-rescaffold the projects
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

We can define an `--output` directory and a configuration for the GitHub Action where users will define where in their repo the path for the Kubebuilder project is. However, this might be out of scope for the initial version.

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
name: Update Kubebuilder Scaffold

on:
  workflow_dispatch:
  schedule:
    - cron: '0 0 * * 0'

jobs:
  update-scaffold:
    runs-on: ubuntu-latest

    steps:
      - name: Check out the repository
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up environment and dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y curl git jq make

      - name: Install latest Kubebuilder CLI
        run: |
          LATEST_VERSION=$(curl -s https://api.github.com/repos/kubernetes-sigs/kubebuilder/releases/latest | jq -r .tag_name)
          curl -L https://github.com/kubernetes-sigs/kubebuilder/releases/download/${LATEST_VERSION}/kubebuilder_${LATEST_VERSION}_linux_amd64.tar.gz -o kubebuilder.tar.gz
          tar -zxvf kubebuilder.tar.gz
          sudo mv kubebuilder /usr/local/kubebuilder
          echo "/usr/local/kubebuilder/bin" >> $GITHUB_PATH

      - name: Run Kubebuilder update if cliVersion is outdated
        run: |
          CURRENT_VERSION=$(grep "cliVersion" PROJECT | awk '{print $2}' | tr -d '"')
          LATEST_VERSION=$(curl -s https://api.github.com/repos/kubernetes-sigs/kubebuilder/releases/latest | jq -r .tag_name)

          echo "Current Kubebuilder version: $CURRENT_VERSION"
          echo "Latest Kubebuilder release: $LATEST_VERSION"

          if [ "$CURRENT_VERSION" != "$LATEST_VERSION" ]; then
            echo "cliVersion is outdated. Running kubebuilder alpha update..."
            kubebuilder alpha update
          else
            echo "cliVersion is already up-to-date. Skipping update."
            echo "SKIP_UPDATE=true" >> $GITHUB_ENV
          fi

      - name: Check for skipped update
        if: env.SKIP_UPDATE == 'true'
        run: echo "Skipping PR creation because no update was needed."

      - name: Set up Git user for commit
        if: env.SKIP_UPDATE != 'true'
        run: |
          git config user.name "github-actions"
          git config user.email "github-actions@github.com"

      - name: Push merge branch
        if: env.SKIP_UPDATE != 'true'
        run: |
          git checkout merge
          git push -u origin merge

      - name: Create Pull Request from merge branch
        if: env.SKIP_UPDATE != 'true'
        uses: peter-evans/create-pull-request@v6
        with:
          commit-message: "Update Kubebuilder scaffold"
          title: "Update Kubebuilder to ${LATEST_VERSION}"
          body: |
            This pull request updates the project scaffold to the latest
            Kubebuilder release, preserving customizations.

            Note that this PR may contain merge conflicts that need to be resolved.
          branch: merge
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
- **In Phase 1, supporting monorepo project layouts or handling repositories that contain more than just the Kubebuilder-generated code**.

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

Initial implementation of the three-way merge has been successfully tested using a custom script:
https://github.com/vitorfloriano/multiversion/pull/1

Some further examples of the three-way merge and the proposed solution above
in action with a script to re-scaffold a project can be found here:

- Without conflicts:
https://github.com/camilamacedo86/wordpress-operator/pull/1

- With conflicts:
https://github.com/camilamacedo86/test-operator/pull/1

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
