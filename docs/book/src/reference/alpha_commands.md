# Alpha Commands

Kubebuilder provides experimental **alpha commands** to assist with advanced operations such as
project migration and scaffold regeneration.

These commands are designed to simplify tasks that were previously manual and error-prone
by automating or partially automating the process.

<aside class="note warning">
<h1>Alpha commands are experimental</h1>

Alpha commands are under active development and may change or be removed in future releases.
They make local changes to your project and may delete files during execution.

Always ensure your work is committed or backed up before using them.
</aside>

The following alpha commands are currently available:

- [`alpha generate`](./../reference/commands/alpha_generate.md) — Re-scaffold the project using the installed CLI version
- [`alpha update`](./../reference/commands/alpha_update.md) — Automate the migration process via 3-way merge using scaffold snapshots

For more information, see each command's dedicated documentation.
