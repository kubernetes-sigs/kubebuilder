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

package all

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/helpers"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

// Test specs for helm/v2-alpha plugin
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

			By("cleaning up CRDs that were preserved by crd.keep=true")
			domainSuffix := fmt.Sprintf(".example.com%s", kbc.TestSuffix)
			listCmd := exec.Command("kubectl", "get", "crds", "-o", "name")
			if output, err := kbc.Run(listCmd); err == nil {
				for crdName := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
					if crdName != "" && strings.Contains(crdName, domainSuffix) {
						deleteCmd := exec.Command("kubectl", "delete", crdName, "--ignore-not-found")
						_, _ = kbc.Run(deleteCmd)
					}
				}
			}

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

		It("should delete CRDs on helm uninstall when crd.keep=false", func() {
			By("generating a project with webhooks")
			helpers.GenerateV4(kbc)

			By("building installer and generating helm chart")
			Expect(kbc.Make("build-installer")).To(Succeed())
			err := kbc.EditHelmPlugin()
			Expect(err).NotTo(HaveOccurred())

			By("installing helm chart with crd.keep=false")
			Expect(kbc.HelmInstallReleaseWithOptions(false)).To(Succeed())

			By("verifying CRDs exist after install")
			domainSuffix := fmt.Sprintf(".example.com%s", kbc.TestSuffix)
			verifyCRDsExist := func(g Gomega) {
				listCmd := exec.Command("kubectl", "get", "crds", "-o", "name")
				output, err := kbc.Run(listCmd)
				g.Expect(err).NotTo(HaveOccurred(), "failed to list CRDs")
				g.Expect(string(output)).To(ContainSubstring(domainSuffix),
					"expected CRDs matching domain suffix %s to exist", domainSuffix)
			}
			verifyCRDsExist(Default)

			By("uninstalling helm release")
			Expect(kbc.UninstallHelmRelease()).To(Succeed())

			By("verifying CRDs are deleted after uninstall (crd.keep=false)")
			verifyCRDsDeleted := func(g Gomega) {
				listCmd := exec.Command("kubectl", "get", "crds", "-o", "name")
				output, err := kbc.Run(listCmd)
				if err != nil {
					// If we can't list CRDs, assume they're gone
					return
				}
				for crdName := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
					g.Expect(crdName).NotTo(ContainSubstring(domainSuffix),
						"CRD %s still exists but should have been deleted", crdName)
				}
			}
			Eventually(verifyCRDsDeleted, "60s", "2s").Should(Succeed())
		})
	})
})
