/*
Copyright 2023 The Kubernetes Authors.

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

package alphagenerate

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/deploy-image/v1alpha1"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("kubebuilder", func() {
	Context("alpha generate", func() {

		var (
			kbc              *utils.TestContext
			projectOutputDir string
			projectFilePath  string
		)

		const outputDir = "output"

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())

			projectOutputDir = filepath.Join(kbc.Dir, outputDir)
			projectFilePath = filepath.Join(projectOutputDir, "PROJECT")

			By("initializing a project")
			err = kbc.Init(
				"--plugins", "go/v4",
				"--project-version", "3",
				"--domain", kbc.Domain,
			)
			Expect(err).NotTo(HaveOccurred(), "Failed to create project")
		})

		AfterEach(func() {
			By("destroying directory")
			kbc.Destroy()
		})

		It("should regenerate the project with success", func() {
			generateProject(kbc)
			regenerateProject(kbc, projectOutputDir)
			validateProjectFile(kbc, projectFilePath)
		})

		It("should regenerate project with grafana plugin with success", func() {
			generateProjectWithGrafanaPlugin(kbc)
			regenerateProject(kbc, projectOutputDir)
			validateGrafanaPlugin(projectFilePath)
		})

		It("should regenerate project with DeployImage plugin with success", func() {
			generateProjectWithDeployImagePlugin(kbc)
			regenerateProject(kbc, projectOutputDir)
			validateDeployImagePlugin(projectFilePath)
		})

		It("should regenerate project with helm plugin with success", func() {
			generateProjectWithHelmPlugin(kbc)
			regenerateProject(kbc, projectOutputDir)
			validateHelmPlugin(projectFilePath)
		})
	})
})

func generateProject(kbc *utils.TestContext) {
	By("editing project to enable multigroup layout")
	err := kbc.Edit("--multigroup", "true")
	Expect(err).NotTo(HaveOccurred(), "Failed to edit project for multigroup layout")

	By("creating API definition")
	err = kbc.CreateAPI(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--namespaced",
		"--resource",
		"--controller",
		"--make=false",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to scaffold api with resource and controller")

	By("creating API definition with controller and resource")
	err = kbc.CreateAPI(
		"--group", "crew",
		"--version", "v1",
		"--kind", "Memcached",
		"--namespaced",
		"--resource",
		"--controller",
		"--make=false",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to scaffold API with resource and controller")

	By("creating API definition with controller and resource")
	err = kbc.CreateAPI(
		"--group", "crew",
		"--version", "v2",
		"--kind", "Memcached",
		"--namespaced",
		"--resource=true",
		"--controller=false",
		"--make=false",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to scaffold API with resource and controller")

	By("creating Webhook for Memcached API")
	err = kbc.CreateWebhook(
		"--group", "crew",
		"--version", "v1",
		"--kind", "Memcached",
		"--defaulting",
		"--programmatic-validation",
		"--conversion",
		"--spoke", "v2",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to scaffold webhook for Memcached API")

	By("creating API without controller (Admiral)")
	err = kbc.CreateAPI(
		"--group", "crew",
		"--version", "v1",
		"--kind", "Admiral",
		"--controller=false",
		"--resource=true",
		"--namespaced=false",
		"--make=false",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to scaffold API without controller")

	By("creating API with controller and resource (Captain)")
	err = kbc.CreateAPI(
		"--group", "crew",
		"--version", "v1",
		"--kind", "Captain",
		"--controller=true",
		"--resource=true",
		"--namespaced=true",
		"--make=false",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to scaffold API with namespaced true")

	By("creating an external API with cert-manager")
	err = kbc.CreateAPI(
		"--group", "certmanager",
		"--version", "v1",
		"--kind", "Certificate",
		"--controller=true",
		"--resource=false",
		"--make=false",
		"--external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1",
		"--external-api-domain=cert-manager.io",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to scaffold API with external API")
}

func regenerateProject(kbc *utils.TestContext, projectOutputDir string) {
	By("regenerating the project")
	err := kbc.Regenerate(
		fmt.Sprintf("--input-dir=%s", kbc.Dir),
		fmt.Sprintf("--output-dir=%s", projectOutputDir),
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to regenerate project")
}

func generateProjectWithGrafanaPlugin(kbc *utils.TestContext) {
	By("editing project to enable Grafana plugin")
	err := kbc.Edit("--plugins", "grafana.kubebuilder.io/v1-alpha")
	Expect(err).NotTo(HaveOccurred(), "Failed to edit project to enable Grafana Plugin")
}

func generateProjectWithHelmPlugin(kbc *utils.TestContext) {
	By("editing project to enable Helm plugin")
	err := kbc.Edit("--plugins", "helm.kubebuilder.io/v1-alpha")
	Expect(err).NotTo(HaveOccurred(), "Failed to edit project to enable Helm Plugin")
}

func generateProjectWithDeployImagePlugin(kbc *utils.TestContext) {
	By("creating an API with DeployImage plugin")
	err := kbc.CreateAPI(
		"--group", "crew",
		"--version", "v1",
		"--kind", "Memcached",
		"--image=memcached:1.6.15-alpine",
		"--image-container-command=memcached,--memory-limit=64,modern,-v",
		"--image-container-port=11211",
		"--run-as-user=1001",
		"--plugins=deploy-image/v1-alpha",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to create API with Deploy Image Plugin")
}

// Validate the PROJECT file for basic content and additional resources
func validateProjectFile(kbc *utils.TestContext, projectFile string) {
	projectConfig := getConfigFromProjectFile(projectFile)

	By("checking the layout in the PROJECT file")
	Expect(projectConfig.GetPluginChain()).To(ContainElement("go.kubebuilder.io/v4"))

	By("checking the multigroup flag in the PROJECT file")
	Expect(projectConfig.IsMultiGroup()).To(BeTrue())

	By("checking the domain in the PROJECT file")
	Expect(projectConfig.GetDomain()).To(Equal(kbc.Domain))

	By("checking the version in the PROJECT file")
	Expect(projectConfig.GetVersion().String()).To(Equal("3"))

	By("validating the Memcached API with controller and resource")
	memcachedGVK := resource.GVK{
		Group:   "crew",
		Domain:  projectConfig.GetDomain(), // Adding the Domain field
		Version: "v1",
		Kind:    "Memcached",
	}
	Expect(projectConfig.HasResource(memcachedGVK)).To(BeTrue(), "Memcached API should be present in the PROJECT file")
	memcachedResource, err := projectConfig.GetResource(memcachedGVK)
	Expect(err).NotTo(HaveOccurred(), "Memcached API should be retrievable")
	Expect(memcachedResource.Controller).To(BeTrue(), "Memcached API should have a controller")
	Expect(memcachedResource.API.Namespaced).To(BeTrue(), "Memcached API should be namespaced")

	By("validating the Webhook for Memcached API")
	Expect(memcachedResource.Webhooks.Defaulting).To(BeTrue(), "Memcached API should have defaulting webhook")
	Expect(memcachedResource.Webhooks.Validation).To(BeTrue(), "Memcached API should have validation webhook")
	Expect(memcachedResource.Webhooks.Conversion).To(BeTrue(), "Memcached API should have a conversion webhook")
	Expect(memcachedResource.Webhooks.WebhookVersion).To(Equal("v1"), "Memcached API should have webhook version v1")

	// Validate the presence of Admiral API without controller
	By("validating the Admiral API without a controller")
	admiralGVK := resource.GVK{
		Group:   "crew",
		Domain:  projectConfig.GetDomain(), // Adding the Domain field
		Version: "v1",
		Kind:    "Admiral",
	}
	Expect(projectConfig.HasResource(admiralGVK)).To(BeTrue(), "Admiral API should be present in the PROJECT file")
	admiralResource, err := projectConfig.GetResource(admiralGVK)
	Expect(err).NotTo(HaveOccurred(), "Admiral API should be retrievable")
	Expect(admiralResource.Controller).To(BeFalse(), "Admiral API should not have a controller")
	Expect(admiralResource.API.Namespaced).To(BeFalse(), "Admiral API should be cluster-scoped (not namespaced)")
	Expect(admiralResource.Webhooks).To(BeNil(), "Admiral API should not have webhooks")

	By("validating the Captain API with controller and namespaced true")
	captainGVK := resource.GVK{
		Group:   "crew",
		Domain:  projectConfig.GetDomain(), // Adding the Domain field
		Version: "v1",
		Kind:    "Captain",
	}
	Expect(projectConfig.HasResource(captainGVK)).To(BeTrue(), "Captain API should be present in the PROJECT file")
	captainResource, err := projectConfig.GetResource(captainGVK)
	Expect(err).NotTo(HaveOccurred(), "Captain API should be retrievable")
	Expect(captainResource.Controller).To(BeTrue(), "Captain API should have a controller")
	Expect(captainResource.API.Namespaced).To(BeTrue(), "Captain API should be namespaced")
	Expect(captainResource.Webhooks).To(BeNil(), "Capitan API should not have webhooks")

	By("validating the External API with kind Certificate from certManager")
	certmanagerGVK := resource.GVK{
		Group:   "certmanager",
		Domain:  "cert-manager.io",
		Version: "v1",
		Kind:    "Certificate",
	}
	Expect(projectConfig.HasResource(certmanagerGVK)).To(BeTrue(),
		"Certificate Resource should be present in the PROJECT file")
	certmanagerResource, err := projectConfig.GetResource(certmanagerGVK)
	Expect(err).NotTo(HaveOccurred(), "Captain API should be retrievable")
	Expect(certmanagerResource.Controller).To(BeTrue(), "Certificate API should have a controller")
	Expect(certmanagerResource.API).To(BeNil(), "Certificate API should not have API scaffold")
	Expect(certmanagerResource.Webhooks).To(BeNil(), "Certificate API should not have webhooks")
	Expect(certmanagerResource.Path).To(Equal("github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"),
		"Certificate API should have expected path")
}

func getConfigFromProjectFile(projectFilePath string) config.Config {
	By("loading the PROJECT configuration")
	fs := afero.NewOsFs()
	store := yaml.New(machinery.Filesystem{FS: fs})
	err := store.LoadFrom(projectFilePath)
	Expect(err).NotTo(HaveOccurred(), "Failed to load PROJECT configuration")

	cfg := store.Config()
	return cfg
}

// Validate the PROJECT file for the Grafana plugin
func validateGrafanaPlugin(projectFile string) {
	projectConfig := getConfigFromProjectFile(projectFile)

	By("checking the Grafana plugin in the PROJECT file")
	var grafanaPluginConfig map[string]interface{}
	err := projectConfig.DecodePluginConfig("grafana.kubebuilder.io/v1-alpha", &grafanaPluginConfig)
	Expect(err).NotTo(HaveOccurred())
	Expect(grafanaPluginConfig).NotTo(BeNil())
}

// Validate the PROJECT file for the DeployImage plugin
func validateDeployImagePlugin(projectFile string) {
	projectConfig := getConfigFromProjectFile(projectFile)

	By("decoding the DeployImage plugin configuration")
	var deployImageConfig v1alpha1.PluginConfig
	err := projectConfig.DecodePluginConfig("deploy-image.go.kubebuilder.io/v1-alpha", &deployImageConfig)
	Expect(err).NotTo(HaveOccurred(), "Failed to decode DeployImage plugin configuration")

	// Validate the resource configuration
	Expect(deployImageConfig.Resources).ToNot(BeEmpty(), "Expected at least one resource for the DeployImage plugin")

	resource := deployImageConfig.Resources[0]
	Expect(resource.Group).To(Equal("crew"), "Expected group to be 'crew'")
	Expect(resource.Kind).To(Equal("Memcached"), "Expected kind to be 'Memcached'")

	options := resource.Options
	Expect(options.Image).To(Equal("memcached:1.6.15-alpine"), "Expected image to match")
	Expect(options.ContainerCommand).To(Equal("memcached,--memory-limit=64,modern,-v"),
		"Expected container command to match")
	Expect(options.ContainerPort).To(Equal("11211"), "Expected container port to match")
	Expect(options.RunAsUser).To(Equal("1001"), "Expected runAsUser to match")
}

func validateHelmPlugin(projectFile string) {
	projectConfig := getConfigFromProjectFile(projectFile)

	By("checking the Helm plugin in the PROJECT file")
	var helmPluginConfig map[string]interface{}
	err := projectConfig.DecodePluginConfig("helm.kubebuilder.io/v1-alpha", &helmPluginConfig)
	Expect(err).NotTo(HaveOccurred(), "Failed to decode Helm plugin configuration")
	Expect(helmPluginConfig).NotTo(BeNil(), "Expected Helm plugin configuration to be present in the PROJECT file")
}
