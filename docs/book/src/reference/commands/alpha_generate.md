# Regenerate your project with (`alpha generate`)

## Overview

The `kubebuilder alpha generate` command re-scaffolds your project using the currently installed
CLI and plugin versions.

It regenerates the full scaffold based on the configuration specified in your [PROJECT][project-config] file.
This allows you to apply the latest layout changes, plugin features, and code generation improvements introduced
in newer Kubebuilder releases.

You may choose to re-scaffold the project in-place (overwriting existing files) or in a separate
directory for diff-based inspection and manual integration.

<aside class="warning">
    <h3>Deletes files during scaffold regeneration</h3>
When executed in-place, this command deletes all files except `.git` and `PROJECT`.

Always back up your project or use version control before running this command.
</aside>

## When to Use It?

You can use `kubebuilder alpha generate` to upgrade your project scaffold when new changes are introduced
in Kubebuilder. This includes updates to plugins (for example, `go.kubebuilder.io/v3` → `go.kubebuilder.io/v4`)
or the CLI releases (for example, 4.3.1 → latest) .

This command is helpful when you want to:

- Update your project to use the latest layout or plugin version
- Regenerate your project scaffold to include recent changes
- Compare the current scaffold with the latest and apply updates manually
- Create a clean scaffold for reviewing or testing changes

Use this command when you want full control of the upgrade process.
It is also useful if your project was created with an older CLI version and does not support `alpha update`.

This approach allows you to compare changes between your current branch and upstream
scaffold updates (e.g., from the main branch), and helps you overlay custom code atop the new scaffold.

<aside class="note tip">
<h4>Looking for a more automated migration?</h4>

If you want to upgrade your project scaffold with less manual work,
try [`kubebuilder alpha update`](./alpha_update.md).

It uses a 3-way merge to keep your code and apply the latest scaffold changes automatically.
Use `alpha generate` if `alpha update` is not available for your project yet
or if you prefer to handle changes manually.

</aside>

## How to Use It?

### Upgrade your current project to CLI version installed (i.e. latest scaffold)

```sh
kubebuilder alpha generate
```

After running this command, your project will be re-scaffolded in place.
You can then compare the local changes with your main branch to see what was updated,
and re-apply your custom code on top as needed.

### Generate Scaffold to a New Directory

Use the `--input-dir` and `--output-dir` flags to specify input and output paths.

```sh
kubebuilder alpha generate \
  --input-dir=/path/to/existing/project \
  --output-dir=/path/to/new/project
```

After running the command, you can inspect the generated scaffold in the specified output directory.

### Flags

| Flag            | Description                                                                 |
|------------------|-----------------------------------------------------------------------------|
| `--input-dir`    | Path to the directory containing the `PROJECT` file. Defaults to CWD. Deletes all files except `.git` and `PROJECT`. |
| `--output-dir`   | Directory where the new scaffold will be written. If unset, re-scaffolds in-place. |
| `--plugins`      | Plugin keys to use for this generation.                                     |
| `-h, --help`     | Show help for this command.                                                 |


## Further Resources

- [Video demo on how it works](https://youtu.be/7997RIbx8kw?si=ODYMud5lLycz7osp)
- [Design proposal documentation](../../../../../designs/helper_to_upgrade_projects_by_rescaffolding.md)

[example]: ../../../../../testdata/project-v4-with-plugins/PROJECT
[project-config]: ../../reference/project-config.md