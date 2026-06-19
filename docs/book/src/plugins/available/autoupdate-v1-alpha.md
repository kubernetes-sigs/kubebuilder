# AutoUpdate (`autoupdate/v1-alpha`)

Keeping your Kubebuilder project up to date with the latest improvements shouldn't be a chore.
With a small amount of setup, you can receive **automatic Pull Request** suggestions whenever a new
Kubebuilder release is available — keeping your project **maintained, secure, and aligned with ecosystem changes**.

This automation uses the [`kubebuilder alpha update`][alpha-update-command] command with a **3-way merge strategy** to
refresh your project scaffold, and wraps it in a GitHub Actions workflow that opens an **Issue** with a **Pull Request compare link** so you can create the PR and review it.

<aside class="warning" role="note">
<p class="note-title">Protect your branches</p>

This workflow by default **only** creates and pushes the merged files to a branch
called `kubebuilder-update-from-<from-version>-to-<to-version>`.

To keep your codebase safe, use branch protection rules to ensure that
changes are not pushed or merged without proper review.

</aside>

## When to use it


- When you want to reduce the burden of keeping the project updated and well-maintained.

## How to use it

- If you want to add the `autoupdate` plugin to your project:

```shell
kubebuilder edit --plugins="autoupdate/v1-alpha"
```

## How it works

The plugin scaffolds a GitHub Actions workflow that checks for new Kubebuilder releases every week. When an update is available, it:

1. Creates a new branch with the merged changes
2. Opens a GitHub Issue with a PR compare link

**Example Issue:**

<img width="638" height="482" alt="Example Issue" src="https://github.com/user-attachments/assets/589fd16b-7709-4cd5-b169-fd53d69790d4" />

**Conflict help** (when needed):

<img width="600" height="188" alt="Conflicts" src="https://github.com/user-attachments/assets/2142887a-730c-499a-94df-c717f09ab600" />

## Customizing the workflow

The generated workflow uses the `kubebuilder alpha update` command with default flags. You can customize the workflow by editing `.github/workflows/auto_update.yml` to add additional flags:

**Default flags used:**
- `--force` - Continue even if conflicts occur (automation-friendly)
- `--push` - Automatically push the output branch to remote
- `--restore-path .github/workflows` - Preserve CI workflows from base branch
- `--open-gh-issue` - Create a GitHub Issue with PR compare link

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
      --merge-message "chore: update kubebuilder scaffold" \
      --conflict-message "chore: update with conflicts - review needed"
```

## Demonstration

<iframe width="560" height="315" src="https://www.youtube.com/embed/dHNKx5jPSqc?si=wYwZZ0QLwFij10Sb" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" referrerpolicy="strict-origin-when-cross-origin" allowfullscreen></iframe>

[alpha-update-command]: ./../../reference/commands/alpha_update.md
