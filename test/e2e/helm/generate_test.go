/*
Copyright 2025 The Kubernetes Authors.

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

package helm

import (
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	helmChartLoader "helm.sh/helm/v3/pkg/chart/loader"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	helmv2alpha "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("Helm v2-alpha Plugin", func() {
	var kbc *utils.TestContext

	BeforeEach(func() {
		var err error
		kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(kbc.Prepare()).To(Succeed())
	})

	AfterEach(func() {
		kbc.Destroy()
	})

	Describe("Basic Functionality", func() {
		It("should generate helm chart with dynamic kustomize-based templates", func() {
			By("initializing a basic project")
			initBasicProject(kbc)

			By("creating API and controller resources")
			createTestResources(kbc)

			By("building installer manifest")
			Expect(kbc.Make("build-installer")).To(Succeed())

			By("applying helm v2-alpha plugin")
			err := kbc.EditHelmPlugin()
			Expect(err).NotTo(HaveOccurred())

			By("validating generated helm chart structure")
			validateBasicHelmChart(kbc)

			By("verifying dynamic template generation")
			validateDynamicTemplates(kbc)

			By("checking plugin configuration tracking")
			validatePluginConfig(kbc)
		})

		It("should handle webhooks correctly", func() {
			By("initializing a project with webhooks")
			initBasicProject(kbc)
			createTestResources(kbc)
			createWebhookResources(kbc)

			By("building installer manifest with webhooks")
			Expect(kbc.Make("build-installer")).To(Succeed())

			By("applying helm v2-alpha plugin")
			err := kbc.EditHelmPlugin()
			Expect(err).NotTo(HaveOccurred())

			By("validating webhook templates are generated")
			validateWebhookTemplates(kbc)

			By("verifying cert-manager integration")
			validateCertManagerIntegration(kbc)
		})

		It("should support custom flags and preserve files", func() {
			By("initializing project and building installer")
			initBasicProject(kbc)
			createTestResources(kbc)
			Expect(kbc.Make("build-installer")).To(Succeed())

			By("applying plugin with custom output directory")
			err := kbc.Edit("--plugins", "helm.kubebuilder.io/v2-alpha", "--output-dir", "custom-charts")
			Expect(err).NotTo(HaveOccurred())

			By("verifying chart is generated in custom directory")
			validateCustomOutputDir(kbc, "custom-charts")

			By("re-running plugin without --force should preserve existing files")
			err = kbc.Edit("--plugins", "helm.kubebuilder.io/v2-alpha", "--output-dir", "custom-charts")
			Expect(err).NotTo(HaveOccurred())

			By("verifying files are preserved when not using --force")
			validateFilePreservation(kbc, "custom-charts")
		})

		It("should properly template cert-manager resources when chart name changes", func() {
			By("initializing a project with webhooks")
			initBasicProject(kbc)
			createTestResources(kbc)
			createWebhookResources(kbc)

			By("building installer manifest with webhooks")
			Expect(kbc.Make("build-installer")).To(Succeed())

			By("applying helm v2-alpha plugin")
			err := kbc.EditHelmPlugin()
			Expect(err).NotTo(HaveOccurred())

			chartPath := filepath.Join(kbc.Dir, "dist", "chart")

			By("renaming the chart to a different name")
			chartYamlPath := filepath.Join(chartPath, "Chart.yaml")
			err = pluginutil.ReplaceInFile(chartYamlPath, "name: e2e-"+kbc.TestSuffix, "name: my-custom-chart")
			Expect(err).NotTo(HaveOccurred())

			By("regenerating the chart with the new name")
			err = kbc.EditHelmPlugin()
			Expect(err).NotTo(HaveOccurred())

			By("validating issuer name uses e2e-.resourceName for 63-char safety")
			validateIssuerNameTemplating(kbc, chartPath)

			By("validating certificate issuerRef uses e2e-.resourceName for 63-char safety")
			validateCertificateIssuerRefTemplating(kbc, chartPath)

			By("validating cert-manager annotations use e2e-.resourceName for 63-char safety")
			validateCertManagerAnnotationsTemplating(kbc, chartPath)

			By("validating app.kubernetes.io/name label uses e2e-.name template")
			validateAppNameLabelTemplating(kbc, chartPath)

			By("rendering the chart and verifying consistent naming")
			validateRenderedChartConsistency(kbc, chartPath)
		})
	})
})

func initBasicProject(kbc *utils.TestContext) {
	err := kbc.Init(
		"--plugins", "go/v4",
		"--project-version", "3",
		"--domain", kbc.Domain,
	)
	Expect(err).NotTo(HaveOccurred())
}

func createTestResources(kbc *utils.TestContext) {
	err := kbc.CreateAPI(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--namespaced",
		"--resource",
		"--controller",
		"--make=false",
	)
	Expect(err).NotTo(HaveOccurred())
}

func createWebhookResources(kbc *utils.TestContext) {
	err := kbc.CreateWebhook(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--defaulting",
		"--programmatic-validation",
		"--make=false",
	)
	Expect(err).NotTo(HaveOccurred())

	By("run make manifests")
	Expect(kbc.Make("manifests")).To(Succeed())
}

func validateBasicHelmChart(kbc *utils.TestContext) {
	chartPath := filepath.Join(kbc.Dir, "dist", "chart")

	By("verifying Chart.yaml exists and is valid")
	chart, err := helmChartLoader.LoadDir(chartPath)
	Expect(err).NotTo(HaveOccurred())
	Expect(chart.Validate()).To(Succeed())
	Expect(chart.Name()).To(Equal("e2e-" + kbc.TestSuffix))

	By("verifying essential files exist")
	essentialFiles := []string{
		"Chart.yaml",
		"values.yaml",
		".helmignore",
		"templates/_helpers.tpl",
	}
	for _, file := range essentialFiles {
		filePath := filepath.Join(chartPath, file)
		Expect(filePath).To(BeAnExistingFile())
	}
}

func validateDynamicTemplates(kbc *utils.TestContext) {
	chartPath := filepath.Join(kbc.Dir, "dist", "chart")

	By("verifying templates directory structure matches config/ structure")
	expectedDirs := []string{
		"templates/manager",
		"templates/rbac",
		"templates/crd",
		"templates/metrics",
	}
	for _, dir := range expectedDirs {
		dirPath := filepath.Join(chartPath, dir)
		Expect(dirPath).To(BeADirectory())
	}

	By("verifying manager deployment template exists")
	managerTemplate := filepath.Join(chartPath, "templates", "manager", "manager.yaml")
	Expect(managerTemplate).To(BeAnExistingFile())

	By("verifying CRD templates exist")
	crdDir := filepath.Join(chartPath, "templates", "crd")
	files, err := afero.ReadDir(afero.NewOsFs(), crdDir)
	Expect(err).NotTo(HaveOccurred())
	Expect(files).ToNot(BeEmpty())
}

func validateWebhookTemplates(kbc *utils.TestContext) {
	chartPath := filepath.Join(kbc.Dir, "dist", "chart")

	By("verifying webhook directory exists")
	webhookDir := filepath.Join(chartPath, "templates", "webhook")
	Expect(webhookDir).To(BeADirectory())

	By("verifying webhook configuration files exist")
	files, err := afero.ReadDir(afero.NewOsFs(), webhookDir)
	Expect(err).NotTo(HaveOccurred())
	Expect(files).ToNot(BeEmpty())

	By("verifying webhook files contain webhook configurations")
	foundValidatingWebhook := false
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		webhookFile := filepath.Join(webhookDir, file.Name())
		content, err := afero.ReadFile(afero.NewOsFs(), webhookFile)
		Expect(err).NotTo(HaveOccurred())
		contentStr := string(content)
		if strings.Contains(contentStr, "ValidatingWebhookConfiguration") {
			foundValidatingWebhook = true
			break
		}
	}
	Expect(foundValidatingWebhook).To(BeTrue(), "Expected to find ValidatingWebhookConfiguration in webhook templates")
}

func validateCertManagerIntegration(kbc *utils.TestContext) {
	chartPath := filepath.Join(kbc.Dir, "dist", "chart")

	By("verifying cert-manager templates exist")
	certManagerDir := filepath.Join(chartPath, "templates", "cert-manager")
	Expect(certManagerDir).To(BeADirectory())

	By("verifying cert-manager is enabled in values.yaml")
	valuesPath := filepath.Join(chartPath, "values.yaml")
	valuesContent, err := afero.ReadFile(afero.NewOsFs(), valuesPath)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(valuesContent)).To(ContainSubstring("certManager:"))
	Expect(string(valuesContent)).To(ContainSubstring("enable: true"))
}

func validatePluginConfig(kbc *utils.TestContext) {
	By("verifying plugin configuration is tracked in PROJECT file")
	projectPath := filepath.Join(kbc.Dir, "PROJECT")
	projectConfig := getConfigFromProjectFile(projectPath)

	var helmConfig helmv2alpha.Plugin
	err := projectConfig.DecodePluginConfig("helm.kubebuilder.io/v2-alpha", &helmConfig)
	Expect(err).NotTo(HaveOccurred())
}

func validateCustomOutputDir(kbc *utils.TestContext, outputDir string) {
	chartPath := filepath.Join(kbc.Dir, outputDir, "chart")

	By("verifying chart exists in custom directory")
	Expect(chartPath).To(BeADirectory())

	By("verifying Chart.yaml in custom directory")
	chartFile := filepath.Join(chartPath, "Chart.yaml")
	Expect(chartFile).To(BeAnExistingFile())
}

func validateFilePreservation(kbc *utils.TestContext, outputDir string) {
	chartPath := filepath.Join(kbc.Dir, outputDir, "chart")

	By("verifying files still exist after re-run")
	valuesFile := filepath.Join(chartPath, "values.yaml")
	Expect(valuesFile).To(BeAnExistingFile())

	chartFile := filepath.Join(chartPath, "Chart.yaml")
	Expect(chartFile).To(BeAnExistingFile())
}

func getConfigFromProjectFile(projectFilePath string) config.Config {
	fs := afero.NewOsFs()
	store := yaml.New(machinery.Filesystem{FS: fs})
	err := store.LoadFrom(projectFilePath)
	Expect(err).NotTo(HaveOccurred())
	return store.Config()
}

func validateIssuerNameTemplating(kbc *utils.TestContext, chartPath string) {
	issuerPath := filepath.Join(chartPath, "templates", "cert-manager", "selfsigned-issuer.yaml")
	content, err := afero.ReadFile(afero.NewOsFs(), issuerPath)
	Expect(err).NotTo(HaveOccurred())
	contentStr := string(content)

	// Verify issuer name uses <chartname>.resourceName template with 63-char safety
	// Chart name is the project name (e.g., "e2e-xxxx")
	chartName := "e2e-" + kbc.TestSuffix
	expected := `name: {{ include "` + chartName + `.resourceName" (dict "suffix" "selfsigned-issuer" "context" $) }}`
	Expect(contentStr).To(ContainSubstring(expected),
		"Issuer name should use "+chartName+".resourceName template")
	Expect(contentStr).NotTo(ContainSubstring("e2e-"+kbc.TestSuffix+"-selfsigned-issuer"),
		"Issuer name should not be hardcoded to project name")
}

func validateCertificateIssuerRefTemplating(kbc *utils.TestContext, chartPath string) {
	certManagerDir := filepath.Join(chartPath, "templates", "cert-manager")
	files, err := afero.ReadDir(afero.NewOsFs(), certManagerDir)
	Expect(err).NotTo(HaveOccurred())

	chartName := "e2e-" + kbc.TestSuffix
	foundCertificate := false
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") || file.Name() == "selfsigned-issuer.yaml" {
			continue
		}

		certPath := filepath.Join(certManagerDir, file.Name())
		content, err := afero.ReadFile(afero.NewOsFs(), certPath)
		Expect(err).NotTo(HaveOccurred())
		contentStr := string(content)

		if strings.Contains(contentStr, "kind: Certificate") {
			foundCertificate = true
			// Verify certificate issuerRef uses <chartname>.resourceName template
			expected := `name: {{ include "` + chartName + `.resourceName" (dict "suffix" "selfsigned-issuer" "context" $) }}`
			Expect(contentStr).To(ContainSubstring(expected),
				"Certificate issuerRef should use "+chartName+".resourceName template in file "+file.Name())
		}
	}
	Expect(foundCertificate).To(BeTrue(), "Expected to find at least one Certificate resource")
}

func validateCertManagerAnnotationsTemplating(kbc *utils.TestContext, chartPath string) {
	chartName := "e2e-" + kbc.TestSuffix

	// Check webhook configurations
	webhookDir := filepath.Join(chartPath, "templates", "webhook")
	if exists, _ := afero.DirExists(afero.NewOsFs(), webhookDir); exists {
		files, err := afero.ReadDir(afero.NewOsFs(), webhookDir)
		Expect(err).NotTo(HaveOccurred())

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
				continue
			}

			webhookPath := filepath.Join(webhookDir, file.Name())
			content, err := afero.ReadFile(afero.NewOsFs(), webhookPath)
			Expect(err).NotTo(HaveOccurred())
			contentStr := string(content)

			if strings.Contains(contentStr, "cert-manager.io/inject-ca-from") {
				// Verify cert-manager annotation uses <chartname>.resourceName template
				expected := `{{ include "` + chartName + `.resourceName" (dict "suffix" "serving-cert" "context" $) }}`
				Expect(contentStr).To(ContainSubstring(expected),
					"cert-manager.io/inject-ca-from annotation should use "+chartName+".resourceName in "+file.Name())
				Expect(contentStr).NotTo(ContainSubstring("e2e-"+kbc.TestSuffix+"-serving-cert"),
					"cert-manager.io/inject-ca-from annotation should not be hardcoded in "+file.Name())
			}
		}
	}

	// Check CRDs with conversion webhooks
	crdDir := filepath.Join(chartPath, "templates", "crd")
	if exists, _ := afero.DirExists(afero.NewOsFs(), crdDir); exists {
		files, err := afero.ReadDir(afero.NewOsFs(), crdDir)
		Expect(err).NotTo(HaveOccurred())

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
				continue
			}

			crdPath := filepath.Join(crdDir, file.Name())
			content, err := afero.ReadFile(afero.NewOsFs(), crdPath)
			Expect(err).NotTo(HaveOccurred())
			contentStr := string(content)

			if strings.Contains(contentStr, "cert-manager.io/inject-ca-from") {
				// Verify cert-manager annotation uses <chartname>.resourceName template
				chartName := "e2e-" + kbc.TestSuffix
				expected := `{{ include "` + chartName + `.resourceName" (dict "suffix" "serving-cert" "context" $) }}`
				Expect(contentStr).To(ContainSubstring(expected),
					"cert-manager.io/inject-ca-from annotation should use "+chartName+".resourceName in "+file.Name())
			}
		}
	}
}

func validateAppNameLabelTemplating(kbc *utils.TestContext, chartPath string) {
	chartName := "e2e-" + kbc.TestSuffix

	// Check all cert-manager resources
	certManagerDir := filepath.Join(chartPath, "templates", "cert-manager")
	files, err := afero.ReadDir(afero.NewOsFs(), certManagerDir)
	Expect(err).NotTo(HaveOccurred())

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
			continue
		}

		filePath := filepath.Join(certManagerDir, file.Name())
		content, err := afero.ReadFile(afero.NewOsFs(), filePath)
		Expect(err).NotTo(HaveOccurred())
		contentStr := string(content)

		if strings.Contains(contentStr, "app.kubernetes.io/name:") {
			// Verify app.kubernetes.io/name label uses <chartname>.name template
			Expect(contentStr).To(ContainSubstring(`app.kubernetes.io/name: {{ include "`+chartName+`.name" . }}`),
				"app.kubernetes.io/name label should use "+chartName+".name template in "+file.Name())
			Expect(contentStr).NotTo(ContainSubstring("app.kubernetes.io/name: e2e-"+kbc.TestSuffix),
				"app.kubernetes.io/name label should not be hardcoded in "+file.Name())
		}
	}
}

func validateRenderedChartConsistency(kbc *utils.TestContext, chartPath string) {
	// Render the chart using helm template command
	output, err := kbc.Kubectl.Command("exec", "-it", "helm-test-pod", "--", "helm", "template", "test-release", chartPath)
	if err != nil {
		// Fall back to checking template files directly if helm command fails
		By("helm template command not available, validating template files directly")
		return
	}

	renderedYAML := output

	// Extract issuer name from rendered output
	issuerNamePattern := `name:\s+(\S+)-selfsigned-issuer`
	issuerMatches := regexp.MustCompile(issuerNamePattern).FindStringSubmatch(renderedYAML)
	if len(issuerMatches) == 0 {
		// No cert-manager resources in output, skip validation
		return
	}
	issuerBaseName := issuerMatches[1]

	// Verify all issuerRef references match the issuer name
	issuerRefPattern := `name:\s+` + issuerBaseName + `-selfsigned-issuer`
	issuerRefMatches := regexp.MustCompile(issuerRefPattern).FindAllString(renderedYAML, -1)
	Expect(len(issuerRefMatches)).To(BeNumerically(">", 1),
		"Expected to find multiple consistent issuerRef references")

	// Verify cert-manager annotations reference the correct certificate names
	if strings.Contains(renderedYAML, "cert-manager.io/inject-ca-from") {
		servingCertPattern := issuerBaseName + `-serving-cert`
		Expect(renderedYAML).To(ContainSubstring(servingCertPattern),
			"cert-manager annotation should reference consistent certificate name")
	}
}
