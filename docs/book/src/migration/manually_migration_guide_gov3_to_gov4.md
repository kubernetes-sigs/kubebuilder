# Migration from go/v3 to go/v4-alpha by updating the files manually

Make sure you understand the [differences between Kubebuilder go/v3 and go/v4-alpha][v3vsv4]
before continuing.

Please ensure you have followed the [installation guide][quick-start]
to install the required components.

The following guide describes the manual steps required to upgrade your PROJECT config file to begin using `go/v4-alpha`.

This way is more complex, susceptible to errors, and success cannot be assured. Also, by following these steps you will not get the improvements and bug fixes in the default generated project files.

Usually it is suggested to do it manually if you have customized your project and deviated too much from the proposed scaffold. Before continuing, ensure that you understand the note about [project customizations][project-customizations]. Note that you might need to spend more effort to do this process manually than to organize your project customizations. The proposed layout will keep your project maintainable and upgradable with less effort in the future.

The recommended upgrade approach is to follow the [Migration Guide go/v3 to go/v4-alpha][migration-guide-gov3-to-gov4] instead.

## Migration from project config version "go/v3" to "go/v4"

Update `PROJECT` file layout which stores the information about the resources are use to enable plugins to make useful decisions when scaffolding.

Furthermore, the `PROJECT` file itself is now versioned. The `version` field corresponds to the version of the `PROJECT` file itself, while the `layout` field indicates the scaffolding and the primary plugin version in use.

Update: 

```yaml
layout:
- go.kubebuilder.io/v3
```

With:

```yaml
layout:
- go.kubebuilder.io/v4-alpha

```

### Steps to migrate

- Update the `main.go` with the changes which can be found in the samples under testdata for the release tag used. (see for example `testdata/project-v4/main.go`).
- Update the Makefile with the changes which can be found in the samples under testdata for the release tag used. (see for example `testdata/project-v4/Makefile`)
- Update the `go.mod` with the changes which can be found in the samples under `testdata` for the release tag used. (see for example `testdata/project-v4/go.mod`). Then, run
`go mod tidy` to ensure that you get the latest dependencies and your Golang code has no breaking changes.
- Update the manifest under `config/` directory with all changes performed in the default scaffold done with `go/v4-alpha` plugin. (see for example `testdata/project-v4/config/`) to get all changes in the 
default scaffolds to be applied on your project
- Replace the import `admissionv1beta1 "k8s.io/api/admission/v1beta1"` with `admissionv1 "k8s.io/api/admission/v1"` in the webhook test files

<aside class="warning">
<h1>`config/` directory with changes into the scaffold files</h1>

Note that under the `config/` directory you will find scaffolding changes since using
`go/v4-alpha` you will ensure that you are no longer using Kustomize v3x.

You can mainly compare the `config/` directory from the samples scaffold under the `testdata`directory by 
checking the differences between the `testdata/project-v3/config/` with `testdata/project-v4/config/` which
are samples created with the same commands with the only difference being versions.

However, note that if you create your project with Kubebuilder CLI 3.0.0, its scaffolds
might change to accommodate changes up to the latest releases using `go/v3` which are not considered 
breaking for users and/or are forced by the changes introduced in the dependencies 
used by the project such as [controller-runtime][controller-runtime] and [controller-tools][controller-tools]. 

</aside>

### Verification

In the steps above, you updated your project manually with the goal of ensuring that it follows
the changes in the layout introduced with the `go/v4-alpha` plugin that update the scaffolds.

There is no option to verify that you properly updated the `PROJECT` file of your project. 
The best way to ensure that everything is updated correctly, would be to initialize a project using the `go/v4-alpha` plugin,
(ie) using `kubebuilder init --domain tutorial.kubebuilder.io plugins=go/v4-alpha` and generating the same API(s),
controller(s), and webhook(s) in order to compare the generated configuration with the manually changed configuration.

Also, after all updates you would run the following commands:

- `make manifests` (to re-generate the files using the latest version of the contrller-gen after you update the Makefile)
- `make all` (to ensure that you are able to build and perform all operations)

[v3vsv4]: v3vsv4.md
[quick-start]: ./../quick-start.md#installation
[migration-guide-gov3-to-gov4]: migration_guide_gov3_to_gov4.md
[controller-tools]: https://github.com/kubernetes-sigs/controller-tools/releases
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime/releases
[multi-group]: multi-group.md

