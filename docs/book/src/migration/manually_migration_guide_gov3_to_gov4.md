# Migration from go/v3 to go/v4 by updating the files manually

Make sure you understand the [differences between Kubebuilder go/v3 and go/v4][v3vsv4]
before continuing.

Please ensure you have followed the [installation guide][quick-start]
to install the required components.

The following guide describes the manual steps required to upgrade your PROJECT config file to begin using `go/v4`.

This way is more complex, susceptible to errors, and success cannot be assured. Also, by following these steps you will not get the improvements and bug fixes in the default generated project files.

Usually it is suggested to do it manually if you have customized your project and deviated too much from the proposed scaffold. Before continuing, ensure that you understand the note about [project customizations][project-customizations]. Note that you might need to spend more effort to do this process manually than to organize your project customizations. The proposed layout will keep your project maintainable and upgradable with less effort in the future.

The recommended upgrade approach is to follow the [Migration Guide go/v3 to go/v4][migration-guide-gov3-to-gov4] instead.

## Migration from project config version "go/v3" to "go/v4"

Update the `PROJECT` file layout which stores information about the resources that are used to enable plugins make
useful decisions while scaffolding. The `layout` field indicates the scaffolding and the primary plugin version in use.

### Steps to migrate

#### Migrate the layout version into the PROJECT file

The following steps describe the manual changes required to bring the project configuration file (`PROJECT`).
These change will add the information that Kubebuilder would add when generating the file. This file can be found in the root directory.

Update the PROJECT file by replacing:

```yaml
layout:
- go.kubebuilder.io/v3
```

With:

```yaml
layout:
- go.kubebuilder.io/v4

```

#### Changes to the layout

##### New layout:

- The directory `apis` was renamed to `api` to follow the standard
- The `controller(s)` directory has been moved under a new directory called `internal` and renamed to singular as well `controller`
- The `main.go` previously scaffolded in the root directory has been moved under a new directory  called `cmd`

Therefore, you can check the changes in the layout results into:

```sh
...
├── cmd
│ └── main.go
├── internal
│ └── controller
└── api
```

##### Migrating to the new layout:

- Create a new directory `cmd` and move the `main.go` under it.
- If your project support multi-group the APIs are scaffold under a directory called `apis`. Rename this directory to `api`
- Move the `controllers` directory under the `internal` and rename it for `controller`
- Now ensure that the imports will be updated accordingly by:
  - Update the `main.go` imports to look for the new path of your controllers under the `internal/controller` directory

**Then, let's update the scaffolds paths**

- Update the Dockerfile to ensure that you will have:

```
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/controller/ internal/controller/
```

Then, replace:

```
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager main.go

```

With:

```
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go
```

- Update the Makefile targets to build and run the manager by replacing:

```
.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go
```

With:

```
.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go
```

- Update the `internal/controller/suite_test.go` to set the path for the `CRDDirectoryPaths`:

Replace:

```
CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
```

With:

```
CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
```

Note that if your project has multiple groups (`multigroup:true`) then the above update should result into `"..", "..", "..",` instead of `"..",".."`

#### Now, let's update the PATHs in the PROJECT file accordingly

The PROJECT tracks the paths of all APIs used in your project. Ensure that they now point to `api/...` as the following example:

**Before update:**
```
  group: crew
  kind: Captain
  path: sigs.k8s.io/kubebuilder/testdata/project-v4/apis/crew/v1
```

**After Update:**
```

  group: crew
  kind: Captain
  path: sigs.k8s.io/kubebuilder/testdata/project-v4/api/crew/v1
```

### Update kustomize manifests with the changes made so far

- Update the manifest under `config/` directory with all changes performed in the default scaffold done with `go/v4` plugin. (see for example `testdata/project-v4/config/`) to get all changes in the
  default scaffolds to be applied on your project
- Create `config/samples/kustomization.yaml` with all Custom Resources samples specified into `config/samples`. _(see for example `testdata/project-v4/config/samples/kustomization.yaml`)_

<aside class="warning">
<h1>`config/` directory with changes into the scaffold files</h1>

Note that under the `config/` directory you will find scaffolding changes since using
`go/v4` you will ensure that you are no longer using Kustomize v3x.

You can mainly compare the `config/` directory from the samples scaffolded under the `testdata`directory by
checking the differences between the `testdata/project-v3/config/` with `testdata/project-v4/config/` which
are samples created with the same commands with the only difference being versions.

However, note that if you create your project with Kubebuilder CLI 3.0.0, its scaffolds
might change to accommodate changes up to the latest releases using `go/v3` which are not considered
breaking for users and/or are forced by the changes introduced in the dependencies
used by the project such as [controller-runtime][controller-runtime] and [controller-tools][controller-tools].

</aside>

### If you have webhooks:

Replace the import `admissionv1beta1 "k8s.io/api/admission/v1beta1"` with `admissionv1 "k8s.io/api/admission/v1"` in the webhook test files

### Makefile updates

Update the Makefile with the changes which can be found in the samples under testdata for the release tag used. (see for example `testdata/project-v4/Makefile`)

### Update the dependencies

Update the `go.mod` with the changes which can be found in the samples under `testdata` for the release tag used. (see for example `testdata/project-v4/go.mod`). Then, run
`go mod tidy` to ensure that you get the latest dependencies and your Golang code has no breaking changes.

### Verification

In the steps above, you updated your project manually with the goal of ensuring that it follows
the changes in the layout introduced with the `go/v4` plugin that update the scaffolds.

There is no option to verify that you properly updated the `PROJECT` file of your project.
The best way to ensure that everything is updated correctly, would be to initialize a project using the `go/v4` plugin,
(ie) using `kubebuilder init --domain tutorial.kubebuilder.io plugins=go/v4` and generating the same API(s),
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

