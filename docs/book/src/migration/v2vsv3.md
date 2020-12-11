# Kubebuilder v2 vs v3

This document covers all breaking changes when migrating from v2 to v3.

The details of all changes (breaking or otherwise) can be found in [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder/releases) release notes.

## Kubebuilder

- A plugin design was introduced to the project. For more info see the [Extensible CLI and Scaffolding Plugins][plugins-phase1-design-doc]
- The GO supported version was upgraded from 1.13+ to 1.15+

## Project config versions

- [v3][project-v3]

## `go.kubebuilder.io` plugin versions

- [v3][plugin-v3]

[plugins-phase1-design-doc]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-1.md
[project-v3]:/migration/project/v2_v3.md
[plugin-v3]:/migration/plugin/v2_v3.md
