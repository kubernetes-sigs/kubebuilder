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
			Expect(pluginutil.ReplaceInFile(managerManifest(kbc), scaffoldedHealthProbePort, "9440")).To(Succeed())

			By("building installer and generating helm chart")
			Expect(kbc.Make("build-installer")).To(Succeed())
			Expect(kbc.EditHelmPlugin()).To(Succeed())

			By("verifying the chart takes the project's address as its default")
			Expect(chartFile(kbc, "values.yaml")).To(ContainSubstring(`healthProbeBindAddress: ":9440"`))

			By("deploying with the project's address and validating the manager reconciles the sample CR")
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:          false,
				HasMetrics:          true,
				HasNetworkPolicies:  false,
				InstallMethod:       helpers.InstallMethodHelm,
				SkipChartGeneration: true, // Chart already generated above
			})

			By("verifying the manager serves the probes on the project's address")
			expectHealthProbeServer(kbc, ":9440", healthProbeServer{"9440", scaffoldedProbePaths})

			runHealthProbeSteps(kbc, []healthProbeStep{
				{"another port", ":9441", false, healthProbeServer{"9441", scaffoldedProbePaths}},
				// Go listens on 0.0.0.0 with a dual-stack socket, which it reports as [::].
				{"all IPv4 interfaces", "0.0.0.0:9442", false, healthProbeServer{"9442", scaffoldedProbePaths}},
				{"all IPv6 interfaces", "[::]:9443", false, healthProbeServer{"9443", scaffoldedProbePaths}},
				{
					"empty: no flag, the manager's default :8081", "", false,
					healthProbeServer{scaffoldedHealthProbePort, scaffoldedProbePaths},
				},
				{"\"0\" as a string: turned off", "0", true, healthProbeServer{}},
				{"0 as a number: turned off", "0", false, healthProbeServer{}},
				{"the project's address again", ":9440", false, healthProbeServer{"9440", scaffoldedProbePaths}},
			})
		})

		// The helm plugin builds the chart from the kustomize output, never from cmd/main.go, so
		// each change below in cmd/main.go goes with the change config/manager/manager.yaml needs
		// for a kustomize deploy to work.
		It("should serve the health probes of a manager with its own default address and probe endpoints", func() {
			const ownPort, renamedProbePaths = "9450", "/livez/ready"

			By("generating a project without webhooks")
			helpers.GenerateV4WithoutWebhooks(kbc)

			By("moving the default address to :9450 and renaming the probe endpoints in cmd/main.go")
			Expect(pluginutil.ReplaceInFile(mainGo(kbc),
				`"health-probe-bind-address", ":8081",`, `"health-probe-bind-address", ":9450",`)).To(Succeed())
			Expect(pluginutil.ReplaceInFile(mainGo(kbc), "HealthProbeBindAddress: probeAddr,\n",
				"HealthProbeBindAddress: probeAddr,\n"+
					"\t\tLivenessEndpointName:   \"/livez\",\n"+
					"\t\tReadinessEndpointName:  \"/ready\",\n")).To(Succeed())

			By("passing no flag in config/manager/manager.yaml, and moving its port and probes")
			removeHealthProbeFlag(kbc)
			Expect(pluginutil.ReplaceInFile(managerManifest(kbc), scaffoldedHealthProbePort, ownPort)).To(Succeed())
			Expect(pluginutil.ReplaceInFile(managerManifest(kbc), "path: /healthz", "path: /livez")).To(Succeed())
			Expect(pluginutil.ReplaceInFile(managerManifest(kbc), "path: /readyz", "path: /ready")).To(Succeed())

			By("building installer and generating helm chart")
			Expect(kbc.Make("build-installer")).To(Succeed())
			Expect(kbc.EditHelmPlugin()).To(Succeed())

			By("verifying the chart passes no flag and takes the port of the project's manifest")
			values := chartFile(kbc, "values.yaml")
			Expect(values).To(ContainSubstring(`healthProbeBindAddress: ""`))
			Expect(values).To(ContainSubstring("Empty: no flag, the manager's default :" + ownPort + "."))

			By("deploying and validating the manager reconciles the sample CR")
			helpers.Run(kbc, helpers.RunOptions{
				HasMetrics:             true,
				InstallMethod:          helpers.InstallMethodHelm,
				SkipChartGeneration:    true,
				WithoutHealthProbeFlag: true,
			})

			By("verifying the manager serves the renamed probes on the default in its cmd/main.go")
			expectHealthProbeServer(kbc, "", healthProbeServer{ownPort, renamedProbePaths})

			runHealthProbeSteps(kbc, []healthProbeStep{
				{"another port", ":9451", false, healthProbeServer{"9451", renamedProbePaths}},
				{"0: turned off", "0", false, healthProbeServer{}},
				{"empty: no flag, the default in cmd/main.go", "", false, healthProbeServer{ownPort, renamedProbePaths}},
			})
		})

		It("should run a manager without the health probe flag and without the Ping checks", func() {
			const ownPort = "8082"

			By("generating a project without webhooks")
			helpers.GenerateV4WithoutWebhooks(kbc)

			By("setting the address in cmd/main.go, and removing the flag and both healthz.Ping checks")
			Expect(pluginutil.ReplaceInFile(mainGo(kbc),
				`flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to. "+
		"Set 0 to disable the health probe server.")
`, "probeAddr = \":"+ownPort+"\"\n")).To(Succeed())
			Expect(pluginutil.ReplaceInFile(mainGo(kbc),
				`	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "Failed to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "Failed to set up ready check")
		os.Exit(1)
	}
`, "")).To(Succeed())
			Expect(pluginutil.ReplaceInFile(mainGo(kbc),
				"\t\"sigs.k8s.io/controller-runtime/pkg/healthz\"\n", "")).To(Succeed())

			// Without the checks, /healthz and /readyz answer 404, so the probes go too.
			By("passing no flag in config/manager/manager.yaml, moving its port, and removing its probes")
			removeHealthProbeFlag(kbc)
			Expect(pluginutil.ReplaceInFile(managerManifest(kbc), scaffoldedHealthProbePort, ownPort)).To(Succeed())
			Expect(pluginutil.ReplaceRegexInFile(managerManifest(kbc),
				`(?m)^        (?:livenessProbe|readinessProbe):\n(?:          .*\n)+`, "")).To(Succeed())

			By("building installer and generating helm chart")
			Expect(kbc.Make("build-installer")).To(Succeed())
			Expect(kbc.EditHelmPlugin()).To(Succeed())

			By("verifying the chart passes no flag, takes the port of the project's manifest and has no probes")
			values := chartFile(kbc, "values.yaml")
			Expect(values).To(ContainSubstring(`healthProbeBindAddress: ""`))
			Expect(values).To(ContainSubstring("Empty: no flag, the manager's default :" + ownPort + "."))
			Expect(chartFile(kbc, "templates", "manager", "manager.yaml")).NotTo(ContainSubstring("Probe:"))

			By("deploying and validating the manager reconciles the sample CR")
			helpers.Run(kbc, helpers.RunOptions{
				HasMetrics:             true,
				InstallMethod:          helpers.InstallMethodHelm,
				SkipChartGeneration:    true,
				WithoutHealthProbeFlag: true,
			})

			By("verifying the manager serves its health probe server on the address set in cmd/main.go")
			expectHealthProbeServer(kbc, "", healthProbeServer{port: ownPort})
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

const (
	// scaffoldedHealthProbePort is the health probe port go/v4 scaffolds.
	scaffoldedHealthProbePort = "8081"
	// scaffoldedProbePaths are the liveness and readiness probe paths go/v4 scaffolds.
	scaffoldedProbePaths = "/healthz/readyz"
)

// healthProbeServerAddr matches the address controller-runtime logs for its health probe server.
var healthProbeServerAddr = regexp.MustCompile(`"name":\s*"health probe",\s*"addr":\s*"([^"]+)"`)

// healthProbeServer is what a running manager serves. The zero value is a manager whose health
// probe server is off.
type healthProbeServer struct {
	port       string // the port the server listens on, which the "health" port declares
	probePaths string // the liveness and readiness probe paths, "" when the manager has none
}

// healthProbeStep upgrades the release with manager.healthProbeBindAddress set to value, which
// the chart passes as the flag, and checks the manager then serves want.
type healthProbeStep struct {
	description, value string
	asString           bool // set the value with --set-string
	want               healthProbeServer
}

func mainGo(kbc *utils.TestContext) string {
	return filepath.Join(kbc.Dir, "cmd", "main.go")
}

func managerManifest(kbc *utils.TestContext) string {
	return filepath.Join(kbc.Dir, "config", "manager", "manager.yaml")
}

// chartFile returns a file of the generated chart.
func chartFile(kbc *utils.TestContext, path ...string) string {
	content, err := os.ReadFile(filepath.Join(append([]string{kbc.Dir, "dist", "chart"}, path...)...))
	Expect(err).NotTo(HaveOccurred())
	return string(content)
}

// removeHealthProbeFlag removes the scaffolded --health-probe-bind-address from the manager's args.
func removeHealthProbeFlag(kbc *utils.TestContext) {
	Expect(pluginutil.ReplaceRegexInFile(managerManifest(kbc),
		`(?m)^[ \t]*- --health-probe-bind-address=:8081\n`, "")).To(Succeed())
}

func runHealthProbeSteps(kbc *utils.TestContext, steps []healthProbeStep) {
	releaseName := fmt.Sprintf("e2e-%s", kbc.TestSuffix)
	for _, step := range steps {
		By("upgrading the release with manager.healthProbeBindAddress set to " + step.description)
		setFlag := "--set"
		if step.asString {
			setFlag = "--set-string"
		}
		cmd := exec.Command("helm", "upgrade", releaseName, "dist/chart",
			"--namespace", kbc.Kubectl.Namespace, "--reuse-values",
			setFlag, "manager.healthProbeBindAddress="+step.value)
		_, err := kbc.Run(cmd)
		Expect(err).NotTo(HaveOccurred())
		expectHealthProbeServer(kbc, step.value, step.want)
	}
}

// expectHealthProbeServer checks the running manager gets flag ("" for none) and serves want. The
// server listens on all interfaces, which Go reports as [::], and the probes reach it through the
// "health" port: the rollout only completes once the readiness probe passes. When the server is
// off there is no "health" port and no probe.
func expectHealthProbeServer(kbc *utils.TestContext, flag string, want healthProbeServer) {
	deploymentName := fmt.Sprintf("e2e-%s-controller-manager", kbc.TestSuffix)
	_, err := kbc.Kubectl.CommandInNamespace("rollout", "status",
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

	args, err := kbc.Kubectl.Get(true, "pod", podName, "-o", "jsonpath={.spec.containers[0].args}")
	Expect(err).NotTo(HaveOccurred())
	if flag == "" {
		Expect(args).NotTo(ContainSubstring("--health-probe-bind-address"))
	} else {
		Expect(args).To(ContainSubstring(`"--health-probe-bind-address=` + flag + `"`))
	}

	port, err := kbc.Kubectl.Get(true, "pod", podName, "-o",
		`jsonpath={.spec.containers[0].ports[?(@.name=="health")].containerPort}`)
	Expect(err).NotTo(HaveOccurred())
	probes, err := kbc.Kubectl.Get(true, "pod", podName, "-o",
		"jsonpath={.spec.containers[0].livenessProbe.httpGet.port}"+
			"{.spec.containers[0].readinessProbe.httpGet.port}"+
			"{.spec.containers[0].livenessProbe.httpGet.path}"+
			"{.spec.containers[0].readinessProbe.httpGet.path}")
	Expect(err).NotTo(HaveOccurred())

	// The manager starts its HTTP servers before it tries to take the leader lease.
	var logs string
	Eventually(func(g Gomega) {
		logs, err = kbc.Kubectl.Logs(podName)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(logs).To(ContainSubstring("Attempting to acquire leader lease"))
	}, 2*time.Minute, time.Second).Should(Succeed())

	if want.port == "" {
		Expect(port).To(BeEmpty())
		Expect(probes).To(BeEmpty())
		Expect(logs).NotTo(ContainSubstring("health probe"))
		return
	}
	Expect(port).To(Equal(want.port))
	if want.probePaths == "" {
		Expect(probes).To(BeEmpty())
	} else {
		Expect(probes).To(Equal("healthhealth" + want.probePaths))
	}
	match := healthProbeServerAddr.FindStringSubmatch(logs)
	Expect(match).NotTo(BeNil(), "the manager did not start the health probe server:\n%s", logs)
	Expect(match[1]).To(Equal("[::]:" + want.port))
}
