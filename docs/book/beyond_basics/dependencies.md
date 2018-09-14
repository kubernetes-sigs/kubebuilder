# Dependency Management

Kubebuilder uses [dep](https://golang.github.io/dep) to manage dependencies.
Different dependency management tasks can be done using the `dep ensure`
command.

## Adding new dependencies

{% panel style="warning", title="Kubernetes Dependencies" %}

Kubebuilder-generated projects depends on a number of Kubernetes
dependencies internally. Kubebuilder (using the controller-runtime
library) makes sure that the parts of these dependencies that are exposed
in the Kubebuilder API remain stable.

It's recommended not to make use of most of these libraries directly, since
they change frequently in incompatible ways.  The `k8s.io/api` repository is
the exception to this, and it's reccomended that you rely on the version that
`kubebuilder` requires, instead of listing it as a direct dependency in
`Gopkg.toml`.

However, if you do add direct dependencies on any of these libraries yourself,
be aware that you may encounter dependency conflicts. See [the problem with
kubernetes libraries](#the-problem-with-kubernetes-libraries) below for more
information.

{% endpanel %}

{% method %}

Dep manages dependency constraints using the `Gopkg.toml` file.  You can add
new dependencies by adding new `[[constraint]]` stanzas to that file.
Alternatively, if you're [not using `kubebuilder
update`](./upgrading_kubebuilder.md#by-hand), you can use the `dep ensure -add`
command to add new dependencies to your `Gopkg.toml`.

{% sample lang="bash" %}
```bash
# edit Gopkg.toml OR perform the following:
dep ensure -add github.com/pkg/errors
```
{% endmethod %}

## Updating existing dependencies

{% method %}

Update dependencies for your project to the latest minor and patch versions.

{% sample lang="bash" %}
```bash
dep ensure -update sigs.k8s.io/controller-runtime sigs.k8s.io/controller-tools
```
{% endmethod %}

## Repopulating your vendor directory

{% method %}

Dependency source code is stored in the vendor directory.  If it ever gets
deleted, you can repopulate it using the exact dependency versions stored in
`Gopkg.lock`.

{% sample lang="bash" %}
```bash
dep ensure
```
{% endmethod %}

## How Kubebuilder's Dependencies Work

{% panel style="warning", title="Under the Hood" %}

The information in this section details how Kubebuilder's dependency graph
works.  It's not necessary for day-to-day use of Kubebuilder, but can be useful
if you want to understand how a particular version of Kubebuilder relates to
a particular version of Kubernetes.

{% endpanel %}

### TL;DR

As of Kubebuilder 1.0.2:

* Projects generated with Kubebuilder list a semantic version of
  controller-runtime and controller-tools as their only direct
  dependencies. All other Kubernetes-related libraries are transative
  dependencies.

* controller-runtime and controller-tools each list a specific, identical
  set of dependencies on Kubernetes libraries and related libraries.

* Once you've updated your dependencies with `kubebuilder update vendor`,
  you'll be able to run `dep ensure` and `dep ensure --update sigs.k8s.io/controller-runtime sigs.k8s.io/controller-tools` to safely
  update all your dependencies in the future.

* You can depend on controller-runtime to follow [semantic versioning
  guarantees](https://semver.org) -- we won't break your code without
  introducing a new major version, for both the interfaces in
  controller-runtime, and the bits of the kubernetes libraries that
  controller-runtime actually exposes.

### The Problem with Kubernetes libraries

The kubernetes project exports a collection of libraries (which we'll call
the **k8s-deps** from now on) that expose common functionality used when
building applications that consume Kubernetes APIs (e.g. clients,
informers, etc).  Due to the way Kubrenetes is versioned
(non-semantically), all of these dependencies must closely match --
differing versions can cause strange compilation or runtime errors.

Beyond this, these libraries have their own set of dependencies which are
not always the latest versions, or are occaisionally in-between versions.

Collecting the correct set of dependencies for any given Kubernetes
project can thus be tricky.

### Using Prebaked Manifests (Kubebuilder pre-1.0.2)

Before version 1.0.2, Kubebuilder shipped a pre-baked manifest of the
correct dependencies.  When scaffolding out at new project using
`kubebuilder init` (a **kb-project**), it would copy over a `Gopkg.toml`
file containing the exact dependency versions required for the project
(which could then be used by `dep` dependency management tool to actually
fetch the dependencies).

In addition to the Kubernetes dependencies required, this also specified
that all kb-projects use the master branch of the **controller-runtime**
library, which provides the abstractions that Kubebuilder is built upon.
Because controller-runtime wraps and consumes Kubernetes, it *also* needs
specific versions of the k8s-deps, and those version *must* match the ones
listed in the kb-project's Gopkg.toml, otherwise we'd have conflicting
dependencies.

#### The Problem with Prebaked Manifests

Using the master branch as the target version of controller-runtime made
it impossible to make breaking changes to controller-runtime.  However,
even when using a specific version of controller-runtime, it's still
difficult to make changes.

Since kb-projects must use an identical set of dependencies to
controller-runtime, any update to the controller-runtime dependencies
(say, to pull in a new feature) would have caused immediate dependency
version conflicts.  Effectively, any update to the dependencies had to be
treated as a major version revision, and there would have been no way to
tell the difference between "this release includes breaking API changes"
and "this release simply switches to a newer version of the k8s-deps".

### Transitive Dependencies (Kubebuilder 1.0.2+)

As noted above, any dependency version in kb-projects must match
dependency versions listed in controller-runtime, exactly. Furthermore, it
turns out, by design, the set of k8s-deps used in controller-runtime is
a superset of the set of dependencies actually imported by kb-projects.

Therefore, in kb-projects generated with Kubebuilder 1.0.2+, no
dependencies are listed besides controller-runtime (and controller-tools).
All of the k8s-deps become transitive dependencies, whose versions are
determined when `dep` (the dependency management tool) looks at the
versions required by controller-runtime.

controller-runtime is semantically versioned, so any changes to either the
interfaces in controller-runtime, or the pieces of the k8s-deps that are
exposed as part of those interfaces, means a new major version of
controller-runtime will be released.  Any other changes (new features, bug
fixes, updates to k8s-deps which don't break interfaces) yield minor or
patch versions (as per [semver](https://semver.org)), which can easily and
safely be updated to by kb-projects.

#### controller-tools Dependencies

controller-tools is the library used to generate CRD and RBAC manifests
for kb-projects. With Kubebuilder 1.0.2+, it does not directly depend on
controller-runtime, but shares the same set of dependencies.  It therefore
must be updated in lockstep with controller-runtime.  This is mostly
a concern of the controller-tools/controller-runtime maintainers, and will
not affect users.  Like controller-runtime, controller-tools uses semver.
