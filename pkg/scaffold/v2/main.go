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
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v2/internal"
)

const (
	apiPkgImportScaffoldMarker    = "// +kubebuilder:scaffold:imports"
	apiSchemeScaffoldMarker       = "// +kubebuilder:scaffold:scheme"
	reconcilerSetupScaffoldMarker = "// +kubebuilder:scaffold:builder"
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

// Update updates main.go with code fragments required to wire a new
// resource/controller.
func (m *Main) Update(opts *MainUpdateOptions) error {
	path := "main.go"

	resPkg, _ := getResourceInfo(opts.Resource, input.Input{
		Domain: opts.Project.Domain,
		Repo:   opts.Project.Repo,
	})

	// generate all the code fragments
	apiImportCodeFragment := fmt.Sprintf(`%s%s "%s/%s"
`, opts.Resource.Group, opts.Resource.Version, resPkg, opts.Resource.Version)
	ctrlImportCodeFragment := fmt.Sprintf(`"%s/controllers"
`, opts.Project.Repo)
	addschemeCodeFragment := fmt.Sprintf(`_ = %s%s.AddToScheme(scheme)
`, opts.Resource.Group, opts.Resource.Version)
	reconcilerSetupCodeFragment := fmt.Sprintf(`err = (&controllers.%sReconciler{
	 	Client: mgr.GetClient(),
        Log: ctrl.Log.WithName("controllers").WithName("%s"),
	 }).SetupWithManager(mgr)
	 if err != nil {
	 	setupLog.Error(err, "unable to create controller", "controller", "%s")
	 	os.Exit(1)
	 }
`, opts.Resource.Kind, opts.Resource.Kind, opts.Resource.Kind)

	if opts.WireResource {
		err := internal.InsertStringsInFile(path,
			map[string][]string{
				apiPkgImportScaffoldMarker: []string{apiImportCodeFragment},
				apiSchemeScaffoldMarker:    []string{addschemeCodeFragment},
			})
		if err != nil {
			return err
		}
	}

	if opts.WireController {
		return internal.InsertStringsInFile(path,
			map[string][]string{
				apiPkgImportScaffoldMarker:    []string{apiImportCodeFragment, ctrlImportCodeFragment},
				apiSchemeScaffoldMarker:       []string{addschemeCodeFragment},
				reconcilerSetupScaffoldMarker: []string{reconcilerSetupCodeFragment},
			})
	}

	return nil
}

// MainUpdateOptions contains info required for wiring an API/Controller in
// main.go.
type MainUpdateOptions struct {
	// Project contains info about the project
	Project *input.ProjectFile

	// Resource is the resource being added
	Resource *resource.Resource

	// Flags to indicate if resource/controller is being scaffolded or not
	WireResource   bool
	WireController bool
}

var mainTemplate = fmt.Sprintf(`{{ .Boilerplate }}

package main

import (
	"flag"
    "os"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/log/zap"
    "k8s.io/apimachinery/pkg/runtime"


	%s
)

var (
	scheme   = clientgoscheme.Scheme
    setupLog = ctrl.Log.WithName("setup")
)

func init() {

	%s
}

func main() {
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


    %s

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
`, apiPkgImportScaffoldMarker, apiSchemeScaffoldMarker, reconcilerSetupScaffoldMarker)
