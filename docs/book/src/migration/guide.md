# Migration from v2 to v3

Make sure you understand the [differences between Kubebuilder v2 and v3](./v2vsv3.md)
before continuing

Please ensure you have followed the [installation guide](/quick-start.md#installation)
to install the required components.

Migration to this new project version is fairly straightforward, requiring only a few
updates to your `PROJECT`, YAML manifest, and Go source files.

## `PROJECT` file changes

Your `PROJECT` file will look something like the following for version "2":

```yaml
domain: tutorial.kubebuilder.io
repo: tutorial.kubebuilder.io/project
resources:
- group: batch
  kind: CronJob
  version: v1
version: "2"
```

The "3-alpha" project version has two new fields relevant to your project,
`projectName` and `layout`, which you should set to the following values:
- `projectName: <base of your repo field>`
- `layout: go.kubebuilder.io/v2`
  - This value comes from kubebuilder's [Go plugin][go-plugin].

Your `PROJECT` file should now look something like this:

```yaml
domain: tutorial.kubebuilder.io
layout: go.kubebuilder.io/v2
projectName: project
repo: tutorial.kubebuilder.io/project
resources:
- group: batch
  kind: CronJob
  version: v1
version: 3-alpha
```

## `config/` manifest changes

TODO

## Go source changes

TODO

## Verification

Finally, we can run `make` and `make docker-build` to ensure things are working fine.

[go-plugin]:https://pkg.go.dev/sigs.k8s.io/kubebuilder/pkg/plugin/v2?tab=doc
