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

package webhooks

import (
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var (
	_ machinery.Template = &WebhookSuite{}
	_ machinery.Inserter = &WebhookSuite{}
)

// WebhookSuite scaffolds the file that sets up the webhook tests
type WebhookSuite struct { //nolint:maligned
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	// todo: currently is not possible to know if an API was or not scaffolded. We can fix it when #1826 be addressed
	WireResource bool

	// K8SVersion define the k8s version used to do the scaffold
	// so that is possible retrieve the binaries
	K8SVersion string

	// BaseDirectoryRelativePath define the Path for the base directory when it is multigroup
	BaseDirectoryRelativePath string

	// Deprecated - The flag should be removed from go/v5
	// IsLegacyPath indicates if webhooks should be scaffolded under the API.
	// Webhooks are now decoupled from APIs based on controller-runtime updates and community feedback.
	// This flag ensures backward compatibility by allowing scaffolding in the legacy/deprecated path.
	IsLegacyPath bool
}

// SetTemplateDefaults implements machinery.Template
func (f *WebhookSuite) SetTemplateDefaults() error {
	if f.Path == "" {
		// Deprecated: Remove me when remove go/v4
		baseDir := "api"
		if !f.IsLegacyPath {
			baseDir = filepath.Join("internal", "webhook")
		}

		if f.MultiGroup && f.Resource.Group != "" {
			f.Path = filepath.Join(baseDir, "%[group]", "%[version]", "webhook_suite_test.go")
		} else {
			f.Path = filepath.Join(baseDir, "%[version]", "webhook_suite_test.go")
		}
	}

	f.Path = f.Resource.Replacer().Replace(f.Path)
	log.Println(f.Path)

	if f.IsLegacyPath {
		f.TemplateBody = fmt.Sprintf(webhookTestSuiteTemplateLegacy,
			machinery.NewMarkerFor(f.Path, importMarker),
			admissionImportAlias,
			machinery.NewMarkerFor(f.Path, addSchemeMarker),
			machinery.NewMarkerFor(f.Path, addWebhookManagerMarker),
			"%s",
			"%d",
		)
	} else {
		f.TemplateBody = fmt.Sprintf(webhookTestSuiteTemplate,
			machinery.NewMarkerFor(f.Path, importMarker),
			f.Resource.ImportAlias(),
			machinery.NewMarkerFor(f.Path, addSchemeMarker),
			machinery.NewMarkerFor(f.Path, addWebhookManagerMarker),
			"%s",
			"%d",
		)
	}

	if f.IsLegacyPath {
		// If is multigroup the path needs to be ../../../ since it has the group dir.
		f.BaseDirectoryRelativePath = `"..", ".."`
		if f.MultiGroup && f.Resource.Group != "" {
			f.BaseDirectoryRelativePath = `"..", "..", ".."`
		}
	} else {
		// If is multigroup the path needs to be ../../../../ since it has the group dir.
		f.BaseDirectoryRelativePath = `"..", "..", ".."`
		if f.MultiGroup && f.Resource.Group != "" {
			f.BaseDirectoryRelativePath = `"..", "..", "..", ".."`
		}
	}

	return nil
}

const (
	admissionImportAlias    = "admissionv1"
	admissionPath           = "k8s.io/api/admission/v1"
	importMarker            = "imports"
	addWebhookManagerMarker = "webhook"
	addSchemeMarker         = "scheme"
)

// GetMarkers implements file.Inserter
func (f *WebhookSuite) GetMarkers() []machinery.Marker {
	return []machinery.Marker{
		machinery.NewMarkerFor(f.Path, importMarker),
		machinery.NewMarkerFor(f.Path, addSchemeMarker),
		machinery.NewMarkerFor(f.Path, addWebhookManagerMarker),
	}
}

const (
	apiImportCodeFragment = `%s "%s"
`

	// Deprecated - TODO: remove for go/v5
	// addWebhookManagerCodeFragmentLegacy is for the path under API
	addWebhookManagerCodeFragmentLegacy = `err = (&%s{}).SetupWebhookWithManager(mgr)
Expect(err).NotTo(HaveOccurred())

`

	addWebhookManagerCodeFragment = `err = Setup%sWebhookWithManager(mgr)
Expect(err).NotTo(HaveOccurred())

`
)

// GetCodeFragments implements file.Inserter
func (f *WebhookSuite) GetCodeFragments() machinery.CodeFragmentsMap {
	fragments := make(machinery.CodeFragmentsMap, 3)

	// Generate import code fragments
	imports := make([]string, 0)

	// Generate add scheme code fragments
	addScheme := make([]string, 0)

	// Generate add webhookManager code fragments
	addWebhookManager := make([]string, 0)
	if f.IsLegacyPath {
		imports = append(imports, fmt.Sprintf(apiImportCodeFragment, admissionImportAlias, admissionPath))
		addWebhookManager = append(addWebhookManager, fmt.Sprintf(addWebhookManagerCodeFragmentLegacy, f.Resource.Kind))
	} else {
		imports = append(imports, fmt.Sprintf(apiImportCodeFragment, f.Resource.ImportAlias(), f.Resource.Path))
		addWebhookManager = append(addWebhookManager, fmt.Sprintf(addWebhookManagerCodeFragment, f.Resource.Kind))
	}

	// Only store code fragments in the map if the slices are non-empty
	if len(addWebhookManager) != 0 {
		fragments[machinery.NewMarkerFor(f.Path, addWebhookManagerMarker)] = addWebhookManager
	}
	if len(imports) != 0 {
		fragments[machinery.NewMarkerFor(f.Path, importMarker)] = imports
	}
	if len(addScheme) != 0 {
		fragments[machinery.NewMarkerFor(f.Path, addSchemeMarker)] = addScheme
	}

	return fragments
}

const webhookTestSuiteTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	%s
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	ctx context.Context
	cancel context.CancelFunc
	k8sClient client.Client
	cfg *rest.Config
	testEnv *envtest.Environment
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Webhook Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	var err error
	err = %s.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	%s

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join({{ .BaseDirectoryRelativePath }}, "config", "crd", "bases")},
		ErrorIfCRDPathMissing: {{ .WireResource }},

		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join({{ .BaseDirectoryRelativePath }}, "config", "webhook")},
		},
	}

	// Retrieve the first found binary directory to allow running tests from IDEs
	if getFirstFoundEnvTestBinaryDir() != "" {
		testEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	}

	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// start webhook server using Manager.
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:               webhookInstallOptions.LocalServingHost,
			Port:               webhookInstallOptions.LocalServingPort,
			CertDir:            webhookInstallOptions.LocalServingCertDir,
		}),
		LeaderElection:     false,
		Metrics:            metricsserver.Options{BindAddress: "0"},

	})
	Expect(err).NotTo(HaveOccurred())

	%s

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	// wait for the webhook server to get ready.
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%s", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}

		return conn.Close();
	}).Should(Succeed())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// getFirstFoundEnvTestBinaryDir locates the first binary in the specified path.
// ENVTEST-based tests depend on specific binaries, usually located in paths set by
// controller-runtime. When running tests directly (e.g., via an IDE) without using
// Makefile targets, the 'BinaryAssetsDirectory' must be explicitly configured.
//
// This function streamlines the process by finding the required binaries, similar to
// setting the 'KUBEBUILDER_ASSETS' environment variable. To ensure the binaries are 
// properly set up, run 'make setup-envtest' beforehand.
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join({{ .BaseDirectoryRelativePath }}, "bin", "k8s")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logf.Log.Error(err, "Failed to read directory", "path", basePath)
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}
`

const webhookTestSuiteTemplateLegacy = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"
	"runtime"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
	%s
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cancel context.CancelFunc
	cfg *rest.Config
	ctx context.Context
	k8sClient client.Client
	testEnv *envtest.Environment
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Webhook Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join({{ .BaseDirectoryRelativePath }}, "config", "crd", "bases")},
		ErrorIfCRDPathMissing: {{ .WireResource }},

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join({{ .BaseDirectoryRelativePath }}, "bin", "k8s",
			fmt.Sprintf("{{ .K8SVersion }}-%%s-%%s", runtime.GOOS, runtime.GOARCH)),

		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join({{ .BaseDirectoryRelativePath }}, "config", "webhook")},
		},
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := apimachineryruntime.NewScheme()
	err = AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = %s.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	%s

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// start webhook server using Manager.
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:               webhookInstallOptions.LocalServingHost,
			Port:               webhookInstallOptions.LocalServingPort,
			CertDir:            webhookInstallOptions.LocalServingCertDir,
		}),
		LeaderElection:     false,
		Metrics:            metricsserver.Options{BindAddress: "0"},

	})
	Expect(err).NotTo(HaveOccurred())

	%s

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	// wait for the webhook server to get ready.
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%s", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}

		return conn.Close();
	}).Should(Succeed())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
`
