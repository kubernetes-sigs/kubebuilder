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
