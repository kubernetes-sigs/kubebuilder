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

package all

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/internal/helpers"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

// Test specs for go/v4 plugin
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

		It("should deploy admission and conversion webhooks protected by network policies", func() {
			helpers.GenerateV4WithNetworkPolicies(kbc)
			helpers.EnableWebhookNamespaceGating(kbc)
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:             true,
				HasMetrics:             true,
				HasNetworkPolicies:     true,
				WebhookNamespaceGating: true,
				InstallMethod:          helpers.InstallMethodKustomize,
			})
		})

		It("should generate a runnable project with a custom webhook port protected by network policies", func() {
			helpers.GenerateV4WithNetworkPolicies(kbc)

			By("configuring the manager, webhook Service, and webhook NetworkPolicy to use a custom port")
			const customWebhookPort = "9444"
			Expect(util.ReplaceInFile(
				filepath.Join(kbc.Dir, "config", "default", "manager_webhook_patch.yaml"),
				"9443", customWebhookPort)).To(Succeed())
			Expect(util.ReplaceInFile(
				filepath.Join(kbc.Dir, "config", "webhook", "service.yaml"),
				"targetPort: 9443", "targetPort: "+customWebhookPort)).To(Succeed())
			// The webhook NetworkPolicy must allow the same pod port, otherwise a CNI that
			// enforces NetworkPolicies would block admission traffic to the custom port.
			Expect(util.ReplaceInFile(
				filepath.Join(kbc.Dir, "config", "network-policy", "allow-webhook-traffic.yaml"),
				"port: 9443", "port: "+customWebhookPort)).To(Succeed())

			By("deploying with the custom webhook port and validating all webhook flows")
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         true,
				HasMetrics:         true,
				HasNetworkPolicies: true,
				WebhookPort:        9444,
				InstallMethod:      helpers.InstallMethodKustomize,
			})

			By("verifying the manager is configured with the custom webhook port")
			controllerPodName := helpers.GetControllerPodName(kbc)
			args, err := kbc.Kubectl.Get(true,
				"pod", controllerPodName, "-o", "jsonpath={.spec.containers[0].args}")
			Expect(err).NotTo(HaveOccurred())
			Expect(args).To(ContainSubstring("--webhook-port=" + customWebhookPort))
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

		It("should generate a runnable project with Server-Side Apply (--ssa)", func() {
			helpers.GenerateV4WithSSA(kbc)
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         false,
				HasMetrics:         true,
				HasNetworkPolicies: false,
				InstallMethod:      helpers.InstallMethodKustomize,
			})
		})

		It("should generate a runnable project with cluster-scoped "+
			"Server-Side Apply (--ssa --namespaced=false)", func() {
			helpers.GenerateV4WithSSAClusterScoped(kbc)
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         false,
				HasMetrics:         true,
				HasNetworkPolicies: false,
				InstallMethod:      helpers.InstallMethodKustomize,
			})
		})
	})
})
