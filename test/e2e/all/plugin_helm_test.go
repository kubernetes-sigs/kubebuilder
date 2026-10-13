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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/internal/helpers"
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

			helpers.CleanupHelmRelease(kbc)

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

			By("upgrading the release to 3 replicas and verifying " +
				"manager.replicas is respected (leader election ensures only one active)")
			Expect(kbc.HelmUpgradeReleaseWithReplicas(3)).To(Succeed())
			deploymentName := fmt.Sprintf("e2e-%s-controller-manager", kbc.TestSuffix)
			Eventually(func(g Gomega) {
				out, err := kbc.Kubectl.Get(
					true,
					"deployment", deploymentName,
					"-o", "jsonpath={.spec.replicas}",
				)
				g.Expect(err).NotTo(HaveOccurred())
				replicas, atoiErr := strconv.Atoi(strings.TrimSpace(out))
				g.Expect(atoiErr).NotTo(HaveOccurred(), "replicas field is not an integer")
				g.Expect(replicas).To(Equal(3), "expected deployment spec.replicas to be 3 after helm upgrade")
			}, helpers.DefaultTimeout, 30*time.Second).Should(Succeed())
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

		It("should not run the webhook server when installed with webhook.enabled=false", func() {
			By("generating a project with admission webhooks and no conversion webhook")
			helpers.GenerateV4WithoutConversionWebhook(kbc)

			By("building installer and generating helm chart")
			Expect(kbc.Make("build-installer")).To(Succeed())
			Expect(kbc.EditHelmPlugin()).To(Succeed())

			By("disabling webhooks and cert-manager in values.yaml")
			valuesPath := filepath.Join(kbc.Dir, "dist", "chart", "values.yaml")
			Expect(pluginutil.ReplaceInFile(valuesPath,
				"webhook:\n  enabled: true", "webhook:\n  enabled: false")).To(Succeed())
			Expect(pluginutil.ReplaceInFile(valuesPath,
				"certManager:\n  enabled: true", "certManager:\n  enabled: false")).To(Succeed())

			By("deploying without webhooks and validating the manager reconciles the sample CR")
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:          false,
				HasMetrics:          true,
				HasNetworkPolicies:  false,
				InstallMethod:       helpers.InstallMethodHelm,
				SkipChartGeneration: true, // Chart already generated and customized above
			})

			By("verifying the chart tells the manager to disable the webhook server")
			controllerPodName := helpers.GetControllerPodName(kbc)
			args, err := kbc.Kubectl.Get(true,
				"pod", controllerPodName, "-o", "jsonpath={.spec.containers[0].args}")
			Expect(err).NotTo(HaveOccurred())
			Expect(args).To(ContainSubstring("--webhook-port=-1"))

			By("verifying the container does not declare the webhook-server port")
			portNames, err := kbc.Kubectl.Get(true,
				"pod", controllerPodName, "-o", "jsonpath={.spec.containers[0].ports[*].name}")
			Expect(err).NotTo(HaveOccurred())
			Expect(portNames).NotTo(ContainSubstring("webhook-server"))

			By("verifying the manager did not start the webhook server")
			logs, err := kbc.Kubectl.Logs(controllerPodName)
			Expect(err).NotTo(HaveOccurred())
			Expect(logs).To(ContainSubstring("Webhook server is disabled"))
			Expect(logs).NotTo(ContainSubstring("Starting webhook server"))

			By("verifying the webhook Service was not rendered")
			_, err = kbc.Kubectl.Get(true,
				"service", fmt.Sprintf("e2e-%s-webhook-service", kbc.TestSuffix))
			Expect(err).To(HaveOccurred(), "webhook Service must not exist when webhook.enabled=false")
		})

		It("should serve the health probes on each health probe bind address the manager accepts", func() {
			By("generating a project without webhooks")
			helpers.GenerateV4WithoutWebhooks(kbc)

			// The flag, the "health" container port and both probes move together.
			By("moving the project's health probe server to port 9440")
			managerPath := filepath.Join(kbc.Dir, "config", "manager", "manager.yaml")
			Expect(pluginutil.ReplaceInFile(managerPath, "8081", "9440")).To(Succeed())

			By("building installer and generating helm chart")
			Expect(kbc.Make("build-installer")).To(Succeed())
			Expect(kbc.EditHelmPlugin()).To(Succeed())

			By("verifying the chart takes the project's address as its default")
			values, err := os.ReadFile(filepath.Join(kbc.Dir, "dist", "chart", "values.yaml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(values)).To(ContainSubstring(`healthProbeBindAddress: ":9440"`))

			By("deploying with the project's address and validating the manager reconciles the sample CR")
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:          false,
				HasMetrics:          true,
				HasNetworkPolicies:  false,
				InstallMethod:       helpers.InstallMethodHelm,
				SkipChartGeneration: true, // Chart already generated above
			})

			releaseName := fmt.Sprintf("e2e-%s", kbc.TestSuffix)
			deploymentName := releaseName + "-controller-manager"
			// controller-runtime logs the address its health probe server listens on.
			serverAddr := regexp.MustCompile(`"name":\s*"health probe",\s*"addr":\s*"([^"]+)"`)

			// expectHealthProbeServer checks the running manager. With an address, the server
			// listens on it, and the probes reach it through the "health" port: the rollout only
			// completes once the readiness probe passes. With "0" there is no server, no "health"
			// port and no probe. wantFlag "" means the chart passes no flag.
			expectHealthProbeServer := func(wantFlag, wantListen, wantPort string) {
				_, err = kbc.Kubectl.CommandInNamespace("rollout", "status",
					"deployment/"+deploymentName, "--timeout=5m")
				Expect(err).NotTo(HaveOccurred())
				// Wait for the previous Pod to be gone, since GetControllerPodName describes all
				// the Pods and fails when one is deleted meanwhile.
				Eventually(func(g Gomega) {
					pods, getErr := kbc.Kubectl.Get(true, "pods", "-l", "control-plane=controller-manager", "-o", "name")
					g.Expect(getErr).NotTo(HaveOccurred())
					g.Expect(strings.Fields(pods)).To(HaveLen(1))
				}, 2*time.Minute, time.Second).Should(Succeed())
				podName := helpers.GetControllerPodName(kbc)

				var args, port, probes, logs string
				args, err = kbc.Kubectl.Get(true, "pod", podName, "-o", "jsonpath={.spec.containers[0].args}")
				Expect(err).NotTo(HaveOccurred())
				if wantFlag == "" {
					Expect(args).NotTo(ContainSubstring("--health-probe-bind-address"))
				} else {
					Expect(args).To(ContainSubstring(`"--health-probe-bind-address=` + wantFlag + `"`))
				}

				port, err = kbc.Kubectl.Get(true, "pod", podName, "-o",
					`jsonpath={.spec.containers[0].ports[?(@.name=="health")].containerPort}`)
				Expect(err).NotTo(HaveOccurred())
				probes, err = kbc.Kubectl.Get(true, "pod", podName, "-o",
					"jsonpath={.spec.containers[0].livenessProbe.httpGet.port}"+
						"{.spec.containers[0].readinessProbe.httpGet.port}")
				Expect(err).NotTo(HaveOccurred())

				// The manager starts its HTTP servers before it tries to take the leader lease.
				Eventually(func(g Gomega) {
					logs, err = kbc.Kubectl.Logs(podName)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(logs).To(ContainSubstring("Attempting to acquire leader lease"))
				}, 2*time.Minute, time.Second).Should(Succeed())

				if wantListen == "" {
					Expect(port).To(BeEmpty())
					Expect(probes).To(BeEmpty())
					Expect(logs).NotTo(ContainSubstring("health probe"))
					return
				}
				Expect(port).To(Equal(wantPort))
				Expect(probes).To(Equal("healthhealth"))
				match := serverAddr.FindStringSubmatch(logs)
				Expect(match).NotTo(BeNil(), "the manager did not start the health probe server:\n%s", logs)
				Expect(match[1]).To(Equal(wantListen))
			}

			By("verifying the manager serves the probes on the project's address")
			expectHealthProbeServer(":9440", "[::]:9440", "9440")

			for _, step := range []struct {
				description, value, wantListen, wantPort string
				asString                                 bool
			}{
				{"another port", ":9441", "[::]:9441", "9441", false},
				// Go listens on 0.0.0.0 with a dual-stack socket, which it reports as [::].
				{"all IPv4 interfaces", "0.0.0.0:9442", "[::]:9442", "9442", false},
				{"all IPv6 interfaces", "[::]:9443", "[::]:9443", "9443", false},
				{"empty: no flag, the manager's default :8081", "", "[::]:8081", "8081", false},
				{"\"0\" as a string: turned off", "0", "", "", true},
				{"0 as a number: turned off", "0", "", "", false},
				{"the project's address again", ":9440", "[::]:9440", "9440", false},
			} {
				By("upgrading the release with manager.healthProbeBindAddress set to " + step.description)
				setFlag := "--set"
				if step.asString {
					setFlag = "--set-string"
				}
				cmd := exec.Command("helm", "upgrade", releaseName, "dist/chart",
					"--namespace", kbc.Kubectl.Namespace, "--reuse-values",
					setFlag, "manager.healthProbeBindAddress="+step.value)
				_, err = kbc.Run(cmd)
				Expect(err).NotTo(HaveOccurred())
				expectHealthProbeServer(step.value, step.wantListen, step.wantPort)
			}
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
			Eventually(verifyCRDsDeleted, helpers.DefaultTimeout, 30*time.Second).Should(Succeed())
		})
	})
})
