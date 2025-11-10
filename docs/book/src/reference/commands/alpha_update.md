# Update Your Project with (`alpha update`)

## Overview

`kubebuilder alpha update` upgrades your project’s scaffold to a newer Kubebuilder release using a **3-way Git merge**. It rebuilds clean scaffolds for the old and new versions, merges your current code into the new scaffold, and gives you a reviewable output branch.
It takes care of the heavy lifting so you can focus on reviewing and resolving conflicts,
not re-applying your code.

By default, the final result is **squashed into a single commit** on a dedicated output branch.
If you prefer to keep the full history (no squash), use `--show-commits`.

<aside class="note">
<H1> Automate this process </H1>

You can reduce the burden of keeping your project up to date by using the
[AutoUpdate Plugin][autoupdate-plugin] which
automates the process of running `kubebuilder alpha update` on a schedule
workflow when new Kubebuilder releases are available.

Moreover, you will be able to get help from [AI models][ai-gh-models] to understand what changes are needed to keep your project up to date
and how to solve conflicts if any are faced.

</aside>

## When to Use It

Use this command when you:

- Want to move to a newer Kubebuilder version or plugin layout
- Want to review scaffold changes on a separate branch
- Want to focus on resolving merge conflicts (not re-applying your custom code)

## How It Works

You tell the tool the **new version**, and which branch has your project.
It rebuilds both scaffolds, merges your code into the new one with a **3-way merge**,
and gives you an output branch you can review and merge safely.
You decide if you want one clean commit, the full history, or an auto-push to remote.

### Step 1: Detect versions
- It looks at your `PROJECT` file or the flags you pass.
- Decides which **old version** you are coming from by reading the `cliVersion` field in the `PROJECT` file (if available).
- Figures out which **new version** you want (defaults to the latest release).
- Chooses which branch has your current code (defaults to `main`).

### Step 2: Create scaffolds
The command creates three temporary branches:
- **Ancestor**: a clean project scaffold from the **old version**.
- **Original**: a snapshot of your **current code**.
- **Upgrade**: a clean scaffold from the **new version**.

### Step 3: Do a 3-way merge
- Merges **Original** (your code) into **Upgrade** (the new scaffold) using Git’s **3-way merge**.
- This keeps your customizations while pulling in upstream changes.
- If conflicts happen:
    - **Default** → stop and let you resolve them manually.
    - **With `--force`** → continue and commit even with conflict markers. **(ideal for automation)**
- Runs `make manifests generate fmt vet lint-fix` to tidy things up.

### Step 4: Write the output branch
- By default, everything is **squashed into one commit** on a safe output branch:
  `kubebuilder-update-from-<from-version>-to-<to-version>`.
- You can change the behavior:
    - `--show-commits`: keep the full history.
    - `--restore-path`: in squash mode, restore specific files (like CI configs) from your base branch.
    - `--output-branch`: pick a custom branch name.
    - `--push`: push the result to `origin` automatically.
    - `--git-config`: sets git configurations.
    - `--open-gh-issue`: create a GitHub issue with a checklist and compare link (requires `gh`).
    - `--use-gh-models`: add an AI overview **comment** to that issue using `gh models`

### Step 5: Cleanup
- Once the output branch is ready, all the temporary working branches are deleted.
- You are left with one clean branch you can test, review, and merge back into your main branch.

## How to Use It (commands)

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

Keep full history instead of squashing:
```
kubebuilder alpha update --from-version v4.5.0 --to-version v4.7.0 --force --show-commits
```

Default squash but **preserve** CI/workflows from the base branch:

```shell
kubebuilder alpha update --force \
--restore-path .github/workflows \
--restore-path docs
```

Use a custom output branch name:

```shell
kubebuilder alpha update --force \
--output-branch upgrade/kb-to-v4.7.0
```

Run update and push the result to origin:

```shell
kubebuilder alpha update --from-version v4.6.0 --to-version v4.7.0 --force --push
```

## Handling Conflicts (`--force` vs default)

When you use `--force`, Git finishes the merge even if there are conflicts.
The commit will include markers like:

```shell
<<<<<<< HEAD
Your changes
=======
Incoming changes
>>>>>>> (original)
```

This allows you to run the command in CI or cron jobs without manual intervention.

- Without `--force`: the command stops on the merge branch and prints guidance; no commit is created.
- With `--force`: the merge is committed (merge or output branch) and contains the markers.

After you fix conflicts, always run:

```shell
make manifests generate fmt vet lint-fix
# or
make all
```

## Using with GitHub Issues (`--open-gh-issue`) and AI (`--use-gh-models`) assistance

Pass `--open-gh-issue` to have the command create a GitHub **Issue** in your repository
to assist with the update. Also, if you also pass `--use-gh-models`, the tool posts a follow-up comment
on that Issue with an AI-generated overview of the most important changes plus brief conflict-resolution
guidance.

### Examples

Create an Issue with a compare link:
```shell
kubebuilder alpha update --open-gh-issue
```

Create an Issue **and** add an AI summary:
```shell
kubebuilder alpha update --open-gh-issue --use-gh-models
```

### What you’ll see

The command opens an Issue that links to the diff so you can create the PR and review it, for example:

<img width="638" height="482" alt="Example Issue" src="https://github.com/user-attachments/assets/589fd16b-7709-4cd5-b169-fd53d69790d4" />

With `--use-gh-models`, an AI comment highlights key changes and suggests how to resolve any conflicts:

<img width="740" height="425" alt="Comment" src="https://github.com/user-attachments/assets/fb5f214e-be0e-43b8-a3fb-b5744ac8f66e" />

Moreover, AI models are used to help you understand what changes are needed to keep your project up to date,
and to suggest resolutions if conflicts are encountered, as in the following example:

### Automation

This integrates cleanly with automation. The [`autoupdate.kubebuilder.io/v1-alpha`][autoupdate-plugin] plugin can scaffold a GitHub Actions workflow that runs the command on a schedule (e.g., weekly). When a new Kubebuilder release is available, it opens an Issue with a compare link so you can create the PR and review it.

## Changing Extra Git configs only during the run (does not change your ~/.gitconfig)_

By default, `kubebuilder alpha update` applies safe Git configs:
`merge.renameLimit=999999`, `diff.renameLimit=999999`, `merge.conflictStyle=merge`
You can add more, or disable them.

- **Add more on top of defaults**
```shell
kubebuilder alpha update \
  --git-config rerere.enabled=true
```

- **Disable defaults entirely**
```shell
kubebuilder alpha update --git-config disable
```

- **Disable defaults and set your own**

```shell
kubebuilder alpha update \
  --git-config disable \
  --git-config rerere.enabled=true
```

<aside class="warning">
    <h3>You might need to upgrade your project first</h3>

This command uses `kubebuilder alpha generate` under the hood.
We support projects created with <strong>v4.5.0+</strong>.
If yours is older, first run `kubebuilder alpha generate` once to modernize the scaffold.
After that, you can use `kubebuilder alpha update` for future upgrades.

Projects created with **Kubebuilder v4.6.0+** include `cliVersion` in the `PROJECT` file.
We use that value to pick the correct CLI for re-scaffolding.

</aside>

## Flags

| Flag               | Description                                                                                                                                                                                                                             |
|--------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `--force`          | Continue even if merge conflicts happen. Conflicted files are committed with conflict markers (CI/cron friendly).                                                                                                                       |
| `--from-branch`    | Git branch that holds your current project code. Defaults to `main`.                                                                                                                                                                    |
| `--from-version`   | Kubebuilder release to update **from** (e.g., `v4.6.0`). If unset, read from the `PROJECT` file when possible.                                                                                                                          |
| `--git-config`     | Repeatable. Pass per-invocation Git config as `-c key=value`. **Default** (if omitted): `-c merge.renameLimit=999999 -c diff.renameLimit=999999`. Your configs are applied on top. To disable defaults, include `--git-config disable`. |
| `--open-gh-issue`  | Create a GitHub issue with a pre-filled checklist and compare link after the update completes (requires `gh`).                                                                                                                          |
| `--output-branch`  | Name of the output branch. Default: `kubebuilder-update-from-<from-version>-to-<to-version>`.                                                                                                                                           |
| `--push`           | Push the output branch to the `origin` remote after the update completes.                                                                                                                                                               |
| `--restore-path`   | Repeatable. Paths to preserve from the base branch when squashing (e.g., `.github/workflows`). **Not supported** with `--show-commits`.                                                                                                 |
| `--show-commits`   | Keep full history (do not squash). **Not compatible** with `--restore-path`.                                                                                                                                                            |
| `--to-version`     | Kubebuilder release to update **to** (e.g., `v4.7.0`). If unset, defaults to the latest available release.                                                                                                                              |
| `--use-gh-models`  | Post an AI overview as an issue comment using `gh models`. Requires `gh` + `gh-models` extension. Effective only when `--open-gh-issue` is also set.                                                                                    |
| `-h, --help`       | Show help for this command.                                                                                                                                                                                                             |

## Demonstration

<iframe width="560" height="315" src="https://www.youtube.com/embed/J8zonID__8k?si=WC-FXOHX0mCjph71" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" referrerpolicy="strict-origin-when-cross-origin" allowfullscreen></iframe>

<aside class="note">
<h1>About this demo</h1>

This video was recorded with Kubebuilder release `v7.0.1`.
Since then, the command has been improved,
so the current behavior may differ slightly from what is shown in the demo.

</aside>

## Further Resources

- [AutoUpdate Plugin][autoupdate-plugin]
- [Design proposal for update automation][design-proposal]
- [Project configuration reference][project-config]

[project-config]: ../../reference/project-config.md
[autoupdate-plugin]: ./../../plugins/available/autoupdate-v1-alpha.md
[design-proposal]: ./../../../../../designs/update_action.md
[ai-gh-models]: https://docs.github.com/en/github-models/about-github-models
