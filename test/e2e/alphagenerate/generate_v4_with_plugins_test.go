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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
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
			dirRelPath := "../../../testdata/project-v4-with-plugins"
			absPath, err := filepath.Abs(dirRelPath)
			Expect(err).NotTo(HaveOccurred())
			kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
			kbc.Dir = absPath
			kbc.Domain = testProjectDomain
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())

			projectOutputDir = filepath.Join(kbc.Dir, outputDir)
			projectFilePath = filepath.Join(projectOutputDir, "PROJECT")
		})

		AfterEach(func() {
			By("destroying directory")
			kbc.Dir = filepath.Join(kbc.Dir, "output")
			kbc.Destroy()
		})

		It("should regenerate the project in project-v4-with-plugins directory with success", func() {
			regenerateProjectWith(kbc, projectOutputDir)
			By("checking that the project file was generated in the current directory")
			validateV4WithPluginsProjectFile(kbc, projectFilePath)
		})

	})
})

// Validate the PROJECT file for basic content and additional resources
func validateV4WithPluginsProjectFile(kbc *utils.TestContext, projectFile string) {
	projectConfig := getConfigFromProjectFile(projectFile)

	By("checking the layout in the PROJECT file")
	Expect(projectConfig.GetPluginChain()).To(ContainElement("go.kubebuilder.io/v4"))

	By("checking the domain in the PROJECT file")
	Expect(projectConfig.GetDomain()).To(Equal(kbc.Domain))

	By("checking the version in the PROJECT file")
	Expect(projectConfig.GetVersion().String()).To(Equal("3"))

	By("validating the Memcached API with controller true")
	memcachedGVK := resource.GVK{
		Group:   "example.com",
		Domain:  projectConfig.GetDomain(), // Adding the Domain field
		Version: "v1alpha1",
		Kind:    "Memcached",
	}
	Expect(projectConfig.HasResource(memcachedGVK)).To(BeTrue(), "Memcached API should be present in the PROJECT file")
	memcachedResource, err := projectConfig.GetResource(memcachedGVK)
	Expect(err).NotTo(HaveOccurred(), "Memcached resource should be retrievable")
	Expect(memcachedResource.Controller).To(BeTrue(), "Memcached API should have a controller")
	Expect(memcachedResource.API.Namespaced).To(BeTrue(), "Memcached API should be namespaced")
	Expect(memcachedResource.Path).To(Equal("sigs.k8s.io/kubebuilder/testdata/project-v4-with-plugins/api/v1alpha1"),
		"Memcached API should have the expected path")

	By("validating the Webhook for Memcached API")
	Expect(memcachedResource.Webhooks.Validation).To(BeTrue(), "Memcached API should have a defaulting webhook")
	Expect(memcachedResource.Webhooks.WebhookVersion).To(Equal("v1"), "Memcached API should have webhook version v1")

	By("validating the Busybox API with controller true")
	busyboxGVK := resource.GVK{
		Group:   "example.com",
		Domain:  projectConfig.GetDomain(), // Adding the Domain field
		Version: "v1alpha1",
		Kind:    "Busybox",
	}
	Expect(projectConfig.HasResource(busyboxGVK)).To(BeTrue(), "Busybox API should be present in the PROJECT file")
	busyboxResource, err := projectConfig.GetResource(busyboxGVK)
	Expect(err).NotTo(HaveOccurred(), "Busybox resource should be retrievable")
	Expect(busyboxResource.Controller).To(BeTrue(), "Busybox API should have a controller")
	Expect(busyboxResource.API.Namespaced).To(BeTrue(), "Busybox API should be namespaced")
	Expect(busyboxResource.Webhooks).To(BeNil(), "Busybox webhooks should be nil")
	Expect(busyboxResource.Path).To(Equal("sigs.k8s.io/kubebuilder/testdata/project-v4-with-plugins/api/v1alpha1"),
		"Busybox API should have the expected path")

	By("validating the Wordpress API with controller true")
	wordpressGVK := resource.GVK{
		Group:   "example.com",
		Domain:  projectConfig.GetDomain(), // Adding the Domain field
		Version: "v1",
		Kind:    "Wordpress",
	}
	Expect(projectConfig.HasResource(wordpressGVK)).To(BeTrue(), "Wordpress API should be present in the PROJECT file")
	wordpressResource, err := projectConfig.GetResource(wordpressGVK)
	Expect(err).NotTo(HaveOccurred(), "Wordpress resource should be retrievable")
	Expect(wordpressResource.Controller).To(BeTrue(), "Wordpress API should have a controller")
	Expect(wordpressResource.API.Namespaced).To(BeTrue(), "Wordpress API should be namespaced")
	Expect(wordpressResource.Path).To(Equal("sigs.k8s.io/kubebuilder/testdata/project-v4-with-plugins/api/v1"),
		"Wordpress API should have the expected path")

	By("validating the Webhook for Wordpress API")
	Expect(wordpressResource.Webhooks.Conversion).To(BeTrue(), "Wordpress API should have a defaulting webhook")
	Expect(wordpressResource.Webhooks.Spoke).To(Equal([]string{"v2"}), "Wordpress API should have a defaulting webhook")
	Expect(wordpressResource.Webhooks.WebhookVersion).To(Equal("v1"), "Wordpress API should have webhook version v1")

	By("validating the Wordpress API v2")
	wordpressv2GVK := resource.GVK{
		Group:   "example.com",
		Domain:  projectConfig.GetDomain(), // Adding the Domain field
		Version: "v2",
		Kind:    "Wordpress",
	}
	Expect(projectConfig.HasResource(wordpressv2GVK)).To(BeTrue(), "Wordpress API should be present in the PROJECT file")
	wordpressv2Resource, err := projectConfig.GetResource(wordpressv2GVK)
	Expect(err).NotTo(HaveOccurred(), "Wordpress resource should be retrievable")
	Expect(wordpressv2Resource.Controller).To(BeFalse(), "Wordpress API should not have a controller")
	Expect(wordpressv2Resource.API.Namespaced).To(BeTrue(), "Wordpress API should be namespaced")
	Expect(wordpressv2Resource.Webhooks).To(BeNil(), "Wordpress API should be namespaced")
	Expect(wordpressv2Resource.Path).To(Equal("sigs.k8s.io/kubebuilder/testdata/project-v4-with-plugins/api/v2"),
		"Wordpress API should have the expected path")

	By("decoding the DeployImage plugin configuration")
	var deployImageConfig v1alpha1.PluginConfig
	err = projectConfig.DecodePluginConfig("deploy-image.go.kubebuilder.io/v1-alpha", &deployImageConfig)
	Expect(err).NotTo(HaveOccurred(), "Failed to decode DeployImage plugin configuration")

	// Validate the resource configuration
	Expect(deployImageConfig.Resources).To(HaveLen(2), "Expected at least one resource for the DeployImage plugin")

	memResource := deployImageConfig.Resources[0]
	Expect(memResource.Group).To(Equal("example.com"), "Expected group to be 'example.com'")
	Expect(memResource.Version).To(Equal("v1alpha1"), "Expected version to be v1alpha1")
	Expect(memResource.Kind).To(Equal("Memcached"), "Expected kind to be 'Memcached'")

	options := memResource.Options
	Expect(options.Image).To(Equal("memcached:1.6.26-alpine3.19"), "Expected image to match")
	Expect(options.ContainerCommand).To(Equal("memcached,--memory-limit=64,-o,modern,-v"),
		"Expected container command to match")
	Expect(options.ContainerPort).To(Equal("11211"), "Expected container port to match")
	Expect(options.RunAsUser).To(Equal("1001"), "Expected runAsUser to match")

	busyBoxResource := deployImageConfig.Resources[1]
	Expect(busyBoxResource.Group).To(Equal("example.com"), "Expected group to be 'example.com'")
	Expect(busyBoxResource.Kind).To(Equal("Busybox"), "Expected kind to be 'Memcached'")
	Expect(busyBoxResource.Version).To(Equal("v1alpha1"), "Expected version to be 'v1alpha1'")

	options = busyBoxResource.Options
	Expect(options.Image).To(Equal("busybox:1.36.1"), "Expected image to match")

	By("decoding the grafana plugin configuration")
	var grafanaConfig v1alpha1.PluginConfig
	err = projectConfig.DecodePluginConfig("grafana.kubebuilder.io/v1-alpha", &grafanaConfig)
	Expect(err).NotTo(HaveOccurred(), "Failed to decode DeployImage plugin configuration")

	// Validate the resource configuration
	Expect(grafanaConfig.Resources).To(BeEmpty(), "Expected zero resource for the Grafana plugin")

	By("decoding the helm plugin configuration")
	var helmConfig v1alpha1.PluginConfig
	err = projectConfig.DecodePluginConfig("grafana.kubebuilder.io/v1-alpha", &helmConfig)
	Expect(err).NotTo(HaveOccurred(), "Failed to decode Helm plugin configuration")

	// Validate the resource configuration
	Expect(helmConfig.Resources).To(BeEmpty(), "Expected zero resource for the Helm plugin")

}
