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
			dirRelPath := "../../../testdata/project-v4"
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

		It("should regenerate the project in project-v4 directory with success", func() {
			regenerateAndValidate(kbc, projectOutputDir, projectFilePath)
		})

	})
})

func regenerateAndValidate(kbc *utils.TestContext, projectOutputDir, projectFilePath string) {
	regenerateProjectWith(kbc, projectOutputDir)
	By("checking that the project file was generated in the current directory")
	validateV4ProjectFile(kbc, projectFilePath)
}

// Validate the PROJECT file for basic content and additional resources
func validateV4ProjectFile(kbc *utils.TestContext, projectFile string) {
	projectConfig := getConfigFromProjectFile(projectFile)

	By("checking the layout in the PROJECT file")
	Expect(projectConfig.GetPluginChain()).To(ContainElement("go.kubebuilder.io/v4"))

	By("checking the domain in the PROJECT file")
	Expect(projectConfig.GetDomain()).To(Equal(kbc.Domain))

	By("checking the version in the PROJECT file")
	Expect(projectConfig.GetVersion().String()).To(Equal("3"))

	By("validating the FirstMate API with controller and resource and with version v1")
	firstMateGVK := resource.GVK{
		Group:   "crew",
		Domain:  projectConfig.GetDomain(), // Adding the Domain field
		Version: "v1",
		Kind:    "FirstMate",
	}
	Expect(projectConfig.HasResource(firstMateGVK)).To(BeTrue(), "FirstMate API should be present in the PROJECT file")
	firstMateResource, err := projectConfig.GetResource(firstMateGVK)
	Expect(err).NotTo(HaveOccurred(), "FirstMate resource should be retrievable")
	Expect(firstMateResource.Controller).To(BeTrue(), "FirstMate API should have a controller")
	Expect(firstMateResource.API.Namespaced).To(BeTrue(), "FirstMate API should be namespaced")

	By("validating the Webhook for FirstMate API")
	Expect(firstMateResource.Webhooks.Conversion).To(BeTrue(), "FirstMate API should have a conversion webhook")
	Expect(firstMateResource.Webhooks.WebhookVersion).To(Equal("v1"), "FirstMate API should have webhook version v1")

	By("validating the FirstMate API with controller and resource and with version v2")
	firstMatev2GVK := resource.GVK{
		Group:   "crew",
		Domain:  projectConfig.GetDomain(), // Adding the Domain field
		Version: "v2",
		Kind:    "FirstMate",
	}
	Expect(projectConfig.HasResource(firstMatev2GVK)).To(BeTrue(),
		"FirstMate v2 API should be present in the PROJECT file")
	firstMatev2Resource, err := projectConfig.GetResource(firstMatev2GVK)
	Expect(err).NotTo(HaveOccurred(), "FirstMate v2 resource should be retrievable")
	Expect(firstMatev2Resource.Controller).To(BeFalse(), "FirstMate v2 API should not have a controller")
	Expect(firstMatev2Resource.API.Namespaced).To(BeTrue(), "FirstMate v2 API should be namespaced")
	Expect(firstMatev2Resource.Webhooks).To(BeNil(), "FirstMate v2 API should not have webhooks")
	Expect(firstMatev2Resource.Path).To(Equal("sigs.k8s.io/kubebuilder/testdata/project-v4/api/v2"),
		"FirstMate API v2 should have expected path")

	// Validate the presence of Admiral API without controller
	By("validating the Admiral API with a controller")
	admiralGVK := resource.GVK{
		Group:   "crew",
		Domain:  projectConfig.GetDomain(), // Adding the Domain field
		Version: "v1",
		Kind:    "Admiral",
	}
	Expect(projectConfig.HasResource(admiralGVK)).To(BeTrue(), "Admiral API should be present in the PROJECT file")
	admiralResource, err := projectConfig.GetResource(admiralGVK)
	Expect(err).NotTo(HaveOccurred(), "Admiral resource should be retrievable")
	Expect(admiralResource.Controller).To(BeTrue(), "Admiral API should have a controller")
	Expect(admiralResource.API.Namespaced).To(BeFalse(), "Admiral API should be cluster-scoped (not namespaced)")

	By("validating the Webhook for Admiral API")
	Expect(admiralResource.Webhooks.Defaulting).To(BeTrue(), "Admiral API should have a defaulting webhook")
	Expect(admiralResource.Webhooks.WebhookVersion).To(Equal("v1"), "Admiral API should have webhook version v1")

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

	By("validating the Webhook for Captain API")
	Expect(captainResource.Webhooks.Defaulting).To(BeTrue(), "Captain API should have a conversion webhook")
	Expect(captainResource.Webhooks.Validation).To(BeTrue(), "Captain API should have a validation webhook")
	Expect(captainResource.Webhooks.WebhookVersion).To(Equal("v1"), "Captain API should have webhook version v1")

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
	Expect(admiralResource.Webhooks.Defaulting).To(BeTrue(), "Issuer API should have a defaulting webhook")
	Expect(admiralResource.Webhooks.WebhookVersion).To(Equal("v1"), "Issuer API should have webhook version v1")

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
	Expect(podResource.Webhooks.Defaulting).To(BeTrue(), "Pod API should have a defaulting webhook")
	Expect(podResource.Webhooks.WebhookVersion).To(Equal("v1"), "Pod API should have webhook version v1")

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
	Expect(deploymentResource.Controller).To(BeFalse(), "Deployment API should have a controller")
	Expect(deploymentResource.API).To(BeNil(), "Deployment API should not have API scaffold")
	Expect(deploymentResource.Path).To(Equal("k8s.io/api/apps/v1"),
		"Deployment API should have expected path")

	By("validating the Webhook for Deployment API")
	Expect(deploymentResource.Webhooks.Defaulting).To(BeTrue(), "Deployment API should have a defaulting webhook")
	Expect(deploymentResource.Webhooks.Validation).To(BeTrue(), "Deployment API should have a defaulting webhook")
	Expect(deploymentResource.Webhooks.WebhookVersion).To(Equal("v1"), "Deployment API should have webhook version v1")

}
