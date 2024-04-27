/*
Copyright 2022 The Kubernetes Authors.

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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

const defaultMainPath = "cmd/main.go"

var _ machinery.Template = &Main{}

// Main scaffolds a file that defines the controller manager entry point
type Main struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
	machinery.DomainMixin
	machinery.RepositoryMixin
	machinery.ComponentConfigMixin
}

// SetTemplateDefaults implements file.Template
func (f *Main) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join(defaultMainPath)
	}

	f.TemplateBody = fmt.Sprintf(mainTemplate,
		machinery.NewMarkerFor(f.Path, importMarker),
		machinery.NewMarkerFor(f.Path, addSchemeMarker),
		machinery.NewMarkerFor(f.Path, setupMarker),
	)

	return nil
}

var _ machinery.Inserter = &MainUpdater{}

// MainUpdater updates cmd/main.go to run Controllers
type MainUpdater struct { //nolint:maligned
	machinery.RepositoryMixin
	machinery.MultiGroupMixin
	machinery.ResourceMixin

	// Flags to indicate which parts need to be included when updating the file
	WireResource, WireController, WireWebhook bool
}

// GetPath implements file.Builder
func (*MainUpdater) GetPath() string {
	return defaultMainPath
}

// GetIfExistsAction implements file.Builder
func (*MainUpdater) GetIfExistsAction() machinery.IfExistsAction {
	return machinery.OverwriteFile
}

const (
	importMarker    = "imports"
	addSchemeMarker = "scheme"
	setupMarker     = "builder"
)

// GetMarkers implements file.Inserter
func (f *MainUpdater) GetMarkers() []machinery.Marker {
	return []machinery.Marker{
		machinery.NewMarkerFor(defaultMainPath, importMarker),
		machinery.NewMarkerFor(defaultMainPath, addSchemeMarker),
		machinery.NewMarkerFor(defaultMainPath, setupMarker),
	}
}

const (
	apiImportCodeFragment = `%s "%s"
`
	controllerImportCodeFragment = `"%s/internal/controller"
`
	multiGroupControllerImportCodeFragment = `%scontroller "%s/internal/controller/%s"
`
	addschemeCodeFragment = `utilruntime.Must(%s.AddToScheme(scheme))
`
	reconcilerSetupCodeFragment = `if err = (&controller.%sReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "%s")
		os.Exit(1)
	}
`
	multiGroupReconcilerSetupCodeFragment = `if err = (&%scontroller.%sReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "%s")
		os.Exit(1)
	}
`
	webhookSetupCodeFragment = `if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&%s.%s{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "%s")
			os.Exit(1)
		}
	}
`
)

// GetCodeFragments implements file.Inserter
func (f *MainUpdater) GetCodeFragments() machinery.CodeFragmentsMap {
	fragments := make(machinery.CodeFragmentsMap, 3)

	// If resource is not being provided we are creating the file, not updating it
	if f.Resource == nil {
		return fragments
	}

	// Generate import code fragments
	imports := make([]string, 0)
	if f.WireResource {
		imports = append(imports, fmt.Sprintf(apiImportCodeFragment, f.Resource.ImportAlias(), f.Resource.Path))
	}

	if f.WireController {
		if !f.MultiGroup || f.Resource.Group == "" {
			imports = append(imports, fmt.Sprintf(controllerImportCodeFragment, f.Repo))
		} else {
			imports = append(imports, fmt.Sprintf(multiGroupControllerImportCodeFragment,
				f.Resource.PackageName(), f.Repo, f.Resource.Group))
		}
	}

	// Generate add scheme code fragments
	addScheme := make([]string, 0)
	if f.WireResource {
		addScheme = append(addScheme, fmt.Sprintf(addschemeCodeFragment, f.Resource.ImportAlias()))
	}

	// Generate setup code fragments
	setup := make([]string, 0)
	if f.WireController {
		if !f.MultiGroup || f.Resource.Group == "" {
			setup = append(setup, fmt.Sprintf(reconcilerSetupCodeFragment,
				f.Resource.Kind, f.Resource.Kind))
		} else {
			setup = append(setup, fmt.Sprintf(multiGroupReconcilerSetupCodeFragment,
				f.Resource.PackageName(), f.Resource.Kind, f.Resource.Kind))
		}
	}
	if f.WireWebhook {
		setup = append(setup, fmt.Sprintf(webhookSetupCodeFragment,
			f.Resource.ImportAlias(), f.Resource.Kind, f.Resource.Kind))
	}

	// Only store code fragments in the map if the slices are non-empty
	if len(imports) != 0 {
		fragments[machinery.NewMarkerFor(defaultMainPath, importMarker)] = imports
	}
	if len(addScheme) != 0 {
		fragments[machinery.NewMarkerFor(defaultMainPath, addSchemeMarker)] = addScheme
	}
	if len(setup) != 0 {
		fragments[machinery.NewMarkerFor(defaultMainPath, setupMarker)] = setup
	}

	return fragments
}

var mainTemplate = `{{ .Boilerplate }}

package main

import (
	"crypto/tls"
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
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
{{- if not .ComponentConfig }}
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. " +
		"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", false, 
		"If set the metrics endpoint is served securely")
	flag.BoolVar(&enableHTTP2, "enable-http2", false, 
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
{{- else }}
  var configFile string
	flag.StringVar(&configFile, "config", "", 
		"The controller will load its initial configuration from this file. " +
		"Omit this flag to use the default configuration values. " +
		"Command-line flags override configuration from this file.")
{{- end }}
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

{{ if not .ComponentConfig }}
	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and 
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	tlsOpts := []func(*tls.Config){}
	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{
			BindAddress: metricsAddr,
			SecureServing: secureMetrics,
			TLSOpts: tlsOpts,
		},
		WebhookServer: webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		{{- if not .Domain }}
		LeaderElectionID:        "{{ hashFNV .Repo }}",
		{{- else }}
		LeaderElectionID:        "{{ hashFNV .Repo }}.{{ .Domain }}",
		{{- end }}
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after 
		// the manager stops, so would be fine to enable this option. However, 
		// if you are doing or is intended to do any operation such as perform cleanups 
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
{{- else }}
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
{{- end }}
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	%s

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
`
