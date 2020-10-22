/*
Copyright 2020 The Kubernetes authors.

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

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	batchv1 "tutorial.kubebuilder.io/project/api/v1"
	"tutorial.kubebuilder.io/project/controllers"
	// +kubebuilder:scaffold:imports
)

// +kubebuilder:docs-gen:collapse=Imports

/*
要注意的第一个区别是 kubebuilder 已经添加了一组新的 API 包（`batchv1`）到 scheme。这意味着可以在我们的控制器中使用这些对象。

如果我们要使用任何其他的 CRD，我们必须用相同的方法添加它们的 scheme。内置的类型例如 Job 就添加了它自己的 scheme `clientgoscheme`。
*/
var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(batchv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

/*
另一个变化是 kubebuilder 已经添加了一个阻塞调用我们的 CornJob 控制器的 `SetupWithManager` 方法。
*/

func main() {
	/*
	 */
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "80807133.tutorial.kubebuilder.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// +kubebuilder:docs-gen:collapse=old stuff

	if err = (&controllers.CronJobReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("CronJob"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CronJob")
		os.Exit(1)
	}
	if err = (&batchv1.CronJob{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "CronJob")
		os.Exit(1)
	}

	/*
	    我们也会为我们的类型设置 webhook，接下来我们会谈到它。我们只需要将他们添加到 manager。因为我们想分开运行 webhook，而不是在本地测试我们的控制器时运行它们，我们会将它们配置到环境变量中。

		当我们本地运行的时候只需确保 `ENABLE_WEBHOOKS=false`。
	*/
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&batchv1.CronJob{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Captain")
			os.Exit(1)
		}
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
