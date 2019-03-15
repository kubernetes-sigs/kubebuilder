package main

import (
    "os"

    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/log/zap"
    "k8s.io/apimachinery/pkg/runtime"

    "sigs.k8s.io/kubebuilder/test/project/api/v1beta1"
    "sigs.k8s.io/kubebuilder/test/project/controllers"
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
	ctrl.SetLogger(zap.Logger(true))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{Scheme: scheme})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

    // +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
