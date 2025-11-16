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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

const (
	tokenRequestRawString = `{"apiVersion": "authentication.k8s.io/v1", "kind": "TokenRequest"}`
)

// tokenRequest is a trimmed down version of the authentication.k8s.io/v1/TokenRequest Type
// that we want to use for extracting the token.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}

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
			GenerateV4(kbc)
			Run(kbc, true, false, false, true, false)
		})
		It("should generate a runnable project with the Installer", func() {
			GenerateV4(kbc)
			Run(kbc, true, true, false, true, false)
		})
		It("should generate a runnable project without metrics exposed", func() {
			GenerateV4WithoutMetrics(kbc)
			Run(kbc, true, false, false, false, false)
		})
		It("should generate a runnable project with metrics protected by network policies", func() {
			GenerateV4WithNetworkPoliciesWithoutWebhooks(kbc)
			Run(kbc, false, false, false, true, true)
		})
		It("should generate a runnable project with webhooks and metrics protected by network policies", func() {
			GenerateV4WithNetworkPolicies(kbc)
			Run(kbc, true, false, false, true, true)
		})
		It("should generate a runnable project with the manager running "+
			"as restricted and without webhooks", func() {
			GenerateV4WithoutWebhooks(kbc)
			Run(kbc, false, false, false, true, false)
		})
		It("should generate a runnable project with custom webhook paths", func() {
			GenerateV4WithCustomWebhookPath(kbc)
			Run(kbc, true, false, false, true, false)
		})
	})
})

// Run runs a set of e2e tests for a scaffolded project defined by a TestContext.
func Run(kbc *utils.TestContext, hasWebhook, isToUseInstaller, isToUseHelmChart, hasMetrics bool,
	hasNetworkPolicies bool,
) {
	var controllerPodName string
	var err error

	By("creating manager namespace")
	err = kbc.CreateManagerNamespace()
	Expect(err).NotTo(HaveOccurred())

	By("labeling the namespace to enforce the restricted security policy")
	err = kbc.LabelNamespacesToEnforceRestricted()
	Expect(err).NotTo(HaveOccurred())

	By("updating the go.mod")
	err = kbc.Tidy()
	Expect(err).NotTo(HaveOccurred())

	By("run make all")
	err = kbc.Make("all")
	Expect(err).NotTo(HaveOccurred())

	By("building the controller image")
	err = kbc.Make("docker-build", "IMG="+kbc.ImageName)
	Expect(err).NotTo(HaveOccurred())

	By("loading the controller docker image into the kind cluster")
	err = kbc.LoadImageToKindCluster()
	Expect(err).NotTo(HaveOccurred())

	if !isToUseInstaller && !isToUseHelmChart {
		By("deploying the controller-manager")
		cmd := exec.Command("make", "deploy", "IMG="+kbc.ImageName)
		_, err = kbc.Run(cmd)
		Expect(err).NotTo(HaveOccurred())
	}

	if isToUseInstaller && !isToUseHelmChart {
		By("building the installer")
		err = kbc.Make("build-installer", "IMG="+kbc.ImageName)
		Expect(err).NotTo(HaveOccurred())

		By("deploying the controller-manager with the installer")
		_, err = kbc.Kubectl.Apply(true, "-f", "dist/install.yaml")
		Expect(err).NotTo(HaveOccurred())
	}
	By("Checking controllerManager and getting the name of the Pod")
	controllerPodName = getControllerName(kbc)

	By("Checking if all flags are applied to the manager pod")
	podOutput, err := kbc.Kubectl.Get(
		true,
		"pod", controllerPodName,
		"-o", "jsonpath={.spec.containers[0].args}",
	)
	Expect(err).NotTo(HaveOccurred())
	Expect(podOutput).To(ContainSubstring("leader-elect"),
		"Expected manager pod to have --leader-elect flag")
	Expect(podOutput).To(ContainSubstring("health-probe-bind-address"),
		"Expected manager pod to have --health-probe-bind-address flag")

	By("validating that the Prometheus manager has provisioned the Service")
	Eventually(func(g Gomega) {
		_, err = kbc.Kubectl.Get(
			false,
			"Service", "prometheus-operator")
		g.Expect(err).NotTo(HaveOccurred())
	}, time.Minute, time.Second).Should(Succeed())

	By("validating that the ServiceMonitor for Prometheus is applied in the namespace")
	_, err = kbc.Kubectl.Get(
		true,
		"ServiceMonitor")
	Expect(err).NotTo(HaveOccurred())

	if hasNetworkPolicies {
		if hasMetrics {
			By("labeling the namespace to allow consume the metrics")
			Expect(kbc.Kubectl.Command("label", "namespaces", kbc.Kubectl.Namespace,
				"metrics=enabled")).Error().NotTo(HaveOccurred())

			By("Ensuring the Allow Metrics Traffic NetworkPolicy exists", func() {
				var output string
				output, err = kbc.Kubectl.Get(
					true,
					"networkpolicy", fmt.Sprintf("e2e-%s-allow-metrics-traffic", kbc.TestSuffix),
				)
				Expect(err).NotTo(HaveOccurred(), "NetworkPolicy allow-metrics-traffic should exist in the namespace")
				Expect(output).To(ContainSubstring("allow-metrics-traffic"), "NetworkPolicy allow-metrics-traffic "+
					"should be present in the output")
			})
		}

		if hasWebhook {
			By("labeling the namespace to allow webhooks traffic")
			_, err = kbc.Kubectl.Command("label", "namespaces", kbc.Kubectl.Namespace,
				"webhook=enabled")
			Expect(err).NotTo(HaveOccurred())

			By("Ensuring the allow-webhook-traffic NetworkPolicy exists", func() {
				var output string
				output, err = kbc.Kubectl.Get(
					true,
					"networkpolicy", fmt.Sprintf("e2e-%s-allow-webhook-traffic", kbc.TestSuffix),
				)
				Expect(err).NotTo(HaveOccurred(), "NetworkPolicy allow-webhook-traffic should exist in the namespace")
				Expect(output).To(ContainSubstring("allow-webhook-traffic"), "NetworkPolicy allow-webhook-traffic "+
					"should be present in the output")
			})
		}
	}

	if hasWebhook {
		By("validating that cert-manager has provisioned the certificate Secret")

		verifyWebhookCert := func(g Gomega) {
			var output string
			output, err = kbc.Kubectl.Get(
				true,
				"secrets", "webhook-server-cert")
			g.Expect(err).ToNot(HaveOccurred(), "webhook-server-cert should exist in the namespace")
			g.Expect(output).To(ContainSubstring("webhook-server-cert"))
		}

		Eventually(verifyWebhookCert, time.Minute, time.Second).Should(Succeed())

		By("validating that the mutating|validating webhooks have the CA injected")
		verifyCAInjection := func(g Gomega) {
			var mwhOutput, vwhOutput string
			mwhOutput, err = kbc.Kubectl.Get(
				false,
				"mutatingwebhookconfigurations.admissionregistration.k8s.io",
				fmt.Sprintf("e2e-%s-mutating-webhook-configuration", kbc.TestSuffix),
				"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
			g.Expect(err).NotTo(HaveOccurred())
			// check that ca should be long enough, because there may be a place holder "\n"
			g.Expect(len(mwhOutput)).To(BeNumerically(">", 10))

			vwhOutput, err = kbc.Kubectl.Get(
				false,
				"validatingwebhookconfigurations.admissionregistration.k8s.io",
				fmt.Sprintf("e2e-%s-validating-webhook-configuration", kbc.TestSuffix),
				"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
			g.Expect(err).NotTo(HaveOccurred())
			// check that ca should be long enough, because there may be a place holder "\n"
			g.Expect(len(vwhOutput)).To(BeNumerically(">", 10))
		}

		Eventually(verifyCAInjection, time.Minute, time.Second).Should(Succeed())

		By("validating that the CA injection is applied for CRD conversion")
		crdKind := "ConversionTest"
		verifyCAInjection = func(g Gomega) {
			var crdOutput string
			crdOutput, err = kbc.Kubectl.Get(
				false,
				"customresourcedefinition.apiextensions.k8s.io",
				"-o", fmt.Sprintf(
					"jsonpath={.items[?(@.spec.names.kind=='%s')].spec.conversion.webhook.clientConfig.caBundle}",
					crdKind),
			)
			g.Expect(err).NotTo(HaveOccurred(),
				"failed to get CRD conversion webhook configuration")

			// Check if the CA bundle is populated (length > 10 to avoid placeholder values)
			g.Expect(len(crdOutput)).To(BeNumerically(">", 10),
				"CA bundle should be injected into the CRD")
		}
		Eventually(verifyCAInjection, time.Minute, time.Second).Should(Succeed(),
			"CA injection validation failed")
	}

	By("creating an instance of the CR")
	// currently controller-runtime doesn't provide a readiness probe, we retry a few times
	// we can change it to probe the readiness endpoint after CR supports it.
	sampleFile := filepath.Join("config", "samples",
		fmt.Sprintf("%s_%s_%s.yaml", kbc.Group, kbc.Version, strings.ToLower(kbc.Kind)))
	sampleFilePath, err := filepath.Abs(filepath.Join(fmt.Sprintf("e2e-%s", kbc.TestSuffix), sampleFile))
	Expect(err).To(Not(HaveOccurred()))

	f, err := os.OpenFile(sampleFilePath, os.O_APPEND|os.O_WRONLY, 0o644)
	Expect(err).To(Not(HaveOccurred()))
	defer func() {
		err = f.Close()
		Expect(err).To(Not(HaveOccurred()))
	}()
	_, err = f.WriteString("  foo: bar")
	Expect(err).To(Not(HaveOccurred()))

	applySample := func(g Gomega) {
		g.Expect(kbc.Kubectl.Apply(true, "-f", sampleFile)).
			Error().NotTo(HaveOccurred())
	}
	Eventually(applySample, time.Minute, time.Second).Should(Succeed())

	if hasMetrics {
		By("checking the metrics values to validate that the created resource object gets reconciled")
		metricsOutput := getMetricsOutput(controllerPodName, kbc)
		Expect(metricsOutput).To(ContainSubstring(fmt.Sprintf(
			`controller_runtime_reconcile_total{controller="%s",result="success"} 1`,
			strings.ToLower(kbc.Kind),
		)))
	}

	if !hasMetrics {
		By("validating the metrics endpoint is not working as expected")
		metricsShouldBeUnavailable(kbc)
	}

	if hasWebhook {
		By("validating that mutating and validating webhooks are working fine")
		cnt, err := kbc.Kubectl.Get(
			true,
			"-f", sampleFile,
			"-o", "go-template={{ .spec.count }}")
		Expect(err).NotTo(HaveOccurred())
		count, err := strconv.Atoi(cnt)
		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(BeNumerically("==", 5))
	}

	if hasWebhook {
		By("creating a namespace")
		namespace := "test-webhooks"
		_, err := kbc.Kubectl.Command("create", "namespace", namespace)
		Expect(err).To(Not(HaveOccurred()), "namespace should be created successfully")

		By("applying the CR in the created namespace")

		applySampleNamespaced := func(g Gomega) {
			_, err = kbc.Kubectl.Apply(false, "-n", namespace, "-f", sampleFile)
			g.Expect(err).To(Not(HaveOccurred()))
		}
		Eventually(applySampleNamespaced, 2*time.Minute, time.Second).Should(Succeed())

		By("validating that mutating webhooks are working fine outside of the manager's namespace")
		cnt, err := kbc.Kubectl.Get(
			false,
			"-n", namespace,
			"-f", sampleFile,
			"-o", "go-template={{ .spec.count }}")
		Expect(err).NotTo(HaveOccurred())

		count, err := strconv.Atoi(cnt)
		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(BeNumerically("==", 5),
			"the mutating webhook should set the count to 5")

		By("removing the namespace")
		Expect(kbc.Kubectl.Command("delete", "namespace", namespace)).
			Error().NotTo(HaveOccurred(), "namespace should be removed successfully")

		By("validating the conversion")

		// Update the ConversionTest CR sample in v1 to set a specific `size`
		By("modifying the ConversionTest CR sample to set `size` for conversion testing")
		conversionCRFile := filepath.Join("config", "samples",
			fmt.Sprintf("%s_v1_conversiontest.yaml", kbc.Group))
		conversionCRPath := filepath.Join(kbc.Dir, conversionCRFile)

		// Edit the file to include `size` in the spec field for v1
		err = util.ReplaceInFile(conversionCRPath, "# TODO(user): Add fields here", `size: 3`)
		Expect(err).NotTo(HaveOccurred(), "failed to replace spec in ConversionTest CR sample")

		// Apply the ConversionTest Custom Resource in v1
		By("applying the modified ConversionTest CR in v1 for conversion")
		_, err = kbc.Kubectl.Apply(true, "-f", conversionCRPath)
		Expect(err).NotTo(HaveOccurred(), "failed to apply modified ConversionTest CR")

		By("waiting for the ConversionTest CR to appear")
		Eventually(func(g Gomega) {
			_, err := kbc.Kubectl.Get(true, "conversiontest", "conversiontest-sample")
			g.Expect(err).NotTo(HaveOccurred(), "expected the ConversionTest CR to exist")
		}, time.Minute, time.Second).Should(Succeed())

		By("validating that the converted resource in v2 has replicas == 3")
		Eventually(func(g Gomega) {
			out, err := kbc.Kubectl.Get(
				true,
				"conversiontest", "conversiontest-sample",
				"-o", "jsonpath={.spec.replicas}",
			)
			g.Expect(err).NotTo(HaveOccurred(), "failed to get converted resource in v2")
			replicas, err := strconv.Atoi(out)
			g.Expect(err).NotTo(HaveOccurred(), "replicas field is not an integer")
			g.Expect(replicas).To(Equal(3), "expected replicas to be 3 after conversion")
		}, time.Minute, time.Second).Should(Succeed())

		if hasMetrics {
			By("validating conversion metrics to confirm conversion operations")
			metricsOutput := getMetricsOutput(controllerPodName, kbc)
			conversionMetric := `controller_runtime_reconcile_total{controller="conversiontest",result="success"} 1`
			Expect(metricsOutput).To(ContainSubstring(conversionMetric),
				"Expected metric for successful ConversionTest reconciliation")
		}
	}
}

func getControllerName(kbc *utils.TestContext) string {
	By("validating that the controller-manager pod is running as expected")
	var controllerPodName string
	verifyControllerUp := func(g Gomega) error {
		// Get pod name
		podOutput, err := kbc.Kubectl.Get(
			true,
			"pods", "-l", "control-plane=controller-manager",
			"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}"+
				"{{ \"\\n\" }}{{ end }}{{ end }}")
		g.Expect(err).NotTo(HaveOccurred())
		podNames := util.GetNonEmptyLines(podOutput)
		if len(podNames) != 1 {
			return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
		}
		controllerPodName = podNames[0]
		g.Expect(controllerPodName).Should(ContainSubstring("controller-manager"))

		// Validate pod status
		status, err := kbc.Kubectl.Get(
			true,
			"pods", controllerPodName, "-o", "jsonpath={.status.phase}")
		g.Expect(err).NotTo(HaveOccurred())
		if status != "Running" {
			return fmt.Errorf("controller pod in %s status", status)
		}
		return nil
	}
	defer func() {
		out, err := kbc.Kubectl.CommandInNamespace("describe", "all")
		Expect(err).NotTo(HaveOccurred())
		_, _ = fmt.Fprintln(GinkgoWriter, out)
	}()
	Eventually(verifyControllerUp, 5*time.Minute, time.Second).Should(Succeed())
	return controllerPodName
}

// getMetricsOutput return the metrics output from curl pod
func getMetricsOutput(controllerPodName string, kbc *utils.TestContext) string {
	_, err := kbc.Kubectl.Command(
		"get", "clusterrolebinding", fmt.Sprintf("metrics-%s", kbc.TestSuffix),
	)
	if err != nil && strings.Contains(err.Error(), "NotFound") {
		// Create the clusterrolebinding only if it doesn't exist
		_, err = kbc.Kubectl.Command(
			"create", "clusterrolebinding", fmt.Sprintf("metrics-%s", kbc.TestSuffix),
			fmt.Sprintf("--clusterrole=e2e-%s-metrics-reader", kbc.TestSuffix),
			fmt.Sprintf("--serviceaccount=%s:%s", kbc.Kubectl.Namespace, kbc.Kubectl.ServiceAccount),
		)
		Expect(err).NotTo(HaveOccurred())
	} else {
		Expect(err).NotTo(HaveOccurred(), "Failed to check clusterrolebinding existence")
	}

	token, err := serviceAccountToken(kbc)
	Expect(err).NotTo(HaveOccurred())
	Expect(token).NotTo(BeEmpty())

	var metricsOutput string
	By("validating that the controller-manager service is available")
	_, err = kbc.Kubectl.Get(
		true,
		"service", fmt.Sprintf("e2e-%s-controller-manager-metrics-service", kbc.TestSuffix),
	)
	Expect(err).NotTo(HaveOccurred(), "Controller-manager service should exist")

	By("ensuring the service endpoint is ready")
	metricsServiceName := fmt.Sprintf("e2e-%s-controller-manager-metrics-service", kbc.TestSuffix)
	checkServiceEndpoint := func(g Gomega) {
		var output string
		output, err = kbc.Kubectl.Command(
			"get", "endpointslices.discovery.k8s.io",
			"-n", kbc.Kubectl.Namespace,
			"-l", fmt.Sprintf("kubernetes.io/service-name=%s", metricsServiceName),
			"-o", "jsonpath={range .items[*]}{range .endpoints[*]}{.addresses[*]}{end}{end}",
		)
		g.Expect(err).NotTo(HaveOccurred(), "endpointslices should exist")
		g.Expect(output).ShouldNot(BeEmpty(), "no endpoints found")
	}
	Eventually(checkServiceEndpoint, 2*time.Minute, time.Second).Should(Succeed(),
		"Service endpoint should be ready")

	// NOTE: On Kubernetes 1.33+, we've observed a delay before the metrics endpoint becomes available
	// when using controller-runtime's WithAuthenticationAndAuthorization() with self-signed certificates.
	// This delay appears to stem from Kubernetes itself, potentially due to changes in how it initializes
	// service account tokens or handles TLS/service readiness.
	By("ensuring the controller pod is fully ready before creating test pods")
	verifyControllerPodReady := func(g Gomega) {
		var output string
		output, err = kbc.Kubectl.Get(
			true,
			"pod", controllerPodName,
			"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}",
		)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(Equal("True"), "Controller pod not ready")
	}
	Eventually(verifyControllerPodReady, 3*time.Minute, time.Second).Should(Succeed())

	webhookServiceName := fmt.Sprintf("e2e-%s-webhook-service", kbc.TestSuffix)
	if _, err = kbc.Kubectl.Get(false, "service", webhookServiceName); err == nil {
		By("waiting for the webhook service endpoints to be ready")
		checkWebhookEndpoint := func(g Gomega) {
			var output string
			output, err = kbc.Kubectl.Command(
				"get", "endpointslices.discovery.k8s.io",
				"-n", kbc.Kubectl.Namespace,
				"-l", fmt.Sprintf("kubernetes.io/service-name=%s", webhookServiceName),
				"-o", "jsonpath={range .items[*]}{range .endpoints[*]}{.addresses[*]}{end}{end}",
			)
			g.Expect(err).NotTo(HaveOccurred(), "webhook endpoints should exist")
			g.Expect(output).ShouldNot(BeEmpty(), "webhook endpoints not yet ready")
		}
		Eventually(checkWebhookEndpoint, 3*time.Minute, time.Second).Should(Succeed(),
			"Webhook service endpoints should be ready")
	}

	By("creating a curl pod to access the metrics endpoint")
	cmdOpts := cmdOptsToCreateCurlPod(kbc, token)
	_, err = kbc.Kubectl.CommandInNamespace(cmdOpts...)
	Expect(err).NotTo(HaveOccurred())

	By("validating that the curl pod is running as expected")
	verifyCurlUp := func(g Gomega) {
		var status string
		status, err = kbc.Kubectl.Get(
			true,
			"pods", "curl", "-o", "jsonpath={.status.phase}")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(status).To(Equal("Succeeded"), fmt.Sprintf("curl pod in %s status", status))
	}
	Eventually(verifyCurlUp, 240*time.Second, time.Second).Should(Succeed())

	By("validating that the correct ServiceAccount is being used")
	saName := kbc.Kubectl.ServiceAccount
	currentSAOutput, err := kbc.Kubectl.Get(
		true,
		"serviceaccount", saName,
		"-o", "jsonpath={.metadata.name}",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to fetch the service account")
	Expect(currentSAOutput).To(Equal(saName), "The ServiceAccount in use does not match the expected one")

	By("validating that the metrics endpoint is serving as expected")
	getCurlLogs := func(g Gomega) {
		metricsOutput, err = kbc.Kubectl.Logs("curl")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(metricsOutput).Should(ContainSubstring("< HTTP/1.1 200 OK"))
	}
	Eventually(getCurlLogs, 10*time.Second, time.Second).Should(Succeed())
	removeCurlPod(kbc)
	return metricsOutput
}

func metricsShouldBeUnavailable(kbc *utils.TestContext) {
	_, err := kbc.Kubectl.Command(
		"create", "clusterrolebinding", fmt.Sprintf("metrics-%s", kbc.TestSuffix),
		fmt.Sprintf("--clusterrole=e2e-%s-metrics-reader", kbc.TestSuffix),
		fmt.Sprintf("--serviceaccount=%s:%s", kbc.Kubectl.Namespace, kbc.Kubectl.ServiceAccount))
	Expect(err).NotTo(HaveOccurred())

	token, err := serviceAccountToken(kbc)
	Expect(err).NotTo(HaveOccurred())
	Expect(token).NotTo(BeEmpty())

	By("creating a curl pod to access the metrics endpoint")
	cmdOpts := cmdOptsToCreateCurlPod(kbc, token)
	_, err = kbc.Kubectl.CommandInNamespace(cmdOpts...)
	Expect(err).NotTo(HaveOccurred())

	By("validating that the curl pod fail as expected")
	verifyCurlUp := func(g Gomega) {
		status, errCurl := kbc.Kubectl.Get(
			true,
			"pods", "curl", "-o", "jsonpath={.status.phase}")
		g.Expect(errCurl).NotTo(HaveOccurred())
		g.Expect(status).NotTo(Equal("Failed"),
			fmt.Sprintf("curl pod in %s status when should fail with an error", status))
	}
	Eventually(verifyCurlUp, 240*time.Second, time.Second).Should(Succeed())

	By("validating that the metrics endpoint is not working as expected")
	getCurlLogs := func(g Gomega) {
		metricsOutput, err := kbc.Kubectl.Logs("curl")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(metricsOutput).Should(ContainSubstring("Could not resolve host"))
	}
	Eventually(getCurlLogs, 10*time.Second, time.Second).Should(Succeed())
	removeCurlPod(kbc)
}

func cmdOptsToCreateCurlPod(kbc *utils.TestContext, token string) []string {
	//nolint:lll
	cmdOpts := []string{
		"run", "curl",
		"--restart=Never",
		"--namespace", kbc.Kubectl.Namespace,
		"--image=curlimages/curl:latest",
		"--overrides",
		fmt.Sprintf(`{
			"spec": {
				"containers": [{
					"name": "curl",
					"image": "curlimages/curl:latest",
					"command": ["/bin/sh", "-c"],
					"args": ["curl -v -k -H 'Authorization: Bearer %s' https://e2e-%s-controller-manager-metrics-service.%s.svc.cluster.local:8443/metrics"],
					"securityContext": {
						"readOnlyRootFilesystem": true,
						"allowPrivilegeEscalation": false,
						"capabilities": {
							"drop": ["ALL"]
						},
						"runAsNonRoot": true,
						"runAsUser": 1000,
						"seccompProfile": {
							"type": "RuntimeDefault"
						}
					}
				}],
				"serviceAccountName": "%s"
			}
    }`, token, kbc.TestSuffix, kbc.Kubectl.Namespace, kbc.Kubectl.ServiceAccount),
	}
	return cmdOpts
}

func removeCurlPod(kbc *utils.TestContext) {
	By("cleaning up the curl pod")
	_, err := kbc.Kubectl.Delete(true, "pods/curl", "--grace-period=0", "--force")
	Expect(err).NotTo(HaveOccurred())
}

// serviceAccountToken provides a helper function that can provide you with a service account
// token that you can use to interact with the service. This function leverages the k8s'
// TokenRequest API in raw format in order to make it generic for all version of the k8s that
// is currently being supported in kubebuilder test infra.
// TokenRequest API returns the token in raw JWT format itself. There is no conversion required.
func serviceAccountToken(kbc *utils.TestContext) (string, error) {
	var out string

	secretName := fmt.Sprintf("%s-token-request", kbc.Kubectl.ServiceAccount)
	tokenRequestFile := filepath.Join(kbc.Dir, secretName)
	if err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o755)); err != nil {
		return out, fmt.Errorf("error creating token request file %s: %w", tokenRequestFile, err)
	}
	getToken := func(g Gomega) {
		// Output of this is already a valid JWT token. No need to covert this from base64 to string format
		rawJSON, err := kbc.Kubectl.Command(
			"create",
			"--raw", fmt.Sprintf(
				"/api/v1/namespaces/%s/serviceaccounts/%s/token",
				kbc.Kubectl.Namespace,
				kbc.Kubectl.ServiceAccount,
			),
			"-f", tokenRequestFile,
		)

		g.Expect(err).NotTo(HaveOccurred())
		var token tokenRequest
		err = json.Unmarshal([]byte(rawJSON), &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(getToken, time.Minute, time.Second).Should(Succeed())

	return out, nil
}
