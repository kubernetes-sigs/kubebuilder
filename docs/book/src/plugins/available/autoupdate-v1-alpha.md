# AutoUpdate (`autoupdate/v1-alpha`)

Keeping your Kubebuilder project up to date with the latest improvements shouldn’t be a chore.
With a small amount of setup, you can receive **automatic Pull Request** suggestions whenever a new
Kubebuilder release is available — keeping your project **maintained, secure, and aligned with ecosystem changes**.

This automation uses the [`kubebuilder alpha update`][alpha-update-command] command with a **3-way merge strategy** to
refresh your project scaffold, and wraps it in a GitHub Actions workflow that opens an **Issue** with a **Pull Request compare link** so you can create the PR and review it.

<aside class="warning">
<h3>Protect your branches</h3>

This workflow by default **only** creates and pushes the merged files to a branch
called `kubebuilder-update-from-<from-version>-to-<to-version>`.

To keep your codebase safe, use branch protection rules to ensure that
changes aren't pushed or merged without proper review.

</aside>

## When to Use It


- When you want to reduce the burden of keeping the project updated and well-maintained.
- When you want guidance and help from AI to know what changes are needed to keep your project up to date and to solve conflicts (requires `--use-gh-models` flag and GitHub Models permissions).

## How to Use It

- If you want to add the `autoupdate` plugin to your project:

```shell
kubebuilder edit --plugins="autoupdate/v1-alpha"
```

- If you want to create a new project with the `autoupdate` plugin:

```shell
kubebuilder init --plugins=go/v4,autoupdate/v1-alpha
```

### Optional: GitHub Models AI Summary

By default, the workflow works without GitHub Models to avoid permission errors.
If you want AI-generated summaries in your update issues:

```shell
kubebuilder edit --plugins="autoupdate/v1-alpha" --use-gh-models
```

<aside class="note">
<h1>Permissions required to use GitHub Models in GitHub Actions</h1>

To use GitHub Models in your workflows, organization and repository administrators must grant this permission.

**If you have admin access:**

1. Go to **Settings → Code and automation → Models**
2. Enable GitHub Models for your repository

**Don't see the Models option?**

Your organization or enterprise may have disabled it. Contact your administrator:

- Organization admins: [Managing Models in your organization][manage-org-models]
- Enterprise admins: [Managing Models at enterprise scale][manage-models-at-scale]

</aside>

## How It Works

The plugin scaffolds a GitHub Actions workflow that checks for new Kubebuilder releases every week. When an update is available, it:

1. Creates a new branch with the merged changes
2. Opens a GitHub Issue with a PR compare link

**Example Issue:**

<img width="638" height="482" alt="Example Issue" src="https://github.com/user-attachments/assets/589fd16b-7709-4cd5-b169-fd53d69790d4" />

**With GitHub Models enabled** (optional), you also get AI-generated summaries:

<img width="582" height="646" alt="AI Summary" src="https://github.com/user-attachments/assets/d460a5af-5ca4-4dd5-afb8-7330dd6de148" />

**Conflict help** (when needed):

<img width="600" height="188" alt="Conflicts" src="https://github.com/user-attachments/assets/2142887a-730c-499a-94df-c717f09ab600" />

## Troubleshooting

#### If you get the 403 Forbidden Error

**Error message:**
```
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
  issues: write
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
        --open-gh-issue
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
  issues: write
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
    # --use-gh-models: Adds an AI-generated comment to the Issue with
    #   a summary of scaffold changes and conflict-resolution guidance (if any).
    run: |
      kubebuilder alpha update \
        --force \
        --push \
        --restore-path .github/workflows \
        --open-gh-issue \
        --use-gh-models
```

## Demonstration

<iframe width="560" height="315" src="https://www.youtube.com/embed/dHNKx5jPSqc?si=wYwZZ0QLwFij10Sb" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" referrerpolicy="strict-origin-when-cross-origin" allowfullscreen></iframe>

[alpha-update-command]: ./../../reference/commands/alpha_update.md
[ai-models]: https://docs.github.com/en/github-models/about-github-models
[manage-models-at-scale]: https://docs.github.com/en/github-models/github-models-at-scale/manage-models-at-scale
[manage-org-models]: https://docs.github.com/en/organizations/managing-organization-settings/managing-or-restricting-github-models-for-your-organization
