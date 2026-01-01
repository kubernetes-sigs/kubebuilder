# Migrations

Upgrading your Kubebuilder project to the latest version ensures you benefit from new features,
bug fixes, and ecosystem improvements. It is recommended to keep your project aligned with ecosystem changes.

Migration may involve updating to a newer plugin version (e.g., from `go.kubebuilder.io/v3` in release 3.x to `go.kubebuilder.io/v4` in release 4.x) or updating the scaffold produced by the same plugin across CLI releases (e.g., from `v4.9.0` to `v4.10.1`).

Kubebuilder provides multiple migration paths to suit your workflow. Choose the approach that best fits your needs.

<aside class="note">
<h1>Understanding the PROJECT File</h1>

From Kubebuilder `v3.0.0` onwards, all inputs used by Kubebuilder are tracked in the [PROJECT][project-config] file.
If you use the CLI to generate your scaffolds, this file will record the project's configuration and metadata,
enabling all automation tools to work effectively.

It is recommended to use the CLI to scaffold all resources (`kubebuilder create api`, `kubebuilder create webhook`, etc.)
whenever possible, including controllers and webhooks for external types. The CLI has been continuously improved
over time to address various options and needs. This ensures all resources are tracked in the PROJECT file,
which automation tools (alpha update, alpha generate, autoupdate plugin) depend on.

</aside>

<aside class="warning">
<h1>Project Customizations</h1>

After using the CLI to create your project, you are free to customize the business logic and add features as you see fit.
However, it is not recommended to deviate from the proposed project layout unless you know what you are doing.

For example, you should refrain from moving the scaffolded files, as doing so may will make it difficult to upgrade
your project in the future. You may also lose the ability to use some of the CLI features and helpers.

Projects that do not use the CLI to generate scaffolds, or that deviate heavily from the proposed layout,
may need to use the manual migration process, as automated migration tools might not work properly while
the [alpha update](./reference/commands/alpha_update.md) and [AutoUpdate Plugin][autoupdate-v1-alpha]
are designed to do a 3-way merge to keep your customizations intact.

For further information on the project layout, see [What's in a basic project?][basic-project-doc]

</aside>

## Migration Options

### Automated Updates via GitHub Actions

The [AutoUpdate Plugin][autoupdate-v1-alpha] scaffolds an action that automatically monitors for new Kubebuilder releases and
opens a GitHub Issue with a Pull Request compare link when updates are available. This is ideal for
keeping your project up to date with minimal manual work.

This plugin provides a mechanism similar to Dependabot for GitHub, offering continuous updates with AI assistance
for projects that follow the standard scaffold.

```bash
kubebuilder edit --plugins="autoupdate/v1-alpha"
```

<aside class="note">
<h1>Requirements and Limitations</h1>

- Requires GitHub repository (GitHub Actions workflow)
- Requires branch protection rules for safety (recommended)
- Needs the same requirements as `alpha update` (see below)

</aside>

See the [AutoUpdate Plugin documentation][autoupdate-v1-alpha] for complete details.

### Using Alpha Update Locally

If you prefer to run updates locally instead of relying on GitHub Actions, you can use the same logic
as the [AutoUpdate Plugin][autoupdate-v1-alpha] directly from your command line.

```shell
kubebuilder alpha update
```

This command uses the same underlying mechanism as the AutoUpdate Plugin. You can migrate your project,
resolve any conflicts if needed, and then push a Pull Request from your local environment.

<aside class="note">
<h1>Requirements and Limitations</h1>

- Requires projects created with Kubebuilder **`v4.5.0`** or later
- For projects created before `v4.6.0`: the CLI version is not tracked in the `PROJECT` file, so you may need to use `alpha generate` first to establish a baseline
- For projects created with `v4.6.0`+: includes `cliVersion` in the `PROJECT` file for automatic version detection

</aside>

See the [`alpha update` command reference](./reference/commands/alpha_update.md) for all options and flags.

### Regenerate with Help and Merge Manually

The `kubebuilder alpha generate` command re-scaffolds your entire project based on your `PROJECT` file
configuration. You can then manually compare and merge your custom code. For example, you can use it to
regenerate your project after upgrading the Kubebuilder CLI version and then, manually use an IDE or
`git diff` to compare and merge changes by hand into your existing codebase to ensure that all your changes
are applied in a new scaffold.

This approach is useful for projects that heavily customize the scaffold or
when other migration methods aren't available. You might need to use this method only once to
establish a baseline for future automated updates.

```shell
kubebuilder alpha generate
```

<aside class="note">
<h1>Requirements and Limitations</h1>

- Requires a `PROJECT` file (projects created with Kubebuilder **v3.0.0** or later)
- Only re-scaffolds resources that were created using the CLI and tracked in the `PROJECT` file
- Manually created APIs, controllers, or webhooks will not be regenerated
- This may result in a partial re-scaffold if you have manually created resources
- Requires manual comparison and merge of custom code after regeneration

</aside>

See the [`alpha generate` command reference](./reference/commands/alpha_generate.md) for details.

### Fully Manual Migration

For complete control, you can manually migrate by creating a new project with the latest Kubebuilder
version and porting your code over.

In this process, you will run all commands from scratch to create a new project, APIs, controllers,
webhooks, and other resources. Then, manually copy your business logic and customizations from your old project to the new one.

To streamline this one-time migration, [AI Migration Helpers](./migration/ai-helpers.md) have been added to automate repetitive tasks.

<aside class="note">
<h1>When to Use Manual Migration</h1>

Use this approach when:
- Your project was created with Kubebuilder versions **before `v3.0.0`** (no `PROJECT` file)
- You have heavily customized the scaffold beyond standard patterns
- You have manually created APIs, controllers, or webhooks not tracked by the CLI
- You want complete control and visibility into every change
- Other automated methods are not available for your project version

</aside>

See the [Manual Migration Process Guide](./migration/manual-process.md) for a complete step-by-step walkthrough with AI helpers.

[project-config]: ./reference/project-config.md
[basic-project-doc]: ./cronjob-tutorial/basic-project.md
[autoupdate-v1-alpha]: ./plugins/available/autoupdate-v1-alpha.md