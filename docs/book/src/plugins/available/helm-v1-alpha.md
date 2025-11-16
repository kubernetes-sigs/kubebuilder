# Helm Plugin (`helm/v1-alpha`) - **DEPRECATED**

<aside class="warning">
<h1> Deprecated Plugin</h1>

The `helm/v1-alpha` plugin is **deprecated**. Please use [`helm/v2-alpha`](./helm-v2-alpha.md) instead.

The v2-alpha version provides:
- Dynamic Helm chart generation from kustomize output
- Better preservation of customizations (env vars, labels, annotations)
- Organized template structure matching your config/ directory
- More flexible configuration options

</aside>

The Helm plugin is an optional plugin that can be used to scaffold a Helm chart, allowing you to distribute the project using Helm.

By default, users can generate a bundle with all the manifests by running the following command:

```bash
make build-installer IMG=<some-registry>/<project-name:tag>
```

This allows the project consumer to install the solution by applying the bundle with:

```bash
kubectl apply -f https://raw.githubusercontent.com/<org>/project-v4/<tag or branch>/dist/install.yaml
```

However, in many scenarios, you might prefer to provide a Helm chart to package your solution.
If so, you can use this plugin to generate the Helm chart under the `dist` directory.

<aside class="note">
<h1>Examples</h1>

You can check the plugin usage by looking at `project-v4-with-plugins` samples
under the [testdata][testdata] directory on the root directory of the Kubebuilder project.

</aside>

## When to use it

- If you want to provide a Helm chart for users to install and manage your project.
- If you need to update the Helm chart generated under `dist/chart/` with the latest project changes:
  - After generating new manifests, use the `edit` option to sync the Helm chart.
  - **IMPORTANT:** If you have created a webhook or an API using the [DeployImage][deployImage-plugin] plugin,
  you must run the `edit` command with the `--force` flag to regenerate the Helm chart values based
  on the latest manifests (_after running `make manifests`_) to ensure that the HelmChart values are
  updated accordingly. In this case, if you have customized the files
  under `dist/chart/values.yaml`, and the `templates/manager/manager.yaml`, you will need to manually reapply your customizations on top
  of the latest changes after regenerating the Helm chart.

<aside class="note">
<H1> Why CRDs are added under templates? </H1>

Although [Helm best practices](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/#method-1-let-helm-do-it-for-you) recommend placing CRDs under a top-level `crds/` directory, the Kubebuilder Helm plugin intentionally places them under `templates/crd`.

The rationale is tied to how Helm itself handles CRDs.
By default, Helm will install CRDs once during the initial release,
but it will **ignore CRD changes** on subsequent upgrades.

This can lead to surprising behavior where chart upgrades silently
skip CRD updates, leaving clusters out of sync.

To avoid endorsing this behavior, the Kubebuilder plugin follows the approach of packaging
CRDs inside `templates/`. In this mode, Helm treats CRDs like
any other resource, ensuring they are applied and upgraded as expected.
While this prevents mixing CRDs and CRs of the same type in a single chart (since Helm cannot wait between creation steps), it ensures predictable and explicit lifecycle management of CRDs.

In short:
- **Helm `crds/` directory** → one-time install only, no upgrades.
- **Kubebuilder `templates/crd`** → CRDs managed like other manifests, upgrades included.

This design choice prioritizes correctness and maintainability over Helm's default convention,
while leaving room for future improvements (such as scaffolding separate charts for APIs and controllers).
</aside>

## How to use it ?

### Basic Usage

The Helm plugin is attached to the `edit` subcommand as the `helm/v1-alpha` plugin
relies on the Go project being scaffolded first.

```sh

# Initialize a new project
kubebuilder init

# Enable or Update the helm chart via the helm plugin to an existing project
# Before run the edit command, run `make manifests` to generate the manifest under `config/`
make manifests
kubebuilder edit --plugins=helm/v1-alpha
```
<aside class="note">
  <h1>Use the edit command to update the Helm Chart with the latest changes</h1>

  After making changes to your project, ensure that you run `make manifests` and then
  use the command `kubebuilder edit --plugins=helm/v1-alpha` to update the Helm Chart.

  Note that the following files will **not** be updated unless you use the `--force` flag:

  <pre>
  dist/chart/
  ├── values.yaml
  └── templates/
      └── manager/
          └── manager.yaml
  </pre>

  The files `chart/Chart.yaml`, `chart/templates/_helpers.tpl`, and `chart/.helmignore` are never updated
  after their initial creation unless you remove them.

</aside>

## Subcommands

The Helm plugin implements the following subcommands:

- edit (`$ kubebuilder edit [OPTIONS]`)

## Affected files

The following scaffolds will be created or updated by this plugin:

- `dist/chart/*`

[testdata]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata/project-v4-with-plugins
[deployImage-plugin]: ./deploy-image-plugin-v1-alpha.md