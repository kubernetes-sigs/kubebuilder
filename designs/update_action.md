| Authors         | Creation Date | Status      | Extra |
|-----------------|---------------|-------------|-------|
| @camilamacedo86 | 2024-11-07 | Implementable | - |

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
2. When Kubebuilder `v4.5.0` is released, they run the tool locally.
3. The tool updates the scaffold and preserves customizations for review and application.

### Handling Merge Conflicts

If conflicts cannot be resolved automatically, developers can manually address them before completing the update.

## Open Questions

### 1. Do we need to create branches to perform the three-way merge, or can we use local temporary directories?

> While temporary directories are sufficient for simple three-way merges, branches are better suited for complex scenarios. They provide history tracking, support collaboration, integrate with CI/CD workflows, and offer more advanced conflict resolution through Git’s merge command. For these reasons, it seems more appropriate to use branches to ensure flexibility and maintainability in the merging process.

### 2. What Git configuration options can facilitate the three-way merge?

Several Git configuration options can improve the three-way merge process:

```bash
# Show all three versions (base, current, and updated) during conflicts
git config --global merge.conflictStyle diff3

# Enable "reuse recorded resolution" to remember and reuse previous conflict resolutions
git config --global rerere.enabled true

# Increase the rename detection limit to better handle renamed or moved files
git config --global merge.renameLimit 999999

# Set up custom merge drivers for specific file types (e.g., YAML or JSON)
git config --global merge.&lt;driver&gt;.name "Custom Merge Driver"
```

These configurations enhance the merging process by improving conflict visibility, reusing resolutions, and providing better file handling, making three-way merges more efficient and developer-friendly.

### 3. If we change Git configurations, can we isolate these changes to avoid affecting the local developer environment when the tool runs locally?

It seems that changes can be made using the `-c` flag, which applies the configuration only for the duration of a specific Git command. This ensures that the local developer environment remains unaffected.

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

> <TODO>

### 7. Could GitHub Copilot help solve conflicts? Does it provide an API we can leverage?

> <TODO>

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
   - [More info](https://github.com/kubernetes-sigs/kubebuilder/issues/4398)

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
    - cron: '0 0 * * 0' # Run weekly to check for new Kubebuilder versions

jobs:
  update-scaffold:
    runs-on: ubuntu-latest

    steps:
      - name: Check out the repository
        uses: actions/checkout@v2
        with:
          fetch-depth: 0  # Ensures the full history is checked out

      - name: Set up environment and dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y jq curl

      - name: Read Kubebuilder version from PROJECT file
        id: read_version
        run: |
          export INITIAL_VERSION=$(grep "clientVersion" PROJECT | awk '{print $2}')
          echo "::set-output name=initial_version::$INITIAL_VERSION"

      - name: Download and install the initial Kubebuilder version
        run: |
          curl -L https://github.com/kubernetes-sigs/kubebuilder/releases/download/${{ steps.read_version.outputs.initial_version }}/kubebuilder_${{ steps.read_version.outputs.initial_version }}_linux_amd64.tar.gz -o kubebuilder_initial.tar.gz
          tar -zxvf kubebuilder_initial.tar.gz
          sudo mv kubebuilder /usr/local/kubebuilder_initial

      - name: Generate initial scaffold in `scaffold_initial` directory
        run: |
          mkdir scaffold_initial
          cp -r . scaffold_initial/
          cd scaffold_initial
          /usr/local/kubebuilder_initial/bin/kubebuilder init
          cd ..

      - name: Check for the latest Kubebuilder release
        id: get_latest_version
        run: |
          export LATEST_VERSION=$(curl -s https://api.github.com/repos/kubernetes-sigs/kubebuilder/releases/latest | jq -r .tag_name)
          echo "::set-output name=latest_version::$LATEST_VERSION"

      - name: Download and install the latest Kubebuilder version
        run: |
          curl -L https://github.com/kubernetes-sigs/kubebuilder/releases/download/${{ steps.get_latest_version.outputs.latest_version }}/kubebuilder_${{ steps.get_latest_version.outputs.latest_version }}_linux_amd64.tar.gz -o kubebuilder_latest.tar.gz
          tar -zxvf kubebuilder_latest.tar.gz
          sudo mv kubebuilder /usr/local/kubebuilder_latest

      - name: Generate updated scaffold in `scaffold_updated` directory
        run: |
          mkdir scaffold_updated
          cp -r . scaffold_updated/
          cd scaffold_updated
          /usr/local/kubebuilder_latest/bin/kubebuilder init
          cd ..

      - name: Copy current project into `scaffold_current` directory
        run: |
          mkdir scaffold_current
          cp -r . scaffold_current/

      - name: Perform three-way merge with scaffolds
        run: |
          # Create a temporary directory to hold the final merged version
          mkdir merged_scaffold
          # Run three-way merge using scaffold_initial, scaffold_current, and scaffold_updated
          # Adjusting merge strategy and paths to use directories
          diff3 -m scaffold_current scaffold_initial scaffold_updated > merged_scaffold/merged_files

      - name: Copy merged files back to main directory
        run: |
          cp -r merged_scaffold/* .
          git add .
          git commit -m "Three-way merge with Kubebuilder updates and custom code"

      - name: Create Pull Request
        uses: peter-evans/create-pull-request@v3
        with:
          commit-message: "Update scaffold to Kubebuilder ${{ steps.get_latest_version.outputs.latest_version }}"
          title: "Update scaffold to Kubebuilder ${{ steps.get_latest_version.outputs.latest_version }}"
          body: |
            This pull request updates the scaffold with the latest Kubebuilder version ${{ steps.get_latest_version.outputs.latest_version }}.
          branch: kubebuilder-update-${{ steps.get_latest_version.outputs.latest_version }}
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

> <TODO>

## Risks and Mitigations
- **Risk**: Frequent conflicts may make the process cumbersome.
    - *Mitigation*: Provide clear conflict summaries and leverage GitHub preview tools.
- **Risk**: High maintenance overhead.
    - *Mitigation*: Build a dedicated command-line tool (`kubebuilder alpha update`) to streamline updates and minimize complexity.

## Proof of Concept

The feasibility of re-scaffolding projects has been demonstrated by the `kubebuilder alpha generate` command.

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

<TODO: Add POC>

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
