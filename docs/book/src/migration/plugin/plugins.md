# Plugin Migrations

Migrating between project [plugins][plugins-doc] involves additions, removals, and/or changes
to files created by any plugin-supported command, ex. `init` and `create`. A plugin supports
one or more project config versions; make sure you [upgrade][project-migration] your project's
config version to the latest supported by your target plugin version before upgrading plugin versions.

[plugins-doc]:/reference/cli-plugins.md
[project-migration]:/migration/projects.md
