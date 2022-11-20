# Monitoring Plugin (`monitoring/v1-alpha`)

The Monitoring plugin is an optional plugin that can be used to scaffold Prometheus based monitoring, will provide best practices and tooling for monitoring requirements and help with standardizing the way monitoring is implemented in operators.

<aside class="note">
<h1>Examples</h1>

You can check its default scaffold by looking at the `project-v4-with-monitoring` or `project-v3-with-monitoring` projects under the [testdata][testdata] directory on the root directory of the Kubebuilder project.

</aside>

## When to use it ?

- If you are looking to implement Prometheus based monitoring to your operator and looking for a scaffold that will help you with the initial onboarding and will provide you with best practices and tooling.

## How to use it ?

### Prerequisites:

- Access to [Prometheus][prometheus].

### Basic Usage

The monitoring plugin is attached to the `init` subcommand and the `edit` subcommand:

```sh
# Initialize a new project with monitoring plugin
kubebuilder init --plugins="go/v4-alpha,monitoring/v1-alpha"
# or
kubebuilder init --plugins="go/v3,monitoring/v1-alpha"

# Enable monitoring plugin to an existing project
kubebuilder edit --plugins="monitoring/v1-alpha"
```

## Affected files

The following scaffolds will be created or updated by this plugin:

- `monitoring/`
- `main.go`
- `MakeFile`

[prometheus]: https://prometheus.io/docs/introduction/overview/
[testdata]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata
