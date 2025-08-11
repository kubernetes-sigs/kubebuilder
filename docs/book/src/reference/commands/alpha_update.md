# Update Your Project with (`alpha update`)

## Overview

`kubebuilder alpha update` upgrades your project’s scaffold to a newer scaffold version.

It uses a **3-way merge** so you do less manual work. The command creates these branches:

- **Ancestor**: clean scaffold from the **old** version
- **Original**: your current project (from your base branch)
- **Upgrade**: clean scaffold from the **new** version
- **Merge**: result of merging **Original** into **Upgrade** (this is where conflicts appear)

You can review and test the merge result before applying it to your main branch.
Optionally, use **`--squash`** to put the merge result into **one commit** on a stable output branch (great for PRs).

<aside class="note warning">
<h1>Creates branches and deletes files</h1>

This command creates branches like `tmp-kb-update-*` and removes files during the process.
Make sure your work is committed before you run it.

</aside>

## When to Use It

Use this command when you:

- Want to move to a newer Kubebuilder version or plugin layout
- Prefer automation over manual file editing
- Want to review scaffold changes on a separate branch
- Want to focus on resolving merge conflicts (not re-applying your custom code)

## How It Works

1. **Detect versions**
   Reads `--from-version` (or the `PROJECT` file) and `--to-version` (or uses the latest).

2. **Create branches & re-scaffold**
   - `tmp-ancestor-*`: clean scaffold from **from-version**
   - `tmp-original-*`: snapshot of your **from-branch** (e.g., `main`)
   - `tmp-upgrade-*`: clean scaffold from **to-version**

3. **3-way merge**
   Creates `tmp-merge-*` from **Upgrade** and merges **Original** into it.
   Runs `make manifests generate fmt vet lint-fix` to normalise outputs.
   Runs `make manifests generate fmt vet lint-fix` to normalize outputs.

4. **(Optional) Squash**
   With `--squash`, copies the merge result to a stable output branch and commits **once**:
   - Default output branch: `kubebuilder-alpha-update-to-<to-version>`
   - Or set your own with `--output-branch`
     If there are conflicts, the single commit will include conflict markers.

## How to Use It

Run from your project root:

```shell
kubebuilder alpha update
```

Pin versions and base branch:

```shell
kubebuilder alpha update \
  --from-version v4.5.2 \
  --to-version   v4.6.0 \
  --from-branch  main
```
Automation-friendly (proceed even with conflicts):

```shell
kubebuilder alpha update --force
```

Create a **single squashed commit** on a stable PR branch:

```shell
kubebuilder alpha update --force --squash
```

Squash while **preserving** paths from your base branch (keep CI/workflows, docs, etc.):

```shell
kubebuilder alpha update --force --squash \
  --preserve-path .github/workflows \
  --preserve-path docs
```

Use a **custom output branch** name:

```shell
kubebuilder alpha update --force --squash \
  -output-branch upgrade/kb-to-v4.7.0
```

## Merge Conflicts with `--force`

When you use `--force`, Git finishes the merge even if there are conflicts.
The commit will include markers like:

```shell
<<<<<<< HEAD
Your changes
=======
Incoming changes
>>>>>>> tmp-original-…
```

- **Without `--force`**: the command stops on `tmp-merge-*` and prints guidance; no commit is created.
- **With `--force`**: the merge is committed (on `tmp-merge-*`, or on the output branch if using `--squash`) and contains the markers.

## Commit message used in `--squash` mode

> [kubebuilder-automated-update]: update scaffold from <from> to <to>; (squashed 3-way merge)

<aside class="note warning">
<h1>You might need to upgrade your project first</h1>

This command uses `kubebuilder alpha generate` under the hood.
We support projects created with <strong>v4.5.0+</strong>.
If yours is older, first run `kubebuilder alpha generate` once to modernize the scaffold.
After that, you can use `kubebuilder alpha update` for future upgrades.

</aside>

<aside class="note">
<h1>CLI Version Tracking</h1>

Projects created with **Kubebuilder v4.6.0+** include `cliVersion` in the `PROJECT` file.
We use that value to pick the correct CLI for re-scaffolding.

</aside>

<aside class="note warning">
You must resolve these conflicts before merging into `main` (or your base branch).
<strong>After resolving conflicts, always run:</strong>

```shell
make manifests generate fmt vet lint-fix
# or
make all
```

</aside>

## Flags

| Flag              | Description                                                                                                                                |
|-------------------|--------------------------------------------------------------------------------------------------------------------------------------------|
| `--from-version`  | Kubebuilder version your project was created with. If unset, taken from the `PROJECT` file. |
| `--to-version`    | Version to upgrade to. Defaults to the latest release.                                                                                     |
| `--from-branch`   | Git branch that has your current project code. Defaults to `main`.                                                                          |
| `--force`         | Continue even if merge conflicts happen. Conflicted files are committed with conflict markers (useful for CI/cron).                         |
| `--squash`        | Write the merge result as **one commit** on a stable output branch.                                                                         |
| `--preserve-path` | Repeatable. With `--squash`, restore these paths from the base branch (e.g., `--preserve-path .github/workflows`).                          |
| `--output-branch` | Branch name to use for the squashed commit (default: `kubebuilder-alpha-update-to-<to-version>`).                                          |
| `-h, --help`      | Show help for this command.                                                                                                                |

<aside class="note">
<h1>CLI Version Tracking</h1>

Projects created with **Kubebuilder v4.6.0+** include `cliVersion` in the `PROJECT` file.
We use that value to pick the correct CLI for re-scaffolding.

If your project was created with an older version,
you can set `--from-version` to the version you used.

However, this command uses `kubebuilder alpha generate` under the hood.
We tested projects created with <strong>v4.5.0+</strong>.
If yours is older, first run `kubebuilder alpha generate` once to modernize the scaffold.
After that, you can use `kubebuilder alpha update` for future upgrades.

</aside>

## Demonstration

<iframe width="560" height="315" src="https://www.youtube.com/embed/J8zonID__8k?si=WC-FXOHX0mCjph71" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" referrerpolicy="strict-origin-when-cross-origin" allowfullscreen></iframe>

## Further Resources

- WIP: Design proposal for update automation — https://github.com/kubernetes-sigs/kubebuilder/pull/4302

[project-config]: ../../reference/project-config.md
