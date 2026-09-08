/*
Copyright 2026 The Kubernetes Authors.

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
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/internal/helpers"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

// Test specs for the scaffolded NetworkPolicies.
//
// Every scenario runs once deployed with kustomize and once installed with the Helm chart.
// Namespace gating and custom ports share a deployment because each deployment costs
// minutes and the two do not interact. The Helm custom-ports spec builds the chart from a
// project without network policies and turns them on in values.yaml, so the chart's own
// policy templates are exercised as well as the ones derived from the kustomize output.
var _ = Describe("kubebuilder", func() {
	Context("network policies", func() {
		var kbc *utils.TestContext
		var installMethod helpers.InstallMethod

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())
			installMethod = ""
		})

		AfterEach(func() {
			By("removing restricted namespace label")
			_ = kbc.RemoveNamespaceLabelToEnforceRestricted()

			if installMethod == helpers.InstallMethodHelm {
				helpers.CleanupHelmRelease(kbc)
			} else {
				By("undeploy the project")
				_ = kbc.Make("undeploy")

				By("uninstalling the project")
				_ = kbc.Make("uninstall")
			}

			By("removing controller image and working dir")
			kbc.Destroy()
		})

		// useInstallMethod records the method so AfterEach runs the matching cleanup.
		// Helm is only installed for chart specs so the kustomize specs do not pay for it.
		useInstallMethod := func(method helpers.InstallMethod) {
			installMethod = method
			if method == helpers.InstallMethodHelm {
				By("installing Helm binary for chart operations")
				Expect(kbc.InstallHelm()).To(Succeed())
			}
		}

		DescribeTable("should protect metrics",
			func(method helpers.InstallMethod) {
				useInstallMethod(method)
				helpers.GenerateV4WithNetworkPoliciesWithoutWebhooks(kbc)
				helpers.Run(kbc, helpers.RunOptions{
					HasWebhook:         false,
					HasMetrics:         true,
					HasNetworkPolicies: true,
					InstallMethod:      method,
				})
			},
			Entry("when deployed with kustomize", helpers.InstallMethodKustomize),
			Entry("when installed with the Helm chart", helpers.InstallMethodHelm),
		)

		It("should protect namespace-gated webhooks on a custom webhook port "+
			"when deployed with kustomize", func() {
			useInstallMethod(helpers.InstallMethodKustomize)
			helpers.GenerateV4WithNetworkPolicies(kbc)
			helpers.EnableWebhookNamespaceGating(kbc)

			By("configuring the manager, webhook Service, and webhook NetworkPolicy to use a custom port")
			const customWebhookPort = 9444
			webhookPort := strconv.Itoa(customWebhookPort)
			Expect(pluginutil.ReplaceInFile(
				filepath.Join(kbc.Dir, "config", "default", "manager_webhook_patch.yaml"),
				"9443", webhookPort)).To(Succeed())
			Expect(pluginutil.ReplaceInFile(
				filepath.Join(kbc.Dir, "config", "webhook", "service.yaml"),
				"targetPort: 9443", "targetPort: "+webhookPort)).To(Succeed())
			// The webhook NetworkPolicy must allow the same pod port, otherwise a CNI that
			// enforces NetworkPolicies would block admission traffic to the custom port.
			Expect(pluginutil.ReplaceInFile(
				filepath.Join(kbc.Dir, "config", "network-policy", "allow-webhook-traffic.yaml"),
				"port: 9443", "port: "+webhookPort)).To(Succeed())

			By("deploying with the custom webhook port and validating all webhook flows")
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:             true,
				HasMetrics:             true,
				HasNetworkPolicies:     true,
				WebhookNamespaceGating: true,
				WebhookPort:            customWebhookPort,
				InstallMethod:          helpers.InstallMethodKustomize,
			})

			By("verifying the manager is configured with the custom webhook port")
			controllerPodName := helpers.GetControllerPodName(kbc)
			args, err := kbc.Kubectl.Get(true,
				"pod", controllerPodName, "-o", "jsonpath={.spec.containers[0].args}")
			Expect(err).NotTo(HaveOccurred())
			Expect(args).To(ContainSubstring("--webhook-port=" + webhookPort))
		})

		It("should protect namespace-gated webhooks on custom metrics, health probe, and webhook ports "+
			"when installed with the Helm chart", func() {
			useInstallMethod(helpers.InstallMethodHelm)

			By("generating a full-featured project with webhooks, metrics and conversion webhooks")
			helpers.GenerateV4(kbc)
			helpers.EnableWebhookNamespaceGating(kbc)

			By("building installer and generating helm chart")
			Expect(kbc.Make("build-installer")).To(Succeed())
			Expect(kbc.EditHelmPlugin()).To(Succeed())

			// Each port value is wired into the manager, so overriding it in values.yaml
			// changes both the manager listener and the Kubernetes resources that target it.
			const (
				customMetricsPort     = 8444
				customHealthProbePort = 8082
				customWebhookPort     = 9444
			)

			By("overriding ports and enabling network policies in values.yaml")
			valuesPath := filepath.Join(kbc.Dir, "dist", "chart", "values.yaml")
			Expect(pluginutil.ReplaceInFile(valuesPath,
				"port: 8443", fmt.Sprintf("port: %d", customMetricsPort))).To(Succeed())
			Expect(pluginutil.ReplaceInFile(valuesPath,
				"port: 8081", fmt.Sprintf("port: %d", customHealthProbePort))).To(Succeed())
			Expect(pluginutil.ReplaceInFile(valuesPath,
				"port: 9443", fmt.Sprintf("port: %d", customWebhookPort))).To(Succeed())
			Expect(pluginutil.ReplaceInFile(valuesPath,
				"networkPolicy:\n  enabled: false", "networkPolicy:\n  enabled: true")).To(Succeed())

			By("deploying with the customized ports and validating the manager runs correctly")
			// The pod only turns Ready if the probe answers on customHealthProbePort, and the
			// NetworkPolicy check must probe that same port to prove it is protected.
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:             true,
				HasMetrics:             true,
				HasNetworkPolicies:     true,
				WebhookNamespaceGating: true,
				MetricsPort:            customMetricsPort,
				WebhookPort:            customWebhookPort,
				HealthProbePort:        customHealthProbePort,
				InstallMethod:          helpers.InstallMethodHelm,
				SkipChartGeneration:    true,
			})

			By("verifying the manager binds probes, metrics, and webhooks to the custom ports")
			controllerPodName := helpers.GetControllerPodName(kbc)
			args, err := kbc.Kubectl.Get(true,
				"pod", controllerPodName, "-o", "jsonpath={.spec.containers[0].args}")
			Expect(err).NotTo(HaveOccurred())
			Expect(args).To(ContainSubstring(
				fmt.Sprintf("--health-probe-bind-address=:%d", customHealthProbePort)))
			Expect(args).To(ContainSubstring(
				fmt.Sprintf("--metrics-bind-address=:%d", customMetricsPort)))
			Expect(args).To(ContainSubstring(fmt.Sprintf("--webhook-port=%d", customWebhookPort)))

			By("verifying the container declares the custom health probe port")
			ports, err := kbc.Kubectl.Get(true,
				"pod", controllerPodName, "-o", "jsonpath={.spec.containers[0].ports}")
			Expect(err).NotTo(HaveOccurred())
			Expect(ports).To(ContainSubstring(strconv.Itoa(customHealthProbePort)))
			Expect(ports).To(ContainSubstring(strconv.Itoa(customWebhookPort)))

			By("verifying the metrics Service exposes the custom port")
			namePrefix := fmt.Sprintf("e2e-%s", kbc.TestSuffix)
			Expect(helpers.GetMetricsServicePort(namePrefix, kbc)).To(Equal(customMetricsPort))
		})

		DescribeTable("should keep the manager isolated with metrics disabled",
			func(method helpers.InstallMethod) {
				useInstallMethod(method)
				helpers.GenerateV4WithNetworkPoliciesWithoutMetrics(kbc)
				helpers.Run(kbc, helpers.RunOptions{
					HasWebhook:         true,
					HasMetrics:         false,
					HasNetworkPolicies: true,
					InstallMethod:      method,
				})
			},
			Entry("when deployed with kustomize", helpers.InstallMethodKustomize),
			Entry("when installed with the Helm chart", helpers.InstallMethodHelm),
		)
	})
})
