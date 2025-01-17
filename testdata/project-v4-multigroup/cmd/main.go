/*
Copyright 2025 The Kubernetes authors.

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
	"crypto/tls"
	"flag"
	"os"
	"path/filepath"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"

	crewv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/crew/v1"
	examplecomv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/example.com/v1"
	examplecomv1alpha1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/example.com/v1alpha1"
	examplecomv2 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/example.com/v2"
	fizv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/fiz/v1"
	foopolicyv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/foo.policy/v1"
	foov1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/foo/v1"
	seacreaturesv1beta1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/sea-creatures/v1beta1"
	seacreaturesv1beta2 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/sea-creatures/v1beta2"
	shipv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/ship/v1"
	shipv1beta1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/ship/v1beta1"
	shipv2alpha1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/ship/v2alpha1"
	appscontroller "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/internal/controller/apps"
	certmanagercontroller "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/internal/controller/cert-manager"
	crewcontroller "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/internal/controller/crew"
	examplecomcontroller "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/internal/controller/example.com"
	fizcontroller "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/internal/controller/fiz"
	foocontroller "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/internal/controller/foo"
	foopolicycontroller "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/internal/controller/foo.policy"
	seacreaturescontroller "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/internal/controller/sea-creatures"
	shipcontroller "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/internal/controller/ship"
	webhookappsv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/internal/webhook/apps/v1"
	webhookcertmanagerv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/internal/webhook/cert-manager/v1"
	webhookcorev1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/internal/webhook/core/v1"
	webhookcrewv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/internal/webhook/crew/v1"
	webhookexamplecomv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/internal/webhook/example.com/v1"
	webhookexamplecomv1alpha1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/internal/webhook/example.com/v1alpha1"
	webhookshipv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/internal/webhook/ship/v1"
	webhookshipv2alpha1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/internal/webhook/ship/v2alpha1"
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
	utilruntime.Must(foov1.AddToScheme(scheme))
	utilruntime.Must(fizv1.AddToScheme(scheme))
	utilruntime.Must(certmanagerv1.AddToScheme(scheme))
	utilruntime.Must(examplecomv1alpha1.AddToScheme(scheme))
	utilruntime.Must(examplecomv1.AddToScheme(scheme))
	utilruntime.Must(examplecomv2.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

// nolint:gocyclo
func main() {
	var metricsAddr string
	var metricsCertPath, metricsCertName, metricsCertKey string
	var webhookCertPath, webhookCertName, webhookCertKey string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var tlsOpts []func(*tls.Config)
	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.StringVar(&webhookCertPath, "webhook-cert-path", "", "The directory that contains the webhook certificate.")
	flag.StringVar(&webhookCertName, "webhook-cert-name", "tls.crt", "The name of the webhook certificate file.")
	flag.StringVar(&webhookCertKey, "webhook-cert-key", "tls.key", "The name of the webhook key file.")
	flag.StringVar(&metricsCertPath, "metrics-cert-path", "",
		"The directory that contains the metrics server certificate.")
	flag.StringVar(&metricsCertName, "metrics-cert-name", "tls.crt", "The name of the metrics server certificate file.")
	flag.StringVar(&metricsCertKey, "metrics-cert-key", "tls.key", "The name of the metrics server key file.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

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

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	// Create watchers for metrics and webhooks certificates
	var metricsCertWatcher, webhookCertWatcher *certwatcher.CertWatcher

	// Initial webhook TLS options
	webhookTLSOpts := tlsOpts

	if len(webhookCertPath) > 0 {
		setupLog.Info("Initializing webhook certificate watcher using provided certificates",
			"webhook-cert-path", webhookCertPath, "webhook-cert-name", webhookCertName, "webhook-cert-key", webhookCertKey)

		var err error
		webhookCertWatcher, err = certwatcher.New(
			filepath.Join(webhookCertPath, webhookCertName),
			filepath.Join(webhookCertPath, webhookCertKey),
		)
		if err != nil {
			setupLog.Error(err, "Failed to initialize webhook certificate watcher")
			os.Exit(1)
		}

		webhookTLSOpts = append(webhookTLSOpts, func(config *tls.Config) {
			config.GetCertificate = webhookCertWatcher.GetCertificate
		})
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: webhookTLSOpts,
	})

	// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.0/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.0/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	// If the certificate is not specified, controller-runtime will automatically
	// generate self-signed certificates for the metrics server. While convenient for development and testing,
	// this setup is not recommended for production.
	//
	// TODO(user): If you enable certManager, uncomment the following lines:
	// - [METRICS-WITH-CERTS] at config/default/kustomization.yaml to generate and use certificates
	// managed by cert-manager for the metrics server.
	// - [PROMETHEUS-WITH-CERTS] at config/prometheus/kustomization.yaml for TLS certification.
	if len(metricsCertPath) > 0 {
		setupLog.Info("Initializing metrics certificate watcher using provided certificates",
			"metrics-cert-path", metricsCertPath, "metrics-cert-name", metricsCertName, "metrics-cert-key", metricsCertKey)

		var err error
		metricsCertWatcher, err = certwatcher.New(
			filepath.Join(metricsCertPath, metricsCertName),
			filepath.Join(metricsCertPath, metricsCertKey),
		)
		if err != nil {
			setupLog.Error(err, "to initialize metrics certificate watcher", "error", err)
			os.Exit(1)
		}

		metricsServerOptions.TLSOpts = append(metricsServerOptions.TLSOpts, func(config *tls.Config) {
			config.GetCertificate = metricsCertWatcher.GetCertificate
		})
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "3e9f67a9.testproject.org",
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
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&crewcontroller.CaptainReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Captain")
		os.Exit(1)
	}
	// nolint:goconst
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = webhookcrewv1.SetupCaptainWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Captain")
			os.Exit(1)
		}
	}
	if err = (&shipcontroller.FrigateReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Frigate")
		os.Exit(1)
	}
	if err = (&shipcontroller.DestroyerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Destroyer")
		os.Exit(1)
	}
	// nolint:goconst
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = webhookshipv1.SetupDestroyerWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Destroyer")
			os.Exit(1)
		}
	}
	if err = (&shipcontroller.CruiserReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Cruiser")
		os.Exit(1)
	}
	// nolint:goconst
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = webhookshipv2alpha1.SetupCruiserWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Cruiser")
			os.Exit(1)
		}
	}
	if err = (&seacreaturescontroller.KrakenReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Kraken")
		os.Exit(1)
	}
	if err = (&seacreaturescontroller.LeviathanReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Leviathan")
		os.Exit(1)
	}
	if err = (&foopolicycontroller.HealthCheckPolicyReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HealthCheckPolicy")
		os.Exit(1)
	}
	if err = (&appscontroller.DeploymentReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Deployment")
		os.Exit(1)
	}
	if err = (&foocontroller.BarReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Bar")
		os.Exit(1)
	}
	if err = (&fizcontroller.BarReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Bar")
		os.Exit(1)
	}
	if err = (&certmanagercontroller.CertificateReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Certificate")
		os.Exit(1)
	}
	// nolint:goconst
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = webhookcertmanagerv1.SetupIssuerWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Issuer")
			os.Exit(1)
		}
	}
	// nolint:goconst
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = webhookcorev1.SetupPodWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Pod")
			os.Exit(1)
		}
	}
	// nolint:goconst
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = webhookappsv1.SetupDeploymentWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Deployment")
			os.Exit(1)
		}
	}
	if err = (&examplecomcontroller.MemcachedReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("memcached-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Memcached")
		os.Exit(1)
	}
	if err = (&examplecomcontroller.BusyboxReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("busybox-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Busybox")
		os.Exit(1)
	}
	// nolint:goconst
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = webhookexamplecomv1alpha1.SetupMemcachedWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Memcached")
			os.Exit(1)
		}
	}
	if err = (&examplecomcontroller.WordpressReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Wordpress")
		os.Exit(1)
	}
	// nolint:goconst
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = webhookexamplecomv1.SetupWordpressWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Wordpress")
			os.Exit(1)
		}
	}
	// +kubebuilder:scaffold:builder

	if metricsCertWatcher != nil {
		setupLog.Info("Adding metrics certificate watcher to manager")
		if err := mgr.Add(metricsCertWatcher); err != nil {
			setupLog.Error(err, "unable to add metrics certificate watcher to manager")
			os.Exit(1)
		}
	}

	if webhookCertWatcher != nil {
		setupLog.Info("Adding webhook certificate watcher to manager")
		if err := mgr.Add(webhookCertWatcher); err != nil {
			setupLog.Error(err, "unable to add webhook certificate watcher to manager")
			os.Exit(1)
		}
	}

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
