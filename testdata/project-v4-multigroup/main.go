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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	crewv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/apis/crew/v1"
	fizv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/apis/fiz/v1"
	foopolicyv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/apis/foo.policy/v1"
	foov1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/apis/foo/v1"
	seacreaturesv1beta1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/apis/sea-creatures/v1beta1"
	seacreaturesv1beta2 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/apis/sea-creatures/v1beta2"
	shipv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/apis/ship/v1"
	shipv1beta1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/apis/ship/v1beta1"
	shipv2alpha1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/apis/ship/v2alpha1"
	testprojectorgv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/apis/v1"
	"sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/controllers"
	appscontrollers "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/controllers/apps"
	crewcontrollers "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/controllers/crew"
	fizcontrollers "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/controllers/fiz"
	foocontrollers "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/controllers/foo"
	foopolicycontrollers "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/controllers/foo.policy"
	seacreaturescontrollers "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/controllers/sea-creatures"
	shipcontrollers "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/controllers/ship"
	//+kubebuilder:scaffold:imports
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
	utilruntime.Must(foov1.AddToScheme(scheme))
	utilruntime.Must(fizv1.AddToScheme(scheme))
	utilruntime.Must(testprojectorgv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	var err error
	options := ctrl.Options{Scheme: scheme}
	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile))
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&crewcontrollers.CaptainReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Captain")
		os.Exit(1)
	}
	if err = (&crewv1.Captain{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Captain")
		os.Exit(1)
	}
	if err = (&shipcontrollers.FrigateReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Frigate")
		os.Exit(1)
	}
	if err = (&shipv1beta1.Frigate{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Frigate")
		os.Exit(1)
	}
	if err = (&shipcontrollers.DestroyerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Destroyer")
		os.Exit(1)
	}
	if err = (&shipv1.Destroyer{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Destroyer")
		os.Exit(1)
	}
	if err = (&shipcontrollers.CruiserReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Cruiser")
		os.Exit(1)
	}
	if err = (&shipv2alpha1.Cruiser{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Cruiser")
		os.Exit(1)
	}
	if err = (&seacreaturescontrollers.KrakenReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Kraken")
		os.Exit(1)
	}
	if err = (&seacreaturescontrollers.LeviathanReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Leviathan")
		os.Exit(1)
	}
	if err = (&foopolicycontrollers.HealthCheckPolicyReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HealthCheckPolicy")
		os.Exit(1)
	}
	if err = (&appscontrollers.DeploymentReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Deployment")
		os.Exit(1)
	}
	if err = (&foocontrollers.BarReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Bar")
		os.Exit(1)
	}
	if err = (&fizcontrollers.BarReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Bar")
		os.Exit(1)
	}
	if err = (&controllers.LakersReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Lakers")
		os.Exit(1)
	}
	if err = (&testprojectorgv1.Lakers{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Lakers")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

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
