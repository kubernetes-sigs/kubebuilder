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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/helpers"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("kubebuilder", func() {
	Context("plugin helm/v2-alpha", func() {
		var kbc *utils.TestContext

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())

			By("installing Helm binary for chart operations")
			Expect(kbc.InstallHelm()).To(Succeed())
		})

		AfterEach(func() {
			By("removing restricted namespace label")
			_ = kbc.RemoveNamespaceLabelToEnforceRestricted()

			By("uninstalling Helm Release (if installed)")
			_ = kbc.UninstallHelmRelease()

			By("removing controller image and working dir")
			kbc.Destroy()
		})

		It("should generate a runnable project using webhooks and installed with the HelmChart", func() {
			helpers.GenerateV4(kbc)

			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         true,
				HasMetrics:         true,
				HasNetworkPolicies: false,
				InstallMethod:      helpers.InstallMethodHelm,
			})
		})

		It("should generate a runnable project without metrics exposed", func() {
			helpers.GenerateV4WithoutMetrics(kbc)

			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         true,
				HasMetrics:         false,
				HasNetworkPolicies: false,
				InstallMethod:      helpers.InstallMethodHelm,
			})
		})

		It("should generate a runnable project with metrics protected by network policies", func() {
			helpers.GenerateV4WithNetworkPoliciesWithoutWebhooks(kbc)

			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         false,
				HasMetrics:         true,
				HasNetworkPolicies: true,
				InstallMethod:      helpers.InstallMethodHelm,
			})
		})

		It("should generate a runnable project with webhooks and metrics protected by network policies", func() {
			helpers.GenerateV4WithNetworkPolicies(kbc)

			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         true,
				HasMetrics:         true,
				HasNetworkPolicies: true,
				InstallMethod:      helpers.InstallMethodHelm,
			})
		})

		It("should generate a runnable project with the manager running as restricted and without webhooks", func() {
			helpers.GenerateV4WithoutWebhooks(kbc)

			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         false,
				HasMetrics:         true,
				HasNetworkPolicies: false,
				InstallMethod:      helpers.InstallMethodHelm,
			})
		})

		It("should work with Helm chart customizations (fullnameOverride and cert-manager)", func() {
			By("generating a full-featured project with webhooks, metrics, and conversion webhooks")
			helpers.GenerateV4(kbc)

			By("building installer and generating helm chart")
			Expect(kbc.Make("build-installer")).To(Succeed())
			err := kbc.EditHelmPlugin()
			Expect(err).NotTo(HaveOccurred())

			By("customizing chart name via fullnameOverride to validate runtime behavior")
			valuesPath := filepath.Join(kbc.Dir, "dist", "chart", "values.yaml")
			err = pluginutil.ReplaceInFile(valuesPath,
				`# fullnameOverride: ""`,
				`fullnameOverride: "custom-operator"`)
			Expect(err).NotTo(HaveOccurred())

			By("deploying with custom chart name - validates cert-manager and all resources work correctly")
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:           true,
				HasMetrics:           true,
				HasNetworkPolicies:   false,
				InstallMethod:        helpers.InstallMethodHelm,
				HelmFullnameOverride: "custom-operator",
				SkipChartGeneration:  true, // Chart already generated and customized above
			})
		})

		It("should generate a namespeced runnable project using webhooks and installed with the HelmChart", func() {
			helpers.GenerateV4Namespaced(kbc)

			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         true,
				HasMetrics:         true,
				HasNetworkPolicies: false,
				IsNamespaced:       true,
				InstallMethod:      helpers.InstallMethodHelm,
			})
		})
	})
})
