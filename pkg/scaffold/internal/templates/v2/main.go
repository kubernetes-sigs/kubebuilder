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

	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/model/file"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/internal/machinery"
)

const (
	APIPkgImportScaffoldMarker    = "// +kubebuilder:scaffold:imports"
	APISchemeScaffoldMarker       = "// +kubebuilder:scaffold:scheme"
	ReconcilerSetupScaffoldMarker = "// +kubebuilder:scaffold:builder"
)

var _ file.Template = &Main{}

// Main scaffolds a main.go to run Controllers
type Main struct {
	file.Input
}

// GetInput implements input.Template
func (f *Main) GetInput() (file.Input, error) {
	if f.Path == "" {
		f.Path = filepath.Join("main.go")
	}
	f.TemplateBody = mainTemplate
	return f.Input, nil
}

// Update updates main.go with code fragments required to wire a new
// resource/controller.
func (f *Main) Update(opts *MainUpdateOptions) error {
	path := "main.go"

	// generate all the code fragments
	apiImportCodeFragment := fmt.Sprintf(`%s "%s"
`, opts.Resource.ImportAlias, opts.Resource.Package)

	addschemeCodeFragment := fmt.Sprintf(`_ = %s.AddToScheme(scheme)
`, opts.Resource.ImportAlias)

	var reconcilerSetupCodeFragment, ctrlImportCodeFragment string

	if opts.Config.MultiGroup {

		ctrlImportCodeFragment = fmt.Sprintf(`%scontroller "%s/controllers/%s"
`, opts.Resource.GroupPackageName, opts.Config.Repo, opts.Resource.Group)

		reconcilerSetupCodeFragment = fmt.Sprintf(`if err = (&%scontroller.%sReconciler{
		Client: mgr.GetClient(),
		Log: ctrl.Log.WithName("controllers").WithName("%s"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "%s")
		os.Exit(1)
	}
`, opts.Resource.GroupPackageName, opts.Resource.Kind, opts.Resource.Kind, opts.Resource.Kind)

	} else {

		ctrlImportCodeFragment = fmt.Sprintf(`"%s/controllers"
`, opts.Config.Repo)

		reconcilerSetupCodeFragment = fmt.Sprintf(`if err = (&controllers.%sReconciler{
		Client: mgr.GetClient(),
		Log: ctrl.Log.WithName("controllers").WithName("%s"),
		Scheme: mgr.GetScheme(),  
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "%s")
		os.Exit(1)
	}
`, opts.Resource.Kind, opts.Resource.Kind, opts.Resource.Kind)

	}

	webhookSetupCodeFragment := fmt.Sprintf(`if err = (&%s.%s{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "%s")
		os.Exit(1)
	}
`, opts.Resource.ImportAlias, opts.Resource.Kind, opts.Resource.Kind)

	if opts.WireResource {
		err := machinery.InsertStringsInFile(path,
			map[string][]string{
				APIPkgImportScaffoldMarker: {apiImportCodeFragment},
				APISchemeScaffoldMarker:    {addschemeCodeFragment},
			})
		if err != nil {
			return err
		}
	}

	if opts.WireController {
		return machinery.InsertStringsInFile(path,
			map[string][]string{
				APIPkgImportScaffoldMarker:    {apiImportCodeFragment, ctrlImportCodeFragment},
				APISchemeScaffoldMarker:       {addschemeCodeFragment},
				ReconcilerSetupScaffoldMarker: {reconcilerSetupCodeFragment},
			})
	}

	if opts.WireWebhook {
		return machinery.InsertStringsInFile(path,
			map[string][]string{
				APIPkgImportScaffoldMarker:    {apiImportCodeFragment, ctrlImportCodeFragment},
				APISchemeScaffoldMarker:       {addschemeCodeFragment},
				ReconcilerSetupScaffoldMarker: {webhookSetupCodeFragment},
			})
	}

	return nil
}

// MainUpdateOptions contains info required for wiring an API/Controller in
// main.go.
type MainUpdateOptions struct {
	// Config contains info about the project
	Config *config.Config

	// Resource is the resource being added
	Resource *resource.Resource

	// Flags to indicate if resource/controller is being scaffolded or not
	WireResource   bool
	WireController bool
	WireWebhook    bool
}

var mainTemplate = fmt.Sprintf(`{{ .Boilerplate }}

package main

import (
	"flag"
	"os"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	%s
)

var (
	scheme = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	%s
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. " +
		"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(func(o *zap.Options) {
		o.Development = true
	}))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		Port:               9443, 
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
`, APIPkgImportScaffoldMarker, APISchemeScaffoldMarker, ReconcilerSetupScaffoldMarker)
