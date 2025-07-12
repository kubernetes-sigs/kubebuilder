# Migrations

Upgrading your project scaffold to adopt the latest changes in Kubebuilder may involve migrating to a new plugin
version (e.g., `go.kubebuilder.io/v3` → `go.kubebuilder.io/v4`)
or newer CLI toolchain. This process often includes re-scaffolding and
manually merging your custom code.

This section details what’s required to migrate, between different versions of Kubebuilder scaffolding,
as well as to more complex project layout structures.

The manual approach can be error-prone. That is why Kubebuilder introduces new alpha commands
that help streamline the migration process.

## Manual Migration

The traditional process involves:

- Re-scaffolding the project using the latest Kubebuilder version or plugins
- Re-adding custom logic manually
- Running project generators:

  ```bash
  make generate
  make manifests
  ```

## Understanding the PROJECT File (Introduced in `v3.0.0`)

All inputs used by Kubebuilder are tracked in the [PROJECT][project-config] file.
If you use the CLI to generate your scaffolds, this file will record the project's configuration and metadata.

<aside class="note warning">
<h1>Project customizations</h1>

After using the CLI to create your project, you are free to customise how you see fit.
Bear in mind, that it is not recommended to deviate from the proposed layout unless you know what you are doing.

For example, you should refrain from moving the scaffolded files, doing so will make it difficult in
upgrading your project in the future. You may also lose the ability to use some of the CLI
features and helpers. For further information on the project layout, see
the doc [What's in a basic project?][basic-project-doc]

</aside>

## Alpha Migration Commands

Kubebuilder provides alpha commands to assist with project upgrades.

<aside class="note warning">
<h1>Automation process will involve deleting all files to regenerate</h1>
Deletes all files except `.git` and `PROJECT`.
</aside>

### `kubebuilder alpha generate`

Re-scaffolds the project using the installed CLI version.

```bash
kubebuilder alpha generate
```

### `kubebuilder alpha update` (available since `v4.7.0`)

Automates the migration by performing a 3-way merge:

- Original scaffold
- Your current customized version
- Latest or specified target scaffold

```bash
kubebuilder alpha update
```

For more details, see the [Alpha Command documentation](./reference/alpha_commands.md).


[project-config]: ./reference/project-config.md
[basic-project-doc]: ./cronjob-tutorial/basic-project.md