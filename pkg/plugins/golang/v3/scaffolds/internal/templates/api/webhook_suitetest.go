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

package api

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &WebhookSuite{}
var _ machinery.Inserter = &WebhookSuite{}

// WebhookSuite scaffolds the file that sets up the webhook tests
type WebhookSuite struct { //nolint:maligned
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	// todo: currently is not possible to know if an API was or not scaffolded. We can fix it when #1826 be addressed
	WireResource bool

	// BaseDirectoryRelativePath define the Path for the base directory when it is multigroup
	BaseDirectoryRelativePath string
}

// SetTemplateDefaults implements file.Template
func (f *WebhookSuite) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup {
			if f.Resource.Group != "" {
				f.Path = filepath.Join("apis", "%[group]", "%[version]", "webhook_suite_test.go")
			} else {
				f.Path = filepath.Join("apis", "%[version]", "webhook_suite_test.go")
			}
		} else {
			f.Path = filepath.Join("api", "%[version]", "webhook_suite_test.go")
		}
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.TemplateBody = fmt.Sprintf(webhookTestSuiteTemplate,
		machinery.NewMarkerFor(f.Path, importMarker),
		admissionImportAlias,
		machinery.NewMarkerFor(f.Path, addSchemeMarker),
		machinery.NewMarkerFor(f.Path, addWebhookManagerMarker),
		"%s",
		"%d",
	)

	// If is multigroup the path needs to be ../../.. since it has the group dir.
	f.BaseDirectoryRelativePath = `"..", ".."`
	if f.MultiGroup && f.Resource.Group != "" {
		f.BaseDirectoryRelativePath = `"..", "..",".."`
	}

	return nil
}

const (
	// TODO: admission webhook versions should be based on the input of the user. For More Info #1664
	admissionImportAlias    = "admissionv1beta1"
	admissionPath           = "k8s.io/api/admission/v1beta1"
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

	addWebhookManagerCodeFragment = `err = (&%s{}).SetupWebhookWithManager(mgr)
Expect(err).NotTo(HaveOccurred())

`
)

// GetCodeFragments implements file.Inserter
func (f *WebhookSuite) GetCodeFragments() machinery.CodeFragmentsMap {
	fragments := make(machinery.CodeFragmentsMap, 3)

	// Generate import code fragments
	imports := make([]string, 0)
	imports = append(imports, fmt.Sprintf(apiImportCodeFragment, admissionImportAlias, admissionPath))

	// Generate add scheme code fragments
	addScheme := make([]string, 0)

	// Generate add webhookManager code fragments
	addWebhookManager := make([]string, 0)
	addWebhookManager = append(addWebhookManager, fmt.Sprintf(addWebhookManagerCodeFragment, f.Resource.Kind))

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
	"path/filepath"
	"testing"
	"fmt"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
	%s
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

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
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join({{ .BaseDirectoryRelativePath }}, "config", "webhook")},
		},
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := runtime.NewScheme()
	err = AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = %s.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	%s

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// start webhook server using Manager
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme,
		Host:               webhookInstallOptions.LocalServingHost,
		Port:               webhookInstallOptions.LocalServingPort,
		CertDir:            webhookInstallOptions.LocalServingCertDir,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	%s

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	// wait for the webhook server to get ready
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%s", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}).Should(Succeed())

})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
`
