# Deploy Image Plugin (deploy-image/v1-alpha)

The `deploy-image` plugin allows users to create [controllers][controller-runtime] and custom resources that deploy and manage container images on the cluster, following Kubernetes best practices. It simplifies the complexities of deploying images while allowing users to customize their projects as needed.

By using this plugin, you will get:

- A controller implementation to deploy and manage an Operand (image) on the cluster.
- Tests to verify the reconciliation logic, using [ENVTEST][envtest].
- Custom resource samples updated with the necessary specifications.
- Environment variable support for managing the Operand (image) within the manager.

<aside class="note">
<h1>Examples</h1>

See the `project-v4-with-plugins` directory under the [testdata][testdata]
directory in the Kubebuilder project to check an example
of scaffolding created using this plugin.

The `Memcached` API and its controller was scaffolded
using the command:

```shell
kubebuilder create api \
  --group example.com \
  --version v1alpha1 \
  --kind Memcached \
  --image=memcached:memcached:1.6.26-alpine3.19 \
  --image-container-command="memcached,--memory-limit=64,-o,modern,-v" \
  --image-container-port="11211" \
  --run-as-user="1001" \
  --plugins="deploy-image/v1-alpha"
```

The `Busybox` API was created with:

```shell
kubebuilder create api \
  --group example.com \
  --version v1alpha1 \
  --kind Busybox \
  --image=busybox:1.36.1 \
  --plugins="deploy-image/v1-alpha"
```
</aside>


## When to use it?

- This plugin is ideal for users who are just getting started with Kubernetes operators.
- It helps users deploy and manage an image (Operand) using the [Operator pattern][operator-pattern].
- If you're looking for a quick and efficient way to set up a custom controller and manage a container image, this plugin is a great choice.

## How to use it?

1. **Initialize your project**:
   After creating a new project with `kubebuilder init`, you can use this
   plugin to create APIs. Ensure that you've completed the
   [quick start][quick-start] guide before proceeding.

2. **Create APIs**:
   With this plugin, you can [create APIs][create-apis] to specify the image (Operand) you want to deploy on the cluster. You can also optionally specify the command, port, and security context using various flags:

   Example command:
   ```sh
   kubebuilder create api --group example.com --version v1alpha1 --kind Memcached --image=memcached:1.6.15-alpine --image-container-command="memcached,--memory-limit=64,modern,-v" --image-container-port="11211" --run-as-user="1001" --plugins="deploy-image/v1-alpha"
   ```

<aside class="warning">
<h1>Note on make run:</h1>

When running the project locally with `make run`, the Operand image
provided will be stored as an environment variable in the
`config/manager/manager.yaml` file.

Ensure you export the environment variable before running the project locally, such as:

```shell
export MEMCACHED_IMAGE="memcached:1.4.36-alpine"
```

</aside>

## Subcommands

The `deploy-image` plugin includes the following subcommand:

- `create api`: Use this command to scaffold the API and controller code to manage the container image.

## Affected files

When using the `create api` command with this plugin, the following
files are affected, in addition to the existing Kubebuilder scaffolding:

- `controllers/*_controller_test.go`: Scaffolds tests for the controller.
- `controllers/*_suite_test.go`: Scaffolds or updates the test suite.
- `api/<version>/*_types.go`: Scaffolds the API specs.
- `config/samples/*_.yaml`: Scaffolds default values for the custom resource.
- `main.go`: Updates the file to add the controller setup.
- `config/manager/manager.yaml`: Updates to include environment variables for storing the image.

## Further Resources:

- Check out this [video][video] to see how it works.

[video]: https://youtu.be/UwPuRjjnMjY
[operator-pattern]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[testdata]: ./.././../../../../testdata/project-v4-with-plugins
[envtest]: ./../../reference/envtest.md
[quick-start]: ./../../quick-start.md
[create-apis]: ../../cronjob-tutorial/new-api.md