# Multi Module Plugin (multi-module/v1-alpha)

The multi-module plugin allows users to modify the existing scaffolding setup of kubebuilder to enable [Monorepo][monorepo] support. It abstracts the complexities to achieve this goal by easing the most difficult part of getting into Monorepos, namely the setup and the scaffolding required.

By using this plugin you will have:

- Separate go.mod modules in your API definition and your controller
- an adjusted and optimized Image Building in your `Dockerfile` for dealing with Monorepos
- an updated main module that has correct `replace` [directives][replace-directives] in place for developing locally.
- Ability to switch from Monorepo to Classic Repository structure at will.

<aside class="note">
<h1>Examples</h1>

See the "project-v3-multimodule" directory under the [testdata][testdata] directory of the Kubebuilder project to check an example of a scaffolding created using this plugin.

</aside>

## When to use it ?

- If you want to reduce impact of transitive dependencies on your API being included in other projects
- If you are looking to separately manage the lifecycle of your API release process from your controller release process.
- If you are looking to modularize your codebase without splitting your code between multiple repositories.

## How to use it ?

After you create a new project with `kubebuilder init --plugins multi-module` you can create modularized APIs using this plugin. Ensure that you have followed the [quick start](https://book.kubebuilder.io/quick-start.html) before trying to use it.

Then, by using this plugin you automatically create a `go.mod` module in your `api` or `apis` directory (depending on wether you also enabled multi-group support)

```sh
kubebuilder create api --group example.com --version v1alpha1 --kind MyKind
```

<aside class="warning">
<h1>Take care of your replace directives</h1>

The `make run` will execute the `main.go` outside of the cluster to let you test the project running it locally. Note that during scaffolding, the plugin will ensure you have a `replace` directive scaffolded in your main go.mod file. This is to ensure that the module resolution works with mono-repo support.

Therefore, before releasing your first version of your operator, make sure to [familiarize yourself with mono-repo/multi-module releases][multi-module-repositories] with multiple `go.mod` files in different subdirectories.

Assuming a single API was created, the release process could look like this:

```sh
git commit
git tag v1.0.0 # this is your main module release
git tag api/v1.0.0 # this is your api release
go mod edit -require github.com/my-repo@v1.0.0 # now we depend on the api module in the main module
go mod edit -dropreplace github.com/my-repo/api # this will drop the replace directive for local development, meaning the sources from the VCS will be used instead of the ones in your monorepo checked out locally.
git push origin main v1.0.0 api/v1.0.0
```

</aside>

## Subcommands

The deploy-image plugin implements the following subcommands:

- init (`$ kubebuilder init [OPTIONS]`)
- create api (`$ kubebuilder create api [OPTIONS]`)
- create webhook (`$ kubebuilder create webhook [OPTIONS]`)
- edit (`$ kubebuilder edit [OPTIONS]`)

## Affected files

With the `create api` command of this plugin, in addition to the existing scaffolding, the following files are affected:

- `controllers/*_controller.go` (imports api from new module)
- `controllers/*_controller_test.go` (imports api from new module)
- `controllers/*_suite_test.go` (imports api from the new module)
- `api/go.mod` (scaffolds the module for the API)
- `go.mod` (requires and issues replace statements for the API)
- `Dockerfile` (adds additional layers for submodule dependencies in `api`/`apis`)

[monorepo]: https://en.wikipedia.org/wiki/Monorepo
[replace-directives]: https://go.dev/ref/mod#go-mod-file-replace
[multi-module-repositories]: https://github.com/golang/go/wiki/Modules#faqs--multi-module-repositories
