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

package templates

import (
	"fmt"
	"hash/fnv"
	"path/filepath"
	"text/template"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

const defaultMainPath = "main.go"

var _ file.Template = &Main{}
var _ file.UseCustomFuncMap = &Main{}

// Main scaffolds the controller manager entry point
type Main struct {
	file.TemplateMixin
	file.BoilerplateMixin
	file.DomainMixin
	file.RepositoryMixin
}

// SetTemplateDefaults implements file.Template
func (f *Main) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join(defaultMainPath)
	}

	f.TemplateBody = fmt.Sprintf(mainTemplate,
		file.NewMarkerFor(f.Path, importMarker),
		file.NewMarkerFor(f.Path, addSchemeMarker),
		file.NewMarkerFor(f.Path, setupMarker),
	)

	return nil
}

func hash(s string) (string, error) {
	hasher := fnv.New32a()
	hasher.Write([]byte(s)) // nolint:errcheck
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// GetFuncMap implements file.UseCustomFuncMap
func (f *Main) GetFuncMap() template.FuncMap {
	fm := file.DefaultFuncMap()
	fm["hash"] = hash
	return fm
}

var _ file.Inserter = &MainUpdater{}

// MainUpdater updates main.go to run Controllers
type MainUpdater struct { //nolint:maligned
	file.RepositoryMixin
	file.MultiGroupMixin
	file.ResourceMixin

	// Flags to indicate which parts need to be included when updating the file
	WireResource, WireController, WireWebhook bool
}

// GetPath implements Builder
func (*MainUpdater) GetPath() string {
	return defaultMainPath
}

// GetIfExistsAction implements Builder
func (*MainUpdater) GetIfExistsAction() file.IfExistsAction {
	return file.Overwrite
}

const (
	importMarker    = "imports"
	addSchemeMarker = "scheme"
	setupMarker     = "builder"
)

// GetMarkers implements file.Inserter
func (f *MainUpdater) GetMarkers() []file.Marker {
	return []file.Marker{
		file.NewMarkerFor(defaultMainPath, importMarker),
		file.NewMarkerFor(defaultMainPath, addSchemeMarker),
		file.NewMarkerFor(defaultMainPath, setupMarker),
	}
}

const (
	apiImportCodeFragment = `%s "%s"
`
	controllerImportCodeFragment = `"%s/controllers"
`
	// TODO(v3): `&%scontrollers` should be used instead of `&%scontroller` as there may be multiple
	//  controller for different Kinds in the same group. However, this is a backwards incompatible
	//  change, and thus should be done for next project version.
	multiGroupControllerImportCodeFragment = `%scontroller "%s/controllers/%s"
`
	addschemeCodeFragment = `utilruntime.Must(%s.AddToScheme(scheme))
`
	reconcilerSetupCodeFragment = `if err = (&controllers.%sReconciler{
		Client: mgr.GetClient(),
		Log: ctrl.Log.WithName("controllers").WithName("%s"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "%s")
		os.Exit(1)
	}
`
	// TODO(v3): loggers for the same Kind controllers from different groups use the same logger.
	//  `.WithName("controllers").WithName(GROUP).WithName(KIND)` should be used instead. However,
	//  this is a backwards incompatible change, and thus should be done for next project version.
	multiGroupReconcilerSetupCodeFragment = `if err = (&%scontroller.%sReconciler{
		Client: mgr.GetClient(),
		Log: ctrl.Log.WithName("controllers").WithName("%s"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "%s")
		os.Exit(1)
	}
`
	webhookSetupCodeFragment = `if err = (&%s.%s{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "%s")
		os.Exit(1)
	}
`
)

// GetCodeFragments implements file.Inserter
func (f *MainUpdater) GetCodeFragments() file.CodeFragmentsMap {
	fragments := make(file.CodeFragmentsMap, 3)

	// If resource is not being provided we are creating the file, not updating it
	if f.Resource == nil {
		return fragments
	}

	// Generate import code fragments
	imports := make([]string, 0)
	imports = append(imports, fmt.Sprintf(apiImportCodeFragment, f.Resource.ImportAlias, f.Resource.Package))
	if f.WireController {
		if !f.MultiGroup {
			imports = append(imports, fmt.Sprintf(controllerImportCodeFragment, f.Repo))
		} else {
			imports = append(imports, fmt.Sprintf(multiGroupControllerImportCodeFragment,
				f.Resource.GroupPackageName, f.Repo, f.Resource.Group))
		}
	}

	// Generate add scheme code fragments
	addScheme := make([]string, 0)
	addScheme = append(addScheme, fmt.Sprintf(addschemeCodeFragment, f.Resource.ImportAlias))

	// Generate setup code fragments
	setup := make([]string, 0)
	if f.WireController {
		if !f.MultiGroup {
			setup = append(setup, fmt.Sprintf(reconcilerSetupCodeFragment,
				f.Resource.Kind, f.Resource.Kind, f.Resource.Kind))
		} else {
			setup = append(setup, fmt.Sprintf(multiGroupReconcilerSetupCodeFragment,
				f.Resource.GroupPackageName, f.Resource.Kind, f.Resource.Kind, f.Resource.Kind))
		}
	}
	if f.WireWebhook {
		setup = append(setup, fmt.Sprintf(webhookSetupCodeFragment,
			f.Resource.ImportAlias, f.Resource.Kind, f.Resource.Kind))
	}

	// Only store code fragments in the map if the slices are non-empty
	if len(imports) != 0 {
		fragments[file.NewMarkerFor(defaultMainPath, importMarker)] = imports
	}
	if len(addScheme) != 0 {
		fragments[file.NewMarkerFor(defaultMainPath, addSchemeMarker)] = addScheme
	}
	if len(setup) != 0 {
		fragments[file.NewMarkerFor(defaultMainPath, setupMarker)] = setup
	}

	return fragments
}

var mainTemplate = `{{ .Boilerplate }}

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
	%s
)

var (
	scheme = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

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

	ctrl.SetLogger(zap.New(zap.UseDevMode(true))) 

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443, 
		LeaderElection:     enableLeaderElection, 
		LeaderElectionID:   "{{ hash .Repo }}.{{ .Domain }}",
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
`
