# Deploy Image Plugin (deploy-image/v1-alpha)

The deploy-image plugin allows users to create [controllers][controller-runtime] and custom resources which will deploy and manage an image on the cluster following
the guidelines and best practices. It abstracts the complexities to achieve this goal while allowing users to improve and customize their projects.

<aside class="note">
<h1>Examples</h1>

See the "project-v3-with-deploy-image" directory under the [testdata][testdata] directory of the Kubebuilder project to check an example of a scaffolding created using this plugin.

</aside>

## When to use it ?

- This plugin is helpful for beginners who are getting started with scaffolding and using operators.
- If you are looking to Deploy and Manage an image (Operand) using [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) and the tool the plugin will create an API/controller to be reconciled to achieve this goal

## How to use it ?

After you create a new project with `kubebuilder init` you can create APIs using this plugin. Ensure that you have followed the [quick start](https://book.kubebuilder.io/quick-start.html) before trying to use it.

Then, by using this plugin you can [create APIs](https://book.kubebuilder.io/cronjob-tutorial/gvks.html) informing the image (Operand) that you would like to deploy on the cluster. Note that you can optionally specify the command that could be used to initialize this container via the flag `--image-container-command` and the port with `--image-container-port` flag. You can also specify the `RunAsUser` value for the Security Context of the container via the flag `--run-as-user`., i.e:

```sh
kubebuilder create api --group example.com --version v1alpha1 --kind Memcached --image=memcached:1.6.15-alpine --image-container-command="memcached,-m=64,modern,-v" --image-container-port="11211" --run-as-user="1001" --plugins="deploy-image/v1-alpha"
```

## Subcommands

The deploy-image plugin implements the following subcommands:

- create api (`$ kubebuilder create api [OPTIONS]`)

## Affected files

With the `create api` command of this plugin, in addition to the existing scaffolding, the following files are affected:

**When multigroup is not enabled**

- `controllers/*_controller.go`
- `controllers/*_suite_test.go`
- `api/<version>/*_types.go` (scaffold the specs for the new api)
- `config/samples/*_.yaml` (scaffold default values for its CR)

**When multigroup is enabled**

- `controllers/<group>/*_controller.go`
- `controllers/<group>/*_suite_test.go`
- `apis/<group>/<version>/*_types.go` (scaffold the specs for the new api)
- `config/samples/*_.yaml` (scaffold default values for its CR)

[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[testdata]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata/
