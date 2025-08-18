# Update Your Project with (`alpha update`)

## Overview

`kubebuilder alpha update` upgrades your project’s scaffold to a newer Kubebuilder release using a **3-way Git merge**. It rebuilds clean scaffolds for the old and new versions, merges your current code into the new scaffold, and gives you a reviewable output branch.
It takes care of the heavy lifting so you can focus on reviewing and resolving conflicts,
not re-applying your code.

By default, the final result is **squashed into a single commit** on a dedicated output branch.
If you prefer to keep the full history (no squash), use `--show-commits`.

## When to Use It

Use this command when you:

- Want to move to a newer Kubebuilder version or plugin layout
- Prefer automation over manual file editing
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

### Changing Extra Git configs only during the run (does not change your ~/.gitconfig)_

By default, `kubebuilder alpha update` applies safe Git configs:
`merge.renameLimit=999999`, `diff.renameLimit=999999`.
You can add more, or disable them.

- **Add more on top of defaults**
```shell
kubebuilder alpha update \
  --git-config merge.conflictStyle=diff3 \
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

## Flags

| Flag               | Description                                                                                                                                                                                                                            |
|--------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `--from-version`   | Kubebuilder release to update **from** (e.g., `v4.6.0`). If unset, read from the `PROJECT` file when possible.                                                                                                                         |
| `--to-version`     | Kubebuilder release to update **to** (e.g., `v4.7.0`). If unset, defaults to the latest available release.                                                                                                                             |
| `--from-branch`    | Git branch that holds your current project code. Defaults to `main`.                                                                                                                                                                   |
| `--force`          | Continue even if merge conflicts happen. Conflicted files are committed with conflict markers (CI/cron friendly).                                                                                                                      |
| `--show-commits`   | Keep full history (do not squash). **Not compatible** with `--preserve-path`.                                                                                                                                                          |
| `--restore-path`   | Repeatable. Paths to preserve from the base branch (repeatable). Not supported with --show-commits. (e.g., `.github/workflows`).                                                                                                       |
| `--output-branch`  | Name of the output branch. Default: `kubebuilder-update-from-<from-version>-to-<to-version>`.                                                                                                                                          |
| `--push`           | Push the output branch to the `origin` remote after the update completes.                                                                                                                                                              |
| `--git-config`     | Repeatable. Pass per-invocation Git config as `-c key=value`. **Default** (if omitted): `-c merge.renameLimit=999999 -c diff.renameLimit=999999`. Your configs are applied on top. To disable defaults, include `--git-config disable` |
| `-h, --help`       | Show help for this command.                                                                                                                                                                                                            |

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

## Demonstration

<iframe width="560" height="315" src="https://www.youtube.com/embed/J8zonID__8k?si=WC-FXOHX0mCjph71" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" referrerpolicy="strict-origin-when-cross-origin" allowfullscreen></iframe>

<aside class="note">
<h1>About this demo</h1>

This video was recorded with Kubebuilder release `v7.0.1`.
Since then, the command has been improved,
so the current behavior may differ slightly from what is shown in the demo.

</aside>

## Further Resources

- WIP: Design proposal for update automation — https://github.com/kubernetes-sigs/kubebuilder/pull/4302

[project-config]: ../../reference/project-config.md
