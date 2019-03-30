/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
# Tutorial

Too many tutorials start out with some really contrived setup, or some toy
application that gets the basics across, and then stalls out on the more
complicated suff.  Instead, this tutorial should take you through (almost)
the full gamut of complexity, starting off simple and building up to
something pretty full-featured.

Let's pretend (and sure, this is a teensy bit contrived) that we've
finally gotten tired of the maintenance burden of the from-scratch
implementation of the CronJob controller in Kuberntes, and we'd like to
rewrite it using KubeBuilder.

The job (no pun intended) of the *CronJob* controller is to run one-off
tasks on the Kubernetes cluster at regular intervals.  It does the by
bulding on top of the *Job* controller, whose task is to run one-off tasks
once, seeing them to completion.

Instead of trying to tackle rewriting the Job controller as well, we'll
use this as an opportunity to see how to interact with external types.
*/

/*
# Scaffolding out our project

Any Go project starts with the same basic stuff: you need some
dependencies, and some starting files, like a main.go.

The same is true for KubeBuilder projects.	Thankfully, KubeBuilder can
scaffold out these initial files for us.  This will create a `Gopkg.toml`
file containing the necessary dependencies, a main.go file, as well as
a project metadata file to help with later scaffolding.

```shell
# our domain is kubebuilder.io, and our group will be testproject,
# yielding a full group name of testproject.kubebuilder.io.
kubebuilder init --domain kubebuilder.io
```
*/

/*
# Every program starts with a main...

Let's take a look at the main.go file:
*/
package main

import (
	"os"

/*
Most common controller-runtime logic can be found in the main
controller-runtime package, which we usually spell `ctrl` to save on
typing (we're programmers, so we're [fundamentally
lazy](http://threevirtues.com/)).

controller-runtime makes use of the
[logr](https://github.com/go-logr/logr) project for logging, which makes
the backing logging implementation pluggable.  We're going to use
[Zap](https://go.uber.org/zap), which controller-runtime has easy setup
for.

We'll also need the `Scheme` type from the core Kubernetes APIMachinery
(more below), and the batch/v1 API group from Kubernetes (for the Job type).
*/
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"k8s.io/apimachinery/pkg/runtime"
	batchv1 "k8s.io/api/batch/v1"

/*
Since we're recreating the CronJob controller, we might as well recreate
the types as well.  Let's scaffold those out:

```shell
# we'll name the group "tutorial" (as decided above),
# and we'll be bold and confident and go straight to v1.
# We can always add [conversion](TODO) if
# we need a new version.
kubebuilder create api --group tutorial --version v1
```
*/

/*
The scaffolding tool will import our API, as well as our controllers
*/
	"sigs.k8s.io/kubebuilder/test/project/api/v1"
	"sigs.k8s.io/kubebuilder/test/project/controllers"
)

/*
In Kubernetes, we call each concrete "type" a Kind.  A Kind is some
serialized bit of data that.  Kinds have corresponding Go types to
represent them when in a Go program.  To map between the two, we use
a type called `Scheme` from the main Kubernetes API machinery.

The scaffolding tool will add a line inserting the mappings from our Go
types to their corresponding Kinds in this file automatically.	We'll also
want to add the mappings from the batch/v1 API group from Kubernetes,
since we'll be referencing the built-in Job type.
*/

var (
	scheme = runtime.NewScheme()
)

func init() {
	v1.AddToScheme(scheme)
	batchv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

/*
Now, on to the main attraction (pun intended) of this file:
the actual entrypoint.
*/
func main() {

/*
First, we'll set up some logging.  We need to tell controller-runtime that
we're using Zap, as noted above.  controller-runtime allows its internals, as
well as our program, to lazily request access to loggers with the `Log` object,
even before we've actually set a concrete implementation.  Nothing happens on
these loggers until we actually set a logging implemention, though.

In this case, we can use a helper from controller-runtime to set up Zap in
development mode.

[Logr](github.com/go-logr/logr) (the generic interface that Zap implements
here) is a *structured* logging library, which means we attach names and key-value
pairs as context to fixed log messages.  The scaffoling has set up a log line
for our setup logic.
*/
	ctrl.SetLogger(zap.Logger(true))  // true means "development mode"
	setupLog := ctrl.Log.WithName("setup")

/*
Next, our scaffolding has set up a *manager*.  A manager sets up common dependencies,
like caches shared between our controllers, and is in charge of actually running
our reconcile loops.  It needs our scheme from above to do its job.
*/
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{Scheme: scheme})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

/*
Then, we'll add our controller to the manager.	While controller-runtime lets
you set up your controllers however, KubeBuilder encourages you to instantiate
the struct in the entrypoint to your program, but keep the general setup logic
for the controller next to the reconciler itself.

This lets you pass in dependencies, flag values, etc from the entrypoint, but
still easily set up the controller in any unit/integration tests that live
alongside the controller.
*/
	err = (&controllers.CronJobReconciler{
		Log: ctrl.Log.WithName("controllers").WithName("cronjob"),
	}).SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "mykind")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

/*
Finally, the manager is started.  It's set up to run until it's told to stop by
a "stop channel" (which we close when we're done).	Here, our stop channel is
actually wired up to a signal handler, so that we can gracefully shut down when
running on Kubernetes.
*/
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

/*
Now that we've got our entrypoint out of the way, we can fill out our types,
and then we'll write our controller.

The types live under `api/<version>/`, so in our case thats `api/v1`.

+goto ./api/v1/doc.go
*/
