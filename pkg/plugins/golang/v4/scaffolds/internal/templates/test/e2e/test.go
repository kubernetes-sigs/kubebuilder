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

package e2e

import (
	"bytes"
	"fmt"
	log "log/slog"
	"os"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var (
	_ machinery.Template = &Test{}
	_ machinery.Inserter = &WebhookTestUpdater{}
)

const webhookChecksMarker = "e2e-webhooks-checks"

// Test defines the basic setup for the e2e test
type Test struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
	machinery.RepositoryMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults set defaults for this template
func (f *Test) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("test", "e2e", "e2e_test.go")
	}

	// This is where the template body is defined with markers
	f.TemplateBody = testCodeTemplate

	return nil
}

// WebhookTestUpdater updates e2e_test.go to insert additional webhook validation tests
type WebhookTestUpdater struct {
	machinery.RepositoryMixin
	machinery.ProjectNameMixin
	machinery.ResourceMixin
	WireWebhook bool
}

// GetPath implements file.Builder
func (*WebhookTestUpdater) GetPath() string {
	return filepath.Join("test", "e2e", "e2e_test.go")
}

// GetIfExistsAction implements file.Builder
func (*WebhookTestUpdater) GetIfExistsAction() machinery.IfExistsAction {
	return machinery.OverwriteFile // Ensures only the marker is replaced
}

// GetMarkers implements file.Inserter
func (f *WebhookTestUpdater) GetMarkers() []machinery.Marker {
	return []machinery.Marker{
		machinery.NewMarkerFor(f.GetPath(), webhookChecksMarker),
	}
}

// GetCodeFragments implements file.Inserter
func (f *WebhookTestUpdater) GetCodeFragments() machinery.CodeFragmentsMap {
	if !f.WireWebhook {
		return nil
	}

	filePath := f.GetPath()

	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Warn("Unable to read file", "file", filePath, "error", err)
		log.Warn("Webhook test code injection will be skipped for this file.")
		log.Warn("This typically occurs when the file was removed and is missing.")
		log.Warn("If you intend to scaffold webhook tests, ensure the file and its markers exist.")
		return nil
	}

	codeFragments := machinery.CodeFragmentsMap{}
	markers := f.GetMarkers()

	for _, marker := range markers {
		if !bytes.Contains(content, []byte(marker.String())) {
			log.Warn("Marker not found in file, skipping webhook test code injection",
				"marker", marker.String(),
				"file_path", filePath)
			continue // skip this marker
		}

		var fragments []string
		fragments = append(fragments, webhookChecksFragment)

		if f.Resource != nil && f.Resource.HasDefaultingWebhook() {
			mutatingWebhookCode := fmt.Sprintf(mutatingWebhookChecksFragment, f.ProjectName)
			fragments = append(fragments, mutatingWebhookCode)
		}

		if f.Resource != nil && f.Resource.HasValidationWebhook() {
			validatingWebhookCode := fmt.Sprintf(validatingWebhookChecksFragment, f.ProjectName)
			fragments = append(fragments, validatingWebhookCode)
		}

		if f.Resource != nil && f.Resource.HasConversionWebhook() {
			conversionWebhookCode := fmt.Sprintf(
				conversionWebhookChecksFragment,
				f.Resource.Kind,
				f.Resource.Plural+"."+f.Resource.Group+"."+f.Resource.Domain,
			)
			fragments = append(fragments, conversionWebhookCode)
		}

		codeFragments[marker] = fragments
	}

	if len(codeFragments) == 0 {
		return nil
	}

	return codeFragments
}

const webhookChecksFragment = `It("should provisioned cert-manager", func() {
	By("validating that cert-manager has the certificate Secret")
	verifyCertManager := func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "secrets", "webhook-server-cert", "-n", namespace)
		_, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
	}
	Eventually(verifyCertManager).Should(Succeed())
})

`

const mutatingWebhookChecksFragment = `It("should have CA injection for mutating webhooks", func() {
	By("checking CA injection for mutating webhooks")
	verifyCAInjection := func(g Gomega) {
		cmd := exec.Command("kubectl", "get",
			"mutatingwebhookconfigurations.admissionregistration.k8s.io",
			"%s-mutating-webhook-configuration",
			"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
		mwhOutput, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(len(mwhOutput)).To(BeNumerically(">", 10))
	}
	Eventually(verifyCAInjection).Should(Succeed())
})

`

const validatingWebhookChecksFragment = `It("should have CA injection for validating webhooks", func() {
	By("checking CA injection for validating webhooks")
	verifyCAInjection := func(g Gomega) {
		cmd := exec.Command("kubectl", "get",
			"validatingwebhookconfigurations.admissionregistration.k8s.io",
			"%s-validating-webhook-configuration",
			"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
		vwhOutput, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(len(vwhOutput)).To(BeNumerically(">", 10))
	}
	Eventually(verifyCAInjection).Should(Succeed())
})

`

const conversionWebhookChecksFragment = `It("should have CA injection for %[1]s conversion webhook", func() {
	By("checking CA injection for %[1]s conversion webhook")
	verifyCAInjection := func(g Gomega) {
		cmd := exec.Command("kubectl", "get",
			"customresourcedefinitions.apiextensions.k8s.io",
			"%[2]s",
			"-o", "go-template={{ .spec.conversion.webhook.clientConfig.caBundle }}")
		vwhOutput, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(len(vwhOutput)).To(BeNumerically(">", 10))
	}
	Eventually(verifyCAInjection).Should(Succeed())
})

`

var testCodeTemplate = `//go:build e2e
// +build e2e

{{ .Boilerplate }}

package e2e

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// namespace where the project is deployed in
const namespace = "{{ .ProjectName }}-system"

// serviceAccountName created for the project
const serviceAccountName = "{{ .ProjectName }}-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "{{ .ProjectName }}-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "{{ .ProjectName }}-metrics-binding"

// Basic project structure tests for bare init projects
var _ = Describe("Project Structure", func() {
	It("should have the required project files", func() {
		By("checking for PROJECT file")
		_, err := os.Stat("PROJECT")
		Expect(err).NotTo(HaveOccurred(), "PROJECT file should exist")

		By("checking for Makefile")
		_, err = os.Stat("Makefile")
		Expect(err).NotTo(HaveOccurred(), "Makefile should exist")

		By("checking for go.mod")
		_, err = os.Stat("go.mod")
		Expect(err).NotTo(HaveOccurred(), "go.mod should exist")

		By("checking for main.go")
		_, err = os.Stat(filepath.Join("cmd", "main.go"))
		Expect(err).NotTo(HaveOccurred(), "cmd/main.go should exist")

		By("checking for config directory")
		_, err = os.Stat("config")
		Expect(err).NotTo(HaveOccurred(), "config directory should exist")
	})
})

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// +kubebuilder:scaffold:e2e-tests

	// The Manager tests will be added here when controllers are created
	_ = controllerPodName // Avoid unused variable warning
})

// +kubebuilder:scaffold:e2e-webhooks-checks

// +kubebuilder:scaffold:e2e-helper-functions
`
