# Update Your Project with (`alpha update`)

## Overview

The `kubebuilder alpha update` command helps you upgrade your project scaffold to a newer Kubebuilder version or plugin layout automatically.

It uses a **3-way merge strategy** to update your project with less manual work.
To achieve that, the command creates the following branches:

- *Ancestor branch*: clean scaffold using the old version
- *Current branch*: your existing project with your custom code
- *Upgrade branch*: scaffold generated using the new version

Then, it creates a **merge branch** that combines everything.
You can review and test this branch before applying the changes.

<aside class="note warning">
<h1>Creates branches and deletes files</h1>

This command creates Git branches starting with `tmp-kb-update-` and deletes files during the process.
Make sure to commit your work before running it.

</aside>

## When to Use It?

Use this command when:

- You want to upgrade your project to a newer Kubebuilder version or plugin layout
- You prefer to automate the migration instead of updating files manually
- You want to review scaffold changes in a separate Git branch
- You want to focus only on fixing merge conflicts instead of re-applying all your code

## How It Works

The command performs the following steps:

1. Downloads the older CLI version (from the `PROJECT` file or `--from-version`)
2. Creates `tmp-kb-update-ancestor` with a clean scaffold using that version
3. Creates `tmp-kb-update-current` and restores your current code on top
4. Creates `tmp-kb-update-upgrade` using the latest scaffold
5. Created `tmp-kb-update-merge` which is a merge of the above branches using the 3-way merge strategy

You can push the `tmp-kb-update-merge` branch to your remote repository,
review the diff, and test the changes before merging into your main branch.

## How to Use It

Run the command from your project directory:

```sh
kubebuilder alpha update
```

If needed, set a specific version or branch:

```sh
kubebuilder alpha update \
  --from-version=v4.5.2 \
  --to-version=v4.6.0 \
  --from-branch=main
```

Force update even with merge conflicts:

```sh
kubebuilder alpha update --force
```

<aside class="note warning">
<h1>You might need to upgrade your project first</h1>

This command uses `kubebuilder alpha generate` internally.
As a result, the version of the CLI originally used to create your project must support `alpha generate`.

This command has only been tested with projects created using **v4.5.0** or later.
It might not work with projects that were initially created using a Kubebuilder version older than **v4.5.0**.

If your project was created with an older version, run `kubebuilder alpha generate` first to re-scaffold it.
Once updated, you can use `kubebuilder alpha update` for future upgrades.
</aside>

## Flags

| Flag                | Description                                                                                                                                                    |
|---------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `--from-version`    | **Required for projects initialized with versions earlier than v4.6.0.** Kubebuilder version your project was created with. If unset, uses the `PROJECT` file. |
| `--to-version`      | Version to upgrade to. Defaults to the latest version.                                                                                                         |
| `--from-branch`     | Git branch that contains your current project code. Defaults to `main`.                                                                                        |
| `--force`           | Force the update even if conflicts occur. Conflicted files will include conflict markers, and a commit will be created automatically. Ideal for automation (e.g., cronjobs, CI).                                                                       |
| `-h, --help`        | Show help for this command.                                                                                                                                    |
<aside class="note">
Projects generated with **Kubebuilder v4.6.0** or later include the `cliVersion` field in the `PROJECT` file.
This field is used by `kubebuilder alpha update` to determine the correct CLI
version for upgrading your project.
</aside>

## Merge Conflicts with `--force`

When you use the `--force` flag with `kubebuilder alpha update`, Git will complete the merge even if there are conflicts. The resulting commit will include conflict markers like:
```
<<<<<<< HEAD
Your changes
=======
Incoming changes
>>>>>>> branch-name
```
These conflicts will be committed in the
`tmp-kb-update-merge` branch.

<aside class="note warning">
You must manually resolve these conflicts before merging into your main branch.

```suggestion
<aside class="note warning">
<H1>If you face conflicts (using or not the --force flag) </H1>
If the merge introduces conflicts, you must resolve them and **ensure** you execute the following command to regenerate the manifests and organise the files properly:

```bash
make manifests generate fmt vet lint-fix
```

Alternatively, you may want to run:

```bash
make all
```
</aside>


## When to Use `--force`
Use `--force` only in scenarios like:
- Automated environments (e.g., CI pipelines or cron jobs)
- When you need to create a PR even if conflicts are present
- When a human will resolve the conflicts later
`kubebuilder alpha update --force`

This ensures the update proceeds without manual blocking but shifts responsibility for conflict resolution to a follow-up manual step.

This approach is typically used in automation workflows where conflict markers are later addressed by a human, or where preserving the conflicting changes is acceptable for follow-up processing.

## Requirements

- A valid [PROJECT][project-config] file at the root of your project
- A clean Git working directory (no uncommitted changes)
- Git must be installed and available

## Further Resources

- [WIP: Design proposal for update automation](https://github.com/kubernetes-sigs/kubebuilder/pull/4302)

[project-config]: ../../reference/project-config.md
