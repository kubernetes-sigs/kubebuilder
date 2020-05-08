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

	crewv1 "sigs.k8s.io/kubebuilder/testdata/project-v2-multigroup/apis/crew/v1"
	foopolicyv1 "sigs.k8s.io/kubebuilder/testdata/project-v2-multigroup/apis/foo.policy/v1"
	seacreaturesv1beta1 "sigs.k8s.io/kubebuilder/testdata/project-v2-multigroup/apis/sea-creatures/v1beta1"
	seacreaturesv1beta2 "sigs.k8s.io/kubebuilder/testdata/project-v2-multigroup/apis/sea-creatures/v1beta2"
	shipv1 "sigs.k8s.io/kubebuilder/testdata/project-v2-multigroup/apis/ship/v1"
	shipv1beta1 "sigs.k8s.io/kubebuilder/testdata/project-v2-multigroup/apis/ship/v1beta1"
	shipv2alpha1 "sigs.k8s.io/kubebuilder/testdata/project-v2-multigroup/apis/ship/v2alpha1"
	crewcontroller "sigs.k8s.io/kubebuilder/testdata/project-v2-multigroup/controllers/crew"
	foopolicycontroller "sigs.k8s.io/kubebuilder/testdata/project-v2-multigroup/controllers/foo.policy"
	seacreaturescontroller "sigs.k8s.io/kubebuilder/testdata/project-v2-multigroup/controllers/sea-creatures"
	shipcontroller "sigs.k8s.io/kubebuilder/testdata/project-v2-multigroup/controllers/ship"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(crewv1.AddToScheme(scheme))
	utilruntime.Must(shipv1beta1.AddToScheme(scheme))
	utilruntime.Must(shipv1.AddToScheme(scheme))
	utilruntime.Must(shipv2alpha1.AddToScheme(scheme))
	utilruntime.Must(seacreaturesv1beta1.AddToScheme(scheme))
	utilruntime.Must(seacreaturesv1beta2.AddToScheme(scheme))
	utilruntime.Must(foopolicyv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
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
		LeaderElectionID:   "e9b53b87.testproject.org",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&crewcontroller.CaptainReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("crew").WithName("Captain"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Captain")
		os.Exit(1)
	}
	if err = (&crewv1.Captain{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Captain")
		os.Exit(1)
	}
	if err = (&shipcontroller.FrigateReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ship").WithName("Frigate"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Frigate")
		os.Exit(1)
	}
	if err = (&shipv1beta1.Frigate{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Frigate")
		os.Exit(1)
	}
	if err = (&shipcontroller.DestroyerReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ship").WithName("Destroyer"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Destroyer")
		os.Exit(1)
	}
	if err = (&shipcontroller.CruiserReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ship").WithName("Cruiser"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Cruiser")
		os.Exit(1)
	}
	if err = (&seacreaturescontroller.KrakenReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("sea-creatures").WithName("Kraken"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Kraken")
		os.Exit(1)
	}
	if err = (&seacreaturescontroller.LeviathanReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("sea-creatures").WithName("Leviathan"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Leviathan")
		os.Exit(1)
	}
	if err = (&foopolicycontroller.HealthCheckPolicyReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("foo.policy").WithName("HealthCheckPolicy"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HealthCheckPolicy")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
