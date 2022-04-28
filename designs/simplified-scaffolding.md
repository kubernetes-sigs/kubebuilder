Simplified Builder-Based Scaffolding
====================================

## Background

The current scaffolding in kubebuilder produces a directory structure that
looks something like this (compiled artifacts like config omitted for
brevity):

<details>

<summary>`tree -d ./test/project`</summary>

```shell
$ tree -d ./test/project
./test/project
├── cmd
│   └── manager
├── pkg
│   ├── apis
│   │   ├── creatures
│   │   │   └── v2alpha1
│   │   ├── crew
│   │   │   └── v1
│   │   ├── policy
│   │   │   └── v1beta1
│   │   └── ship
│   │       └── v1beta1
│   ├── controller
│   │   ├── firstmate
│   │   ├── frigate
│   │   ├── healthcheckpolicy
│   │   ├── kraken
│   │   └── namespace
│   └── webhook
│       └── default_server
│           ├── firstmate
│           │   └── mutating
│           ├── frigate
│           │   └── validating
│           ├── kraken
│           │   └── validating
│           └── namespace
│               └── mutating
└── vendor
```

</details>

API packages have a separate file for each API group that creates a SchemeBuilder,
a separate file to aggregate those scheme builders together, plus files for types,
and the per-group-version scheme builders as well:

<details>

<summary>`tree ./test/project/pkg/apis`</summary>

```shell
$ ./test/project/pkg/apis
├── addtoscheme_creatures_v2alpha1.go
├── apis.go
├── creatures
│   ├── group.go
│   └── v2alpha1
│       ├── doc.go
│       ├── kraken_types.go
│       ├── kraken_types_test.go
│       ├── register.go
│       ├── v2alpha1_suite_test.go
│       └── zz_generated.deepcopy.go
...
```

</details>

Controller packages have a separate file that registers each controller with a global list
of controllers, a file that provides functionality to register that list with a manager,
as well as a file that constructs the individual controller itself:

<details>

<summary>`tree ./test/project/pkg/controller`</summary>

```shell
$ tree ./test/project/pkg/controller
./test/project/pkg/controller
├── add_firstmate.go
├── controller.go
├── firstmate
│   ├── firstmate_controller.go
│   ├── firstmate_controller_suite_test.go
│   └── firstmate_controller_test.go
...
```

</details>

## Motivation

The current scaffolding in Kubebuilder has two main problems:
comprehensibility and dependency passing.

### Complicated Initial Structure

While the structure of Kubebuilder projects will likely feel at home for
existing Kubernetes contributors (since it matches the structure of
Kubernetes itself quite closely), it provides a fairly convoluted
experience out of the box.

Even for a single controller and API type (without a webhook), it
generates 8 API-related files and 5 controller-related files.  Of those
files, 6 are Kubebuilder-specific glue code, 4 are test setup, and
1 contains standard Kubernetes glue code, leaving only 2 with actual
user-edited code.

This proliferation of files makes it difficult for users to understand how
their code relates to the library, posing some barrier for initial adoption
and moving beyond a basic knowledge of functionality to actual
understanding of the structure.  A common line of questioning amongst
newcomers to Kubebuilder includes "where should I put my code that adds
new types to a scheme" (and similar questions), which indicates that it's
not immediately obvious to these users why the project is structured the
way it is.

Additionally, we scaffold out API "tests" that test that the API server is
able to receive create requests for the objects, but don't encourage
modification beyond that.  An informal survey seems to indicate that most
users don't actually modify these tests (many repositories continue to
look like
[this](https://github.com/replicatedhq/gatekeeper/blob/3bfe0f7213b6d41abf2df2a6746f3351e709e6ff/pkg/apis/policies/v1alpha2/admissionpolicy_types_test.go)).
If we want to help users test that their object's structure is the way
they think it is, we're probably better served coming up with a standard
"can I create this example YAML file".

Furthermore, since the structure is quite convoluted, it makes it more
difficult to write examples, since the actual code we care about ends up
scattered deep in a folder structure.

### Lack of Builder

We introduced the builder pattern for controller construction in
controller-runtime
([GoDoc](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/builder?tab=doc#ControllerManagedBy))
as a way to simplify construction of controllers and reduce boilerplate
for the common cases of controller construction.  Informal feedback from
this has been positive, and it enables fairly rapid, clear, and concise
construction of controllers (e.g. this [one file
controller](https://github.com/DirectXMan12/sample-controller/blob/workshop/main.go)
used as a getting started example for a workshop).

Current Kubebuilder scaffolding does not take advantage of the builder,
leaving generated code using the lower-level constructs which require more
understanding of the internals of controller-runtime to comprehend.

### Dependency Passing Woes

Another common line of questioning amongst Kubebuilder users is "how to
I pass dependencies to my controllers?".  This ranges from "how to I pass
custom clients for the software I'm running" to "how to I pass
configuration from files and flags down to my controllers" (e.g.
[kubernete-sigs/kubebuilder#611](https://github.com/kubernetes-sigs/kubebuilder/issues/611)

Since reconciler implementations are initialized in `Add` methods with
standard signatures, dependencies cannot be passed directly to
reconcilers.  This has lead to requests for dependency injection in
controller-runtime (e.g.
[kubernetes-sigs/controller-runtime#102](https://github.com/kubernetes-sigs/controller-runtime/issues/102)),
but in most cases, a structure more amicable to passing in the
dependencies directly would solve the issue (as noted in
[kubernetes-sigs/controller-runtime#182](https://github.com/kubernetes-sigs/controller-runtime/pull/182#issuecomment-442615175)).

## Revised Structure

In the revised structure, we use the builder pattern to focus on the
"code-refactor-code-refactor" cycle: start out with a simple structure,
refactor out as your project becomes more complicated.

Users receive a simply scaffolded structure to start. Simple projects can
remain relatively simple, and complicated projects can decide to adopt
a different structure as they grow.

The new scaffold project structure looks something like this (compiled
artifacts like config omitted for brevity):

```shell
$ tree ./test/project
./test/project
├── main.go
├── controller
│   ├── mykind_controller.go
│   ├── mykind_controller_test.go
│   └── controllers_suite_test.go
├── api
│   └── v1
│       └── mykind_types.go
│       └── groupversion_info.go
└── vendor
```

In this new layout, `main.go` constructs the reconciler:

```go
// ...
func main() {
	// ...
	err := (&controllers.MyReconciler{
		MySuperSpecialAppClient: doSomeThingsWithFlags(),
	}).SetupWithManager(mgr)
	// ...
}
```

while `mykind_controller.go` actually sets up the controller using the
reconciler:

```go
func (r *MyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.MyAppType{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
```

This makes it abundantly clear where to start looking at the code
(`main.go` is the defacto standard entry-point for many go programs), and
simplifies the levels of hierarchy.  Furthermore, since `main.go` actually
instantiates an instance of the reconciler, users are able to add custom
logic having to do with flags.

Notice that we explicitly construct the reconciler in `main.go`, but put
the setup logic for the controller details in `mykind_controller.go`. This
makes testing easier (see
[below](#put-the-controller-setup-code-in-main-go)), but still allows us
to pass in dependencies from `main`.

### Why don't we...

#### Put the controller setup code in main.go

While this is an attractive pattern from a prototyping perspective, it
makes it harder to write integration tests, since you can't easily say
"run this controller with all its setup in processes".  With a separate
`SetupWithManager` method associated with reconcile, it becomes fairly
easy to setup with a manager.

#### Put the types directly under api/, or not have groupversion_info.go

These suggestions make it much harder to scaffold out additional versions
and kinds.  You need to have each version in a separate package, so that
type names don't conflict.  While we could put scheme registration in with
`kind_types.go`, if a project has multiple "significant" Kinds in an API
group, it's not immediately clear which file has the scheme registration.

#### Use a single types.go file

This works fine when you have a single "major" Kind, but quickly grows
unwieldy when you have multiple major kinds and end up with
a hundreds-of-lines-long `types.go` file (e.g. the `appsv1` API group in
core Kubernetes).  Splitting out by "major" Kind (`Deployment`,
`ReplicaSet`, etc) makes the code organization clearer.

#### Change the current scaffold to just make Add a method on the reconciler

While this solves the dependency issues (mostly, since you might want to
further pass configuration to the setup logic and not just the runtime
logic), it does not solve the underlying pedagogical issues around the
initial structure burying key logic amidst a sprawl of generated files and
directories.

### Making this work with multiple controllers, API versions, API groups, etc

#### Versions

Most projects will eventually grow multiple API versions.  The only
wrinkle here is making sure API versions get added to a scheme.  This can
be solved by adding a specially-marked init function that registration
functions get added to (see the example).

#### Groups

Some projects eventually grow multiple API groups.  Presumably, in the
case of multiple API groups, the desired hierarchy is:

```shell
$ tree ./test/project/api
./test/project/api
├── groupa
│   └── v1
│       └── types.go
└── groupb
    └── v1
        └── types.go
```

There are three options here:

1. Scaffold with the more complex API structure (this looks pretty close
   to what we do today).  It doesn't add a ton of complexity, but does
   bury types deeper in a directory structure.

2. Try to move things and rename references.  This takes a lot more effort
   on the Kubebuilder maintainers' part if we try to rename references
   across the codebase.  Not so much if we force the user to, but that's
   a poorer experience.

3. Tell users to move things, and scaffold out with the new structure.
   This is fairly messy for the user.

Since growing to multiple API groups seems to be fairly uncommon, it's
mostly like safe to take a hybrid approach here -- allow manually
specifying the output path, and, when not specified, asking the user to
first restructure before running the command.

#### Controllers

Multiple controllers don't need their own package, but we'd want to
scaffold out the builder.  We have two options here:

1. Looking for a particular code comment, and appending a new builder
   after it.  This is a bit more complicated for us, but perhaps provides
   a nicer UX.

2. Simply adding a new controller, and reminding the user to add the
   builder themselves.  This is easier for the maintainers, but perhaps
   a slightly poorer UX for the users.  However, writing out a builder by
   hand is significantly less complex than adding a controller by hand in
   the current structure.

Option 1 should be fairly simple, since the logic is already needed for
registering types to the scheme, and we can always fall back to emitting
code for the user to place in manually if we can't find the correct
comment.

### Making this work with Existing Kubebuilder Installations

Kubebuilder projects currently have a `PROJECT` file that can be used to
store information about project settings.  We can make use of this to
store a "scaffolding version", where we increment versions when making
incompatible changes to how the scaffolding works.

A missing scaffolding version field implies the version `1`, which uses
our current scaffolding semantics.  Version `2` uses the semantics
proposed here.  New projects are scaffolded with `2`, and existing
projects check the scaffold version before attempting to add addition API
versions, controllers, etc

### Teaching more complicated project structures

Some controllers may eventually want more complicated project structures.
We should have a section of the book recommending options for when you
project gets very complicated.

### Additional Tooling Work

* Currently the `api/` package will need a `doc.go` file to make
  `deepcopy-gen` happy.  We should fix this.

* Currently, `controller-gen crd` needs the `api` directory to be
  `pkg/apis/<group>/<version>`.  We should fix this.

## Example

See #000 for an example with multiple stages of code generation
(representing the examples is this form is rather complicated, since it
involves multiple files).

```shell
$ kubebuilder init --domain test.k8s.io
$ kubebuilder create api --group mygroup --version v1beta1 --kind MyKind
$ kubebuilder create api --group mygroup --version v2beta1 --kind MyKind
$ tree .
.
├── main.go
├── controller
│   ├── mykind_controller.go
│   ├── controller_test.go
│   └── controllers_suite_test.go
├── api
│   ├── v1beta1
│   │   ├── mykind_types.go
│   │   └── groupversion_info.go
│   └── v1
│       ├── mykind_types.go
│       └── groupversion_info.go
└── vendor
```

<details>

<summary>main.go</summary>

```go
package main

import (
    "os"

    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/log/zap"
    "k8s.io/apimachinery/pkg/runtime"

    "my.repo/api/v1beta1"
    "my.repo/api/v1"
    "my.repo/controllers"
)

var (
    scheme = runtime.NewScheme()
    setupLog = ctrl.Log.WithName("setup")
)

func init() {
    v1beta1.AddToScheme(scheme)
    v1.AddToScheme(scheme)
    // +kubebuilder:scaffold:scheme
}

func main() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{Scheme: scheme})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	err = (&controllers.MyKindReconciler{
		Client: mgr.GetClient(),
        log: ctrl.Log.WithName("mykind-controller"),
	}).SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "mykind")
		os.Exit(1)
	}

    // +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
```

</details>

<details>

<summary>mykind_controller.go</summary>

```go
package controllers

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"github.com/go-logr/logr"

	"my.repo/api/v1"
)

type MyKindReconciler struct {
	client.Client
	log logr.Logger
}

func (r *MyKindReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.log.WithValues("mykind", req.NamespacedName)

	// your logic here

	return req.Result{}, nil
}

func (r *MyKindReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(v1.MyKind{}).
		Complete(r)
}
```

</details>

`*_types.go` looks nearly identical to the current standard.

<details>

<summary>groupversion_info.go</summary>

```go
package v1

import (
	"sigs.k8s.io/controller-runtime/pkg/scheme"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	GroupVersion = schema.GroupVersion{Group: "mygroup.test.k8s.io", Version: "v1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
```

</details>
