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

package main

import (
	"flag"
	"os"

	kbatchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	batchv1 "tutorial.kubebuilder.io/project/api/v1"
	"tutorial.kubebuilder.io/project/controllers"
	// +kubebuilder:scaffold:imports
)

// +kubebuilder:docs-gen:collapse=Imports

/*
The first difference to notice is that kubebuilder has added the new API
group's package (`batchv1`) to our scheme.  This means that we can use those
objects in our controller.

We'll also need to add the kubernetes batch v1 (`kbatchv1`) scheme, since we're creating
and listing Jobs.
*/
var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {

	kbatchv1.AddToScheme(scheme) // we've added this ourselves
	batchv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

/*
The other thing that's changed is that kubebuilder has added a block calling our
CronJob controller's `SetupWithManager` method.  Since we now use a `Scheme` as well,
we'll need to pass that to the reconciler ourselves.
*/

func main() {
	/*
	 */
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.Logger(true))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	// +kubebuilder:docs-gen:collapse=old stuff

	err = (&controllers.CronJobReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("CronJob"),
		Scheme: mgr.GetScheme(), // we've added this ourselves
	}).SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CronJob")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	/*
	 */

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
	// +kubebuilder:docs-gen:collapse=old stuff
}
