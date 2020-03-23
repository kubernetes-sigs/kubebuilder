/*

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
// +kubebuilder:docs-gen:collapse=Apache License

/*

Our package starts out with some basic imports.  Particularly:

- The core [controller-runtime](https://godoc.org/sigs.k8s.io/controller-runtime) library
- The default controller-runtime logging, Zap (more on that a bit later)

*/

package main

import (
	"flag"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

/*
Every set of controllers needs a
[*Scheme*](https://book.kubebuilder.io/cronjob-tutorial/gvks.html#err-but-whats-that-scheme-thing),
which provides mappings between Kinds and their corresponding Go types.  We'll
talk a bit more about Kinds when we write our API definition, so just keep this
in mind for later.
*/
var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {

	// +kubebuilder:scaffold:scheme
}

/*
At this point, our main function is fairly simple:

- We set up some basic flags for metrics.

- We instantiate a
[*manager*](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/manager#Manager),
which keeps track of running all of our controllers, as well as setting up
shared caches and clients to the API server (notice we tell the manager about
our Scheme).

- We run our manager, which in turn runs all of our controllers and webhooks.
The manager is set up to run until it receives a graceful shutdown signal.
This way, when we're running on Kubernetes, we behave nicely with graceful
pod termination.

While we don't have anything to run just yet, remember where that
`+kubebuilder:scaffold:builder` comment is -- things'll get interesting there
soon.

*/

func main() {
	var metricsAddr string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{Scheme: scheme, MetricsBindAddress: metricsAddr})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	/*
		Note that the Manager can restrict the namespace that all controllers will watch for resources by:
	*/

	mgr, err = ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		Namespace:          namespace,
		MetricsBindAddress: metricsAddr,
	})

	/*
		The above example will change the scope of your project to a single Namespace. In this scenario,
		it is also suggested to restrict the provided authorization to this namespace by replacing the default
		ClusterRole and ClusterRoleBinding to Role and RoleBinding respectively
		For further information see the kubernetes documentation about Using [RBAC Authorization](https://kubernetes.io/docs/reference/access-authn-authz/rbac/).

		Also, it is possible to use the MultiNamespacedCacheBuilder to watch a specific set of namespaces:
	*/

	var namespaces []string // List of Namespaces

	mgr, err = ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		NewCache:           cache.MultiNamespacedCacheBuilder(namespaces),
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})

	/*
		For further information see [MultiNamespacedCacheBuilder](https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder)
	*/

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
