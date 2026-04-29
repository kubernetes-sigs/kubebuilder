# AutoUpdate (`autoupdate/v1-alpha`)

Keeping your Kubebuilder project up to date with the latest improvements shouldn’t be a chore.
With a small amount of setup, you can receive **automatic Pull Request** suggestions whenever a new
Kubebuilder release is available — keeping your project **maintained, secure, and aligned with ecosystem changes**.

This automation uses the [`kubebuilder alpha update`][alpha-update-command] command with a **3-way merge strategy** to
refresh your project scaffold, and wraps it in a GitHub Actions workflow that **creates both GitHub Issues and Pull Requests** by default.

<aside class="warning" role="note">
<p class="note-title">Protect your branches</p>

This workflow by default **only** creates and pushes the merged files to a branch
called `kubebuilder-update-from-<from-version>-to-<to-version>`.

To keep your codebase safe, use branch protection rules to ensure that
changes are not pushed or merged without proper review.

</aside>

## When to use it


- When you want to reduce the burden of keeping the project updated and well-maintained.
- When you want guidance and help from AI to know what changes are needed to keep your project up to date and to solve conflicts (requires `--use-gh-models` flag and GitHub Models permissions).

## How to use it

- If you want to add the `autoupdate` plugin to your project:

```shell
kubebuilder edit --plugins="autoupdate/v1-alpha"
```

### Optional: GitHub Models AI summary

By default, the workflow works without GitHub Models to avoid permission errors.
If you want AI-generated summaries in your update Pull Requests:

```shell
kubebuilder edit --plugins="autoupdate/v1-alpha" --use-gh-models
```

**Note:** AI summaries only work with Pull Requests. To use `--use-gh-models`, ensure `--open-gh-pr=true` (which is the default).

<aside class="note" role="note">
<p class="note-title">Permissions required to use GitHub Models in GitHub Actions</p>

To use GitHub Models in your workflows, organization and repository administrators must grant this permission.

**If you have admin access:**

1. Go to **Settings → Code and automation → Models**
2. Enable GitHub Models for your repository

**Don't see the Models option?**

Your organization or enterprise may have disabled it. Contact your administrator:

- Organization admins: [Managing Models in your organization][manage-org-models]
- Enterprise admins: [Managing Models at enterprise scale][manage-models-at-scale]

</aside>

## How it works

The plugin scaffolds a GitHub Actions workflow that checks for new Kubebuilder releases every week. When an update is available, it:

1. Creates a new branch with the merged changes
2. Opens a GitHub Issue to notify about the update
3. Opens a Pull Request with the changes for review

You can customize which notifications to receive by disabling either `--open-gh-issue` or `--open-gh-pr` when scaffolding the workflow.

**With GitHub Models enabled** (optional), you get AI-generated summaries in the PR description:

<img width="582" height="646" alt="AI Summary" src="https://github.com/user-attachments/assets/d460a5af-5ca4-4dd5-afb8-7330dd6de148" />

**Conflict help** (when needed):

<img width="600" height="188" alt="Conflicts" src="https://github.com/user-attachments/assets/2142887a-730c-499a-94df-c717f09ab600" />

## Customizing the workflow

The generated workflow uses the `kubebuilder alpha update` command with default flags. You can customize the workflow by editing `.github/workflows/auto_update.yml` to add additional flags:

**Default flags used:**
- `--force` - Continue even if conflicts occur (automation-friendly)
- `--push` - Automatically push the output branch to remote
- `--restore-path .github/workflows` - Preserve CI workflows from base branch
- `--open-gh-issue` - Create a GitHub Issue notification (enabled by default)
- `--open-gh-pr` - Create a GitHub Pull Request for review (enabled by default)
- `--use-gh-models` - (optional) Add AI summary to the PR description

**Customization options:**
- `--open-gh-issue=false` - Disable issue notifications (only create PRs)
- `--open-gh-pr=false` - Disable PRs (only create issue notifications)

**Additional available flags:**
- `--merge-message` - Custom commit message for clean merges
- `--conflict-message` - Custom commit message when conflicts occur
- `--from-version` - Specify the version to upgrade from
- `--to-version` - Specify the version to upgrade to
- `--output-branch` - Custom output branch name
- `--show-commits` - Keep full history instead of squashing
- `--git-config` - Pass per-invocation Git config

For complete documentation on all available flags, see the [`kubebuilder alpha update`][alpha-update-command] reference.

**Example: Customize commit messages**

Edit `.github/workflows/auto_update.yml`:

```yaml
- name: Run kubebuilder alpha update
  run: |
    kubebuilder alpha update \
      --force \
      --push \
      --restore-path .github/workflows \
      --open-gh-issue \
      --open-gh-pr \
      --merge-message "chore: update kubebuilder scaffold" \
      --conflict-message "chore: update with conflicts - review needed"
```

**Example: Customize notifications**

If you prefer to receive only Issues (without creating PRs):

```shell
kubebuilder edit --plugins="autoupdate/v1-alpha" --open-gh-pr=false
```

Or to create only PRs (without issue notifications):

```shell
kubebuilder edit --plugins="autoupdate/v1-alpha" --open-gh-issue=false
```

## Troubleshooting

#### If you get the 403 Forbidden Error

**Error message:**
```text
ERROR Update failed error=failed to open GitHub issue: gh models run failed: exit status 1
Error: unexpected response from the server: 403 Forbidden
```

**Quick fix:** Disable GitHub Models (works for everyone)

```shell
kubebuilder edit --plugins="autoupdate/v1-alpha"
```

This regenerates the workflow without GitHub Models:

```yaml
permissions:
  contents: write
  pull-requests: write
  # No models: read permission

steps:
  - name: Checkout repository
    uses: actions/checkout@v4
    # ... other setup steps

  - name: Run kubebuilder alpha update
    # WARNING: This workflow does not use GitHub Models AI summary by default.
    # To enable AI-generated summaries, you need permissions to use GitHub Models.
    # If you have the required permissions, re-run:
    #   kubebuilder edit --plugins="autoupdate/v1-alpha" --use-gh-models
    run: |
      kubebuilder alpha update \
        --force \
        --push \
        --restore-path .github/workflows \
        --open-gh-pr
```

The workflow continues to work—just without AI summaries.

**To enable GitHub Models instead:**

1. Ask your GitHub administrator to enable Models (see links below)
2. Enable it in **Settings → Code and automation → Models**
3. Re-run with:

```shell
kubebuilder edit --plugins="autoupdate/v1-alpha" --use-gh-models
```

This regenerates the workflow WITH GitHub Models:

```yaml
permissions:
  contents: write
  pull-requests: write
  models: read  # Added for GitHub Models

steps:
  - name: Checkout repository
    uses: actions/checkout@v4
    # ... other setup steps

  - name: Install gh-models extension
    run: |
      gh extension install github/gh-models --force
      gh models --help >/dev/null

  - name: Run kubebuilder alpha update
    # --use-gh-models: Adds an AI summary to the PR description.
    run: |
      kubebuilder alpha update \
        --force \
        --push \
        --restore-path .github/workflows \
        --open-gh-pr \
        --use-gh-models
```

## Demonstration

<iframe width="560" height="315" src="https://www.youtube.com/embed/dHNKx5jPSqc?si=wYwZZ0QLwFij10Sb" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" referrerpolicy="strict-origin-when-cross-origin" allowfullscreen></iframe>

[alpha-update-command]: ./../../reference/commands/alpha_update.md
[ai-models]: https://docs.github.com/en/github-models/about-github-models
[manage-models-at-scale]: https://docs.github.com/en/github-models/github-models-at-scale/manage-models-at-scale
[manage-org-models]: https://docs.github.com/en/organizations/managing-organization-settings/managing-or-restricting-github-models-for-your-organization
