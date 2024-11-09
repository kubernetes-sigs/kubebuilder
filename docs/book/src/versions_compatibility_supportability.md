# Versions Compatibility and Supportability

Projects created by Kubebuilder contain a `Makefile` that installs tools at versions defined during project creation.
The main tools included are:

- [kustomize](https://github.com/kubernetes-sigs/kustomize)
- [controller-gen](https://github.com/kubernetes-sigs/controller-tools)
- [setup-envtest](https://github.com/kubernetes-sigs/controller-runtime/tree/main/tools/setup-envtest)

Additionally, these projects include a `go.mod` file specifying dependency versions.
Kubebuilder relies on [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) and its Go and Kubernetes dependencies.
Therefore, the versions defined in the `Makefile` and `go.mod` files are the ones that have been tested, supported, and recommended.

Each minor version of Kubebuilder is tested with a specific minor version of client-go.
While a Kubebuilder minor version *may* be compatible with other client-go minor versions,
or other tools this compatibility is not guaranteed, supported, or tested.

The minimum Go version required by Kubebuilder is determined by the highest minimum
Go version required by its dependencies. This is usually aligned with the minimum
Go version required by the corresponding `k8s.io/*` dependencies.

Compatible `k8s.io/*` versions, client-go versions, and minimum Go versions can be found in the `go.mod`
file scaffolded for each project for each [tag release](https://github.com/kubernetes-sigs/kubebuilder/tags).

**Example:** For the `4.1.1` release, the minimum Go version compatibility is `1.22`.
You can refer to the samples in the testdata directory of the tag released [v4.1.1](https://github.com/kubernetes-sigs/kubebuilder/tree/v4.1.1/testdata),
such as the [go.mod](https://github.com/kubernetes-sigs/kubebuilder/blob/v4.1.1/testdata/project-v4/go.mod#L3) file for `project-v4`. You can also check the tools versions supported and
tested for this release by examining the [Makefile](https://github.com/kubernetes-sigs/kubebuilder/blob/v4.1.1/testdata/project-v4/Makefile#L160-L165).

## Operating Systems Supported

Currently, Kubebuilder officially supports macOS and Linux platforms. If you are using a Windows OS, you may encounter issues.
Contributions towards supporting Windows are welcome

<aside class="note warning">
<h1>Project customizations</h1>

After using the CLI to create your project, you are free to customize how
you see fit. Bear in mind, that it is not recommended to deviate from
the proposed layout unless you know what you are doing.

For example, you should refrain from moving the scaffolded files,
doing so will make it difficult in upgrading your project in the future.
You may also lose the ability to use some of the CLI features and helpers.
For further information on the project layout, see the doc [What's in a basic project?][basic-project-doc]

</aside>

[basic-project-doc]: ./reference/project-config.md