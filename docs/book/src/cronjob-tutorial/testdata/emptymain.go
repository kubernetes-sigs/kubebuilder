/*
Copyright 2022 The Kubernetes authors.

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

- The core [controller-runtime](https://pkg.go.dev/sigs.k8s.io/controller-runtime?tab=doc) library
- The default controller-runtime logging, [Zap](https://pkg.go.dev/go.uber.org/zap) (more on that a bit later)

*/

package main

import (
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
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
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme
}

/*
At this point, our main function is fairly simple:

- We set up some basic flags for metrics.

- We instantiate a
[*manager*](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/manager?tab=doc#Manager),
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
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer:          webhook.NewServer(webhook.Options{Port: 9443}),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "80807133.tutorial.kubebuilder.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	/*
		Note that the `Manager` can restrict the namespace that all controllers will watch for resources by:
	*/

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				namespace: {},
			},
		},
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer:          webhook.NewServer(webhook.Options{Port: 9443}),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "80807133.tutorial.kubebuilder.io",
	})

	/*
		The above example will change the scope of your project to a single `Namespace`. In this scenario,
		it is also suggested to restrict the provided authorization to this namespace by replacing the default
		`ClusterRole` and `ClusterRoleBinding` to `Role` and `RoleBinding` respectively.
		For further information see the Kubernetes documentation about [Using RBAC Authorization](https://kubernetes.io/docs/reference/access-authn-authz/rbac/).

		Also, it is possible to use the [`DefaultNamespaces`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/cache#Options)
		from `cache.Options{}` to cache objects in a specific set of namespaces:
	*/

	var namespaces []string // List of Namespaces
	defaultNamespaces := make(map[string]cache.Config)

	for _, ns := range namespaces {
		defaultNamespaces[ns] = cache.Config{}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Cache: cache.Options{
			DefaultNamespaces: defaultNamespaces,
		},
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer:          webhook.NewServer(webhook.Options{Port: 9443}),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "80807133.tutorial.kubebuilder.io",
	})

	/*
		For further information see [`cache.Options{}`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/cache#Options)
	*/

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
