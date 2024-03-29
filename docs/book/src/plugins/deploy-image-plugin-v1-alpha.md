# Deploy Image Plugin (deploy-image/v1-alpha)

The deploy-image plugin allows users to create [controllers][controller-runtime] and custom resources which will deploy and manage an image on the cluster following
the guidelines and best practices. It abstracts the complexities to achieve this goal while allowing users to improve and customize their projects.

By using this plugin you will have:

- a controller implementation to Deploy and manage an Operand(image) on the cluster
- tests to check the reconciliation implemented using [ENVTEST][envtest]
- the custom resources samples updated with the specs used
- you will check that the Operand(image) will be added on the manager via environment variables

<aside class="note">
<h1>Examples</h1>

See the "project-v3-with-deploy-image" directory under the [testdata][testdata] directory of the Kubebuilder project to check an example of a scaffolding created using this plugin.

</aside>


## When to use it ?

- This plugin is helpful for those who are getting started.
- If you are looking to Deploy and Manage an image (Operand) using [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) and the tool the plugin will create an API/controller to be reconciled to achieve this goal
- If you are looking to speed up

## How to use it ?

After you create a new project with `kubebuilder init` you can create APIs using this plugin. Ensure that you have followed the [quick start](https://book.kubebuilder.io/quick-start.html) before trying to use it.

Then, by using this plugin you can [create APIs](https://book.kubebuilder.io/cronjob-tutorial/gvks.html) informing the image (Operand) that you would like to deploy on the cluster. Note that you can optionally specify the command that could be used to initialize this container via the flag `--image-container-command` and the port with `--image-container-port` flag. You can also specify the `RunAsUser` value for the Security Context of the container via the flag `--run-as-user`., i.e:

```sh
kubebuilder create api --group example.com --version v1alpha1 --kind Memcached --image=memcached:1.6.15-alpine --image-container-command="memcached,-m=64,modern,-v" --image-container-port="11211" --run-as-user="1001" --plugins="deploy-image/v1-alpha"
```

<aside class="warning">
<h1>Using make run</h1>

The `make run` will execute the `main.go` outside of the cluster to let you test the project running it locally. Note that by using this plugin the Operand image informed will be stored via an environment variable in the `config/manager/manager.yaml` manifest.

Therefore, before run `make run` you need to export any environment variable that you might have. Example:

```sh
export MEMCACHED_IMAGE="memcached:1.4.36-alpine"
```

</aside>

## Subcommands

The deploy-image plugin implements the following subcommands:

- create api (`$ kubebuilder create api [OPTIONS]`)

## Affected files

With the `create api` command of this plugin, in addition to the existing scaffolding, the following files are affected:

- `controllers/*_controller.go` (scaffold controller with reconciliation implemented)
- `controllers/*_controller_test.go` (scaffold the tests for the controller)
- `controllers/*_suite_test.go` (scaffold/update the suite of tests)
- `api/<version>/*_types.go` (scaffold the specs for the new api)
- `config/samples/*_.yaml` (scaffold default values for its CR)
- `main.go` (update to add controller setup)
- `config/manager/manager.yaml` (update with envvar to store the image)

## Further Resources:

- Check out [video to show how it works](https://youtu.be/UwPuRjjnMjY)
- See the [desing proposal documentation](../../../../designs/code-generate-image-plugin.md)

[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[testdata]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata/
[envtest]: ../reference/envtest.md
