/*
Copyright 2020 The Kubernetes Authors.

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

package v4

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/helpers"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("kubebuilder", func() {
	Context("plugin go/v4", func() {
		var kbc *utils.TestContext

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())
		})

		AfterEach(func() {
			By("removing restricted namespace label")
			_ = kbc.RemoveNamespaceLabelToEnforceRestricted()

			By("undeploy the project")
			_ = kbc.Make("undeploy")

			By("uninstalling the project")
			_ = kbc.Make("uninstall")

			By("removing controller image and working dir")
			kbc.Destroy()
		})
		It("should generate a runnable project", func() {
			helpers.GenerateV4(kbc)
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         true,
				HasMetrics:         true,
				HasNetworkPolicies: false,
				InstallMethod:      helpers.InstallMethodKustomize,
			})
		})
		It("should generate a runnable project with the Installer", func() {
			helpers.GenerateV4(kbc)
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         true,
				HasMetrics:         true,
				HasNetworkPolicies: false,
				InstallMethod:      helpers.InstallMethodInstaller,
			})
		})
		It("should generate a runnable project without metrics exposed", func() {
			helpers.GenerateV4WithoutMetrics(kbc)
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         true,
				HasMetrics:         false,
				HasNetworkPolicies: false,
				InstallMethod:      helpers.InstallMethodKustomize,
			})
		})
		It("should generate a runnable project with metrics protected by network policies", func() {
			helpers.GenerateV4WithNetworkPoliciesWithoutWebhooks(kbc)
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         false,
				HasMetrics:         true,
				HasNetworkPolicies: true,
				InstallMethod:      helpers.InstallMethodKustomize,
			})
		})
		It("should generate a runnable project with webhooks and metrics protected by network policies", func() {
			helpers.GenerateV4WithNetworkPolicies(kbc)
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         true,
				HasMetrics:         true,
				HasNetworkPolicies: true,
				InstallMethod:      helpers.InstallMethodKustomize,
			})
		})
		It("should generate a runnable project with the manager running "+
			"as restricted and without webhooks", func() {
			helpers.GenerateV4WithoutWebhooks(kbc)
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         false,
				HasMetrics:         true,
				HasNetworkPolicies: false,
				InstallMethod:      helpers.InstallMethodKustomize,
			})
		})
		It("should generate a runnable project with custom webhook paths", func() {
			helpers.GenerateV4WithCustomWebhookPath(kbc)
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         true,
				HasMetrics:         true,
				HasNetworkPolicies: false,
				InstallMethod:      helpers.InstallMethodKustomize,
			})
		})
		It("should generate a runnable project", func() {
			helpers.GenerateV4Namespaced(kbc)
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         true,
				HasMetrics:         true,
				HasNetworkPolicies: false,
				IsNamespaced:       true,
				InstallMethod:      helpers.InstallMethodKustomize,
			})
		})
	})
})
