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
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

const (
	apiPkgImportScaffoldMarker    = "+kubebuilder:scaffold:imports"
	apiSchemeScaffoldMarker       = "+kubebuilder:scaffold:scheme"
	reconcilerSetupScaffoldMarker = "+kubebuilder:scaffold:builder"
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

var mainTemplate = fmt.Sprintf(`{{ .Boilerplate }}

package main

import (
	"flag"
    "os"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/log/zap"
    "k8s.io/apimachinery/pkg/runtime"


	// %s
)

var (
    scheme = runtime.NewScheme()
    setupLog = ctrl.Log.WithName("setup")
)

func init() {

	// %s
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


    // %s

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
`, apiPkgImportScaffoldMarker, apiSchemeScaffoldMarker, reconcilerSetupScaffoldMarker)
