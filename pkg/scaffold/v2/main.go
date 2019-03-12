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

package v2

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

var _ input.File = &Main{}

// Main scaffolds a main.go to run Controllers
type Main struct {
	input.Input
}

// GetInput implements input.File
func (m *Main) GetInput() (input.Input, error) {
	if m.Path == "" {
		m.Path = filepath.Join("main.go")
	}
	m.TemplateBody = mainTemplate
	return m.Input, nil
}

var mainTemplate = `{{ .Boilerplate }}

package main

import (
	"flag"
    "os"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/log/zap"
    "k8s.io/apimachinery/pkg/runtime"
)


// +kubebuilder:scaffold:import-api
// import (
// 	"{{ .Repo }}/pkg/apis"
// )
	

var (
    scheme = runtime.NewScheme()
    setupLog = ctrl.Log.WithName("setup")
)

// +kubebuilder:scaffold:scheme
func init() {
    // v1beta1.AddToScheme(scheme)
    // v1.AddToScheme(scheme)
}

func main() {
	var metricsAddr string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.Parse()

	ctrl.SetLogger(zap.Logger(true))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{ Scheme: scheme, MetricsBindAddress: metricsAddr })
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// err = controllers.MyKindReconciler{
	// 	Client: mgr.GetClient(),
    //     log: ctrl.Log.WithName("mykind-controller"),
	// }.SetupWithManager(mgr)
	// if err != nil {
	// 	setupLog.Error(err, "unable to create controller", "controller", "mykind")
	// 	os.Exit(1)
	// }

    // +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
`
