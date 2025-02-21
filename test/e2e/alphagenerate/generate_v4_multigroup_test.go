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

const testProjectDomain = "testproject.org"

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
			dirRelPath := "../../../testdata/project-v4-multigroup"
			absPath, err := filepath.Abs(dirRelPath)
			Expect(err).NotTo(HaveOccurred())
			kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())

			kbc.Dir = absPath
			kbc.Domain = testProjectDomain
			projectOutputDir = filepath.Join(kbc.Dir, outputDir)
			projectFilePath = filepath.Join(projectOutputDir, "PROJECT")
		})

		AfterEach(func() {
			By("destroying directory")
			kbc.Dir = filepath.Join(kbc.Dir, "output")
			kbc.Destroy()
		})

		It("should regenerate the project in project-v4-multigroup directory with success", func() {
			regenerateProjectWith(kbc, projectOutputDir)
			By("checking that the project file was generated in the current directory")
			validateV4MultigroupProjectFile(kbc, projectFilePath)
		})

	})
})

// Validate the PROJECT file for basic content and additional resources
func validateV4MultigroupProjectFile(kbc *utils.TestContext, projectFile string) {
	projectConfig := getConfigFromProjectFile(projectFile)

	By("checking the layout in the PROJECT file")
	Expect(projectConfig.GetPluginChain()).To(ContainElement("go.kubebuilder.io/v4"))

	By("checking the domain in the PROJECT file")
	Expect(projectConfig.GetDomain()).To(Equal(kbc.Domain))

	By("checking the version in the PROJECT file")
	Expect(projectConfig.GetVersion().String()).To(Equal("3"))

	By("checking the multigroup flag in the PROJECT file")
	Expect(projectConfig.IsMultiGroup()).To(BeTrue())
	By("validating the Captain API with controller and namespaced true")
	captainGVK := resource.GVK{
		Group:   "crew",
		Domain:  projectConfig.GetDomain(), // Adding the Domain field
		Version: "v1",
		Kind:    "Captain",
	}
	Expect(projectConfig.HasResource(captainGVK)).To(BeTrue(), "Captain API should be present in the PROJECT file")
	captainResource, err := projectConfig.GetResource(captainGVK)
	Expect(err).NotTo(HaveOccurred(), "Captain resource should be retrievable")
	Expect(captainResource.Controller).To(BeTrue(), "Captain API should have a controller")
	Expect(captainResource.API.Namespaced).To(BeTrue(), "Captain API should be namespaced")
	Expect(captainResource.Path).To(Equal("sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/crew/v1"),
		"Captain API should have the expected path")

	By("validating the Webhook for Captain API")
	Expect(captainResource.Webhooks.Defaulting).To(BeTrue(), "Captain API should have a defaulting webhook")
	Expect(captainResource.Webhooks.Validation).To(BeTrue(), "Captain API should have a validation webhook")
	Expect(captainResource.Webhooks.WebhookVersion).To(Equal("v1"), "Captain API should have webhook version v1")

	By("validating the Frigate API with controller and namespaced true")
	frigateGVK := resource.GVK{
		Group:   "ship",
		Domain:  projectConfig.GetDomain(), // Adding the Domain field
		Version: "v1beta1",
		Kind:    "Frigate",
	}
	Expect(projectConfig.HasResource(captainGVK)).To(BeTrue(), "Frigate API should be present in the PROJECT file")
	frigateResource, err := projectConfig.GetResource(frigateGVK)
	Expect(err).NotTo(HaveOccurred(), "Frigate resource should be retrievable")
	Expect(frigateResource.Controller).To(BeTrue(), "Frigate API should have a controller")
	Expect(frigateResource.API.Namespaced).To(BeTrue(), "Frigate API should be namespaced")
	Expect(frigateResource.Webhooks).To(BeNil(), "Frigate API should not have webhooks")
	Expect(frigateResource.Path).To(Equal("sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/ship/v1beta1"),
		"Frigate API should have the expected path")

	By("validating the Destroyer API with controller true")
	destroyerGVK := resource.GVK{
		Group:   "ship",
		Domain:  projectConfig.GetDomain(), // Adding the Domain field
		Version: "v1",
		Kind:    "Destroyer",
	}
	Expect(projectConfig.HasResource(captainGVK)).To(BeTrue(), "Destroyer API should be present in the PROJECT file")
	destroyerResource, err := projectConfig.GetResource(destroyerGVK)
	Expect(err).NotTo(HaveOccurred(), "Destroyer resource should be retrievable")
	Expect(destroyerResource.Controller).To(BeTrue(), "Destroyer API should have a controller")
	Expect(destroyerResource.API.Namespaced).To(BeFalse(), "Destroyer API should be namespaced")
	Expect(destroyerResource.Path).To(Equal("sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/ship/v1"),
		"Destroyer API should have the expected path")

	By("validating the Webhook for Destroyer API")
	Expect(destroyerResource.Webhooks.Defaulting).To(BeTrue(), "Destroyer API should have a defaulting webhook")
	Expect(destroyerResource.Webhooks.WebhookVersion).To(Equal("v1"), "Destroyer API should have webhook version v1")

	By("validating the Cruiser API with controller true")
	cruiserGVK := resource.GVK{
		Group:   "ship",
		Domain:  projectConfig.GetDomain(), // Adding the Domain field
		Version: "v2alpha1",
		Kind:    "Cruiser",
	}
	Expect(projectConfig.HasResource(captainGVK)).To(BeTrue(), "Cruiser API should be present in the PROJECT file")
	cruiserResource, err := projectConfig.GetResource(cruiserGVK)
	Expect(err).NotTo(HaveOccurred(), "Cruiser resource should be retrievable")
	Expect(cruiserResource.Controller).To(BeTrue(), "Cruiser API should have a controller")
	Expect(cruiserResource.API.Namespaced).To(BeFalse(), "Cruiser API should be namespaced")
	Expect(cruiserResource.Path).To(Equal("sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/ship/v2alpha1"),
		"Cruiser API should have the expected path")

	By("validating the Webhook for Destroyer API")
	Expect(cruiserResource.Webhooks.Validation).To(BeTrue(), "Cruiser API should have a defaulting webhook")
	Expect(cruiserResource.Webhooks.WebhookVersion).To(Equal("v1"), "Cruiser API should have webhook version v1")

	By("validating the Kraken API with controller true")
	krakenGVK := resource.GVK{
		Group:   "sea-creatures",
		Domain:  projectConfig.GetDomain(), // Adding the Domain field
		Version: "v1beta1",
		Kind:    "Kraken",
	}
	Expect(projectConfig.HasResource(krakenGVK)).To(BeTrue(), "Kraken API should be present in the PROJECT file")
	krakenResource, err := projectConfig.GetResource(krakenGVK)
	Expect(err).NotTo(HaveOccurred(), "Kraken resource should be retrievable")
	Expect(krakenResource.Controller).To(BeTrue(), "Kraken API should have a controller")
	Expect(krakenResource.API.Namespaced).To(BeTrue(), "Kraken API should be namespaced")
	Expect(krakenResource.Webhooks).To(BeNil(), "Kraken API should be namespaced")
	Expect(krakenResource.Path).To(Equal(
		"sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/sea-creatures/v1beta1"),
		"Kraken API should have the expected path")

	By("validating the Leviathan API with controller true")
	leviathanGVK := resource.GVK{
		Group:   "sea-creatures",
		Domain:  projectConfig.GetDomain(), // Adding the Domain field
		Version: "v1beta2",
		Kind:    "Leviathan",
	}
	Expect(projectConfig.HasResource(leviathanGVK)).To(BeTrue(), "Leviathan API should be present in the PROJECT file")
	leviathanResource, err := projectConfig.GetResource(leviathanGVK)
	Expect(err).NotTo(HaveOccurred(), "Leviathan resource should be retrievable")
	Expect(leviathanResource.Controller).To(BeTrue(), "Leviathan API should have a controller")
	Expect(leviathanResource.API.Namespaced).To(BeTrue(), "Leviathan API should be namespaced")
	Expect(leviathanResource.Webhooks).To(BeNil(), "Leviathan API should be namespaced")
	Expect(leviathanResource.Path).To(Equal(
		"sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/sea-creatures/v1beta2"),
		"Leviathan API should have the expected path")

	By("validating the HealthCheckPolicy API with controller true")
	healthCheckPolicyGVK := resource.GVK{
		Group:   "foo.policy",
		Domain:  projectConfig.GetDomain(), // Adding the Domain field
		Version: "v1",
		Kind:    "HealthCheckPolicy",
	}
	Expect(projectConfig.HasResource(healthCheckPolicyGVK)).To(BeTrue(),
		"HealthCheckPolicy API should be present in the PROJECT file")
	healthCheckPolicyResource, err := projectConfig.GetResource(healthCheckPolicyGVK)
	Expect(err).NotTo(HaveOccurred(), "HealthCheckPolicy resource should be retrievable")
	Expect(healthCheckPolicyResource.Controller).To(BeTrue(), "HealthCheckPolicy API should have a controller")
	Expect(healthCheckPolicyResource.API.Namespaced).To(BeTrue(), "HealthCheckPolicy API should be namespaced")
	Expect(healthCheckPolicyResource.Webhooks).To(BeNil(), "HealthCheckPolicy API should be namespaced")
	Expect(healthCheckPolicyResource.Path).To(Equal(
		"sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/foo.policy/v1"),
		"HealthCheckPolicy API should have the expected path")

	By("validating the Core API with kind Deployment")
	deploymentGVK := resource.GVK{
		Group:   "apps",
		Domain:  "",
		Version: "v1",
		Kind:    "Deployment",
	}

	Expect(projectConfig.HasResource(deploymentGVK)).To(BeTrue(),
		"Deployment Resource should be present in the PROJECT file")
	deploymentResource, err := projectConfig.GetResource(deploymentGVK)
	Expect(err).NotTo(HaveOccurred(), "Deployment API should be retrievable")
	Expect(deploymentResource.Controller).To(BeTrue(), "Deployment API should have a controller")
	Expect(deploymentResource.API).To(BeNil(), "Deployment API should not have API scaffold")
	Expect(deploymentResource.Path).To(Equal("k8s.io/api/apps/v1"),
		"Deployment API should have expected path")

	By("validating the Webhook for Deployment API")
	Expect(deploymentResource.Webhooks.Defaulting).To(BeTrue(), "Deployment API should have a defaulting webhook")
	Expect(deploymentResource.Webhooks.Validation).To(BeTrue(), "Deployment API should have a defaulting webhook")
	Expect(deploymentResource.Webhooks.WebhookVersion).To(Equal("v1"), "Deployment API should have webhook version v1")

	By("validating the foo API with kind Bar")
	barGVK := resource.GVK{
		Group:   "foo",
		Domain:  projectConfig.GetDomain(),
		Version: "v1",
		Kind:    "Bar",
	}

	Expect(projectConfig.HasResource(barGVK)).To(BeTrue(),
		"Bar Resource should be present in the PROJECT file")
	barResource, err := projectConfig.GetResource(barGVK)
	Expect(err).NotTo(HaveOccurred(), "Bar API should be retrievable")
	Expect(barResource.Controller).To(BeTrue(), "Bar API should have a controller")
	Expect(barResource.API.Namespaced).To(BeTrue(), "Bar API should not have API scaffold")
	Expect(barResource.Path).To(Equal("sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/foo/v1"),
		"Bar API should have expected path")

	By("validating the fiz API with kind Bar")
	fizBarGVK := resource.GVK{
		Group:   "fiz",
		Domain:  projectConfig.GetDomain(),
		Version: "v1",
		Kind:    "Bar",
	}

	Expect(projectConfig.HasResource(fizBarGVK)).To(BeTrue(),
		"Bar Resource should be present in the PROJECT file")
	fizBarResource, err := projectConfig.GetResource(fizBarGVK)
	Expect(err).NotTo(HaveOccurred(), "Bar API should be retrievable")
	Expect(fizBarResource.Controller).To(BeTrue(), "Bar API should have a controller")
	Expect(barResource.API.Namespaced).To(BeTrue(), "Bar API should not have API scaffold")
	Expect(fizBarResource.Path).To(Equal("sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/fiz/v1"),
		"Bar API should have expected path")

	By("validating the External API with kind Certificate from certManager")
	certmanagerGVK := resource.GVK{
		Group:   "cert-manager",
		Domain:  "io",
		Version: "v1",
		Kind:    "Certificate",
	}

	Expect(projectConfig.HasResource(certmanagerGVK)).To(BeTrue(),
		"Certificate Resource should be present in the PROJECT file")
	certmanagerResource, err := projectConfig.GetResource(certmanagerGVK)
	Expect(err).NotTo(HaveOccurred(), "Certificate resource should be retrievable")
	Expect(certmanagerResource.Controller).To(BeTrue(), "Certificate API should have a controller")
	Expect(certmanagerResource.API).To(BeNil(), "Certificate API should not have API scaffold")
	Expect(certmanagerResource.Webhooks).To(BeNil(), "Certificate API should not have webhooks")
	Expect(certmanagerResource.Path).To(Equal("github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"),
		"Certificate API should have expected path")

	By("validating the External API with kind Issuer from certManager")
	issuerGVK := resource.GVK{
		Group:   "cert-manager",
		Domain:  "io",
		Version: "v1",
		Kind:    "Issuer",
	}

	Expect(projectConfig.HasResource(issuerGVK)).To(BeTrue(),
		"Issuer Resource should be present in the PROJECT file")
	issuerResource, err := projectConfig.GetResource(issuerGVK)
	Expect(err).NotTo(HaveOccurred(), "Issuer resource should be retrievable")
	Expect(issuerResource.Controller).To(BeFalse(), "Issuer API should not have a controller")
	Expect(issuerResource.API).To(BeNil(), "Issuer API should not have API scaffold")
	Expect(issuerResource.Path).To(Equal("github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"),
		"Issuer API should have expected path")

	By("validating the Webhook for Issuer API")
	Expect(issuerResource.Webhooks.Defaulting).To(BeTrue(), "Issuer API should have a defaulting webhook")
	Expect(issuerResource.Webhooks.WebhookVersion).To(Equal("v1"), "Issuer API should have webhook version v1")

	By("validating the Core API with kind Pod")
	podGVK := resource.GVK{
		Group:   "core",
		Domain:  "",
		Version: "v1",
		Kind:    "Pod",
	}

	Expect(projectConfig.HasResource(podGVK)).To(BeTrue(),
		"Pod Resource should be present in the PROJECT file")
	podResource, err := projectConfig.GetResource(podGVK)
	Expect(err).NotTo(HaveOccurred(), "Pod API should be retrievable")
	Expect(podResource.Controller).To(BeFalse(), "Pod API should have a controller")
	Expect(podResource.API).To(BeNil(), "Pod API should not have API scaffold")
	Expect(podResource.Path).To(Equal("k8s.io/api/core/v1"),
		"Pod API should have expected path")

	By("validating the Webhook for Pod API")
	Expect(podResource.Webhooks.Validation).To(BeTrue(), "Pod API should have a validation webhook")
	Expect(podResource.Webhooks.WebhookVersion).To(Equal("v1"), "Pod API should have webhook version v1")

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
	Expect(memcachedResource.Path).To(Equal(
		"sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/example.com/v1alpha1"),
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
	Expect(busyboxResource.Path).To(Equal(
		"sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/example.com/v1alpha1"),
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
	Expect(wordpressResource.Path).To(Equal("sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/example.com/v1"),
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
	Expect(wordpressv2Resource.Path).To(Equal("sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/example.com/v2"),
		"Wordpress API should have the expected path")

	By("decoding the DeployImage plugin configuration")
	var deployImageConfig v1alpha1.PluginConfig
	err = projectConfig.DecodePluginConfig("deploy-image.go.kubebuilder.io/v1-alpha", &deployImageConfig)
	Expect(err).NotTo(HaveOccurred(), "Failed to decode DeployImage plugin configuration")

	// Validate the resource configuration
	Expect(deployImageConfig.Resources).To(HaveLen(2), "Expected two resources for the DeployImage plugin")

	memResource := deployImageConfig.Resources[0]
	Expect(memResource.Group).To(Equal("example.com"), "Expected group to be 'crew'")
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
	Expect(busyBoxResource.Kind).To(Equal("Busybox"), "Expected kind to be 'Busybox'")
	Expect(busyBoxResource.Version).To(Equal("v1alpha1"), "Expected kind to be 'v1alpha1'")

	options = busyBoxResource.Options
	Expect(options.Image).To(Equal("busybox:1.36.1"), "Expected image to match")

	By("decoding the grafana plugin configuration")
	var grafanaConfig v1alpha1.PluginConfig
	err = projectConfig.DecodePluginConfig("grafana.kubebuilder.io/v1-alpha", &grafanaConfig)
	Expect(err).NotTo(HaveOccurred(), "Failed to decode DeployImage plugin configuration")

	// Validate the resource configuration
	Expect(grafanaConfig.Resources).To(BeEmpty(), "Expected zero resource for the Grafana plugin")

}
