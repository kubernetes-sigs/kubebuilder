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
			By("By removing restricted namespace label")
			_ = kbc.RemoveNamespaceLabelToWarnAboutRestricted()

			By("clean up API objects created during the test")
			_ = kbc.Make("undeploy")

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
		It("should generate a runnable project using webhooks and installed with the HelmChart", func() {
			GenerateV4(kbc)
			By("installing Helm")
			Expect(kbc.InstallHelm()).To(Succeed())

			Run(kbc, true, false, true, true, false)

			By("uninstalling Helm Release")
			Expect(kbc.UninstallHelmRelease()).To(Succeed())
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
	})
})

// Run runs a set of e2e tests for a scaffolded project defined by a TestContext.
func Run(kbc *utils.TestContext, hasWebhook, isToUseInstaller, isToUseHelmChart, hasMetrics bool,
	hasNetworkPolicies bool) {
	var controllerPodName string
	var err error

	By("creating manager namespace")
	err = kbc.CreateManagerNamespace()
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("labeling all namespaces to warn about restricted")
	err = kbc.LabelNamespacesToWarnAboutRestricted()
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("updating the go.mod")
	err = kbc.Tidy()
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("run make all")
	err = kbc.Make("all")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("building the controller image")
	err = kbc.Make("docker-build", "IMG="+kbc.ImageName)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("loading the controller docker image into the kind cluster")
	err = kbc.LoadImageToKindCluster()
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	if !isToUseInstaller && !isToUseHelmChart {
		By("deploying the controller-manager")
		cmd := exec.Command("make", "deploy", "IMG="+kbc.ImageName)
		_, err = kbc.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
	}

	if isToUseInstaller && !isToUseHelmChart {
		By("building the installer")
		err = kbc.Make("build-installer", "IMG="+kbc.ImageName)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())

		By("deploying the controller-manager with the installer")
		_, err = kbc.Kubectl.Apply(true, "-f", "dist/install.yaml")
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
	}

	if isToUseHelmChart && !isToUseInstaller {
		By("building the helm-chart")
		err = kbc.EditHelmPlugin()
		Expect(err).NotTo(HaveOccurred(), "Failed to edit project to generate helm-chart")

		By("updating values with image name")
		values := filepath.Join(kbc.Dir, "dist", "chart", "values.yaml")
		err = util.ReplaceInFile(values, "repository: controller", "repository: e2e-test/controller-manager")
		Expect(err).NotTo(HaveOccurred(), "Failed to edit repository in the chart/values.yaml")
		err = util.ReplaceInFile(values, "tag: latest", fmt.Sprintf("tag: %s", kbc.TestSuffix))
		Expect(err).NotTo(HaveOccurred(), "Failed to edit tag in the chart/values.yaml")

		By("updating values to enable prometheus")
		err = util.ReplaceInFile(values, "prometheus:\n  enable: false", "prometheus:\n  enable: true")
		Expect(err).NotTo(HaveOccurred(), "Failed to enable prometheus in the chart/values.yaml")

		By("updating values to set crd.keep false")
		err = util.ReplaceInFile(values, "keep: true", "keep: false")
		Expect(err).NotTo(HaveOccurred(), "Failed to set keep false in the chart/values.yaml")

		By("install with Helm release")
		err = kbc.HelmInstallRelease()
		Expect(err).NotTo(HaveOccurred(), "Failed to install helm release")
	}

	By("Checking controllerManager and getting the name of the Pod")
	controllerPodName = getControllerName(kbc)

	By("Checking if all flags are applied to the manager pod")
	podOutput, err := kbc.Kubectl.Get(
		true,
		"pod", controllerPodName,
		"-o", "jsonpath={.spec.containers[0].args}",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, podOutput).To(ContainSubstring("leader-elect"),
		"Expected manager pod to have --leader-elect flag")
	ExpectWithOffset(1, podOutput).To(ContainSubstring("health-probe-bind-address"),
		"Expected manager pod to have --health-probe-bind-address flag")

	By("validating that the Prometheus manager has provisioned the Service")
	EventuallyWithOffset(1, func() error {
		_, err := kbc.Kubectl.Get(
			false,
			"Service", "prometheus-operator")
		return err
	}, time.Minute, time.Second).Should(Succeed())

	By("validating that the ServiceMonitor for Prometheus is applied in the namespace")
	_, err = kbc.Kubectl.Get(
		true,
		"ServiceMonitor")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	if hasNetworkPolicies {
		By("Checking for Calico pods")
		outputGet, err := kbc.Kubectl.Get(
			false,
			"pods",
			"-n", "kube-system",
			"-l", "k8s-app=calico-node",
			"-o", "jsonpath={.items[*].status.phase}",
		)
		Expect(err).NotTo(HaveOccurred(), "Failed to get Calico pods")
		Expect(outputGet).To(ContainSubstring("Running"), "All Calico pods should be in Running state")

		if hasMetrics {
			By("labeling the namespace to allow consume the metrics")
			_, err = kbc.Kubectl.Command("label", "namespaces", kbc.Kubectl.Namespace,
				"metrics=enabled")
			ExpectWithOffset(2, err).NotTo(HaveOccurred())

			By("Ensuring the Allow Metrics Traffic NetworkPolicy exists", func() {
				output, err := kbc.Kubectl.Get(
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
			ExpectWithOffset(2, err).NotTo(HaveOccurred())

			By("Ensuring the allow-webhook-traffic NetworkPolicy exists", func() {
				output, err := kbc.Kubectl.Get(
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
		EventuallyWithOffset(1, func() error {
			_, err := kbc.Kubectl.Get(
				true,
				"secrets", "webhook-server-cert")
			return err
		}, time.Minute, time.Second).Should(Succeed())

		By("validating that the mutating|validating webhooks have the CA injected")
		verifyCAInjection := func() error {
			mwhOutput, err := kbc.Kubectl.Get(
				false,
				"mutatingwebhookconfigurations.admissionregistration.k8s.io",
				fmt.Sprintf("e2e-%s-mutating-webhook-configuration", kbc.TestSuffix),
				"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
			ExpectWithOffset(2, err).NotTo(HaveOccurred())
			// check that ca should be long enough, because there may be a place holder "\n"
			ExpectWithOffset(2, len(mwhOutput)).To(BeNumerically(">", 10))

			vwhOutput, err := kbc.Kubectl.Get(
				false,
				"validatingwebhookconfigurations.admissionregistration.k8s.io",
				fmt.Sprintf("e2e-%s-validating-webhook-configuration", kbc.TestSuffix),
				"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
			ExpectWithOffset(2, err).NotTo(HaveOccurred())
			// check that ca should be long enough, because there may be a place holder "\n"
			ExpectWithOffset(2, len(vwhOutput)).To(BeNumerically(">", 10))

			return nil
		}
		EventuallyWithOffset(1, verifyCAInjection, time.Minute, time.Second).Should(Succeed())

		By("validating that the CA injection is applied for CRD conversion")
		crdKind := "ConversionTest"
		verifyCAInjection = func() error {
			crdOutput, err := kbc.Kubectl.Get(
				false,
				"customresourcedefinition.apiextensions.k8s.io",
				"-o", fmt.Sprintf(
					"jsonpath={.items[?(@.spec.names.kind=='%s')].spec.conversion.webhook.clientConfig.caBundle}",
					crdKind),
			)
			ExpectWithOffset(1, err).NotTo(HaveOccurred(),
				"failed to get CRD conversion webhook configuration")

			// Check if the CA bundle is populated (length > 10 to avoid placeholder values)
			ExpectWithOffset(1, len(crdOutput)).To(BeNumerically(">", 10),
				"CA bundle should be injected into the CRD")
			return nil
		}
		EventuallyWithOffset(1, verifyCAInjection, time.Minute, time.Second).Should(Succeed(),
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

	EventuallyWithOffset(1, func() error {
		_, err = kbc.Kubectl.Apply(true, "-f", sampleFile)
		return err
	}, time.Minute, time.Second).Should(Succeed())

	if hasMetrics {
		By("checking the metrics values to validate that the created resource object gets reconciled")
		metricsOutput := getMetricsOutput(kbc)
		ExpectWithOffset(1, metricsOutput).To(ContainSubstring(fmt.Sprintf(
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
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		count, err := strconv.Atoi(cnt)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, count).To(BeNumerically("==", 5))
	}

	if hasWebhook {
		By("creating a namespace")
		namespace := "test-webhooks"
		_, err := kbc.Kubectl.Command("create", "namespace", namespace)
		Expect(err).NotTo(HaveOccurred(), "namespace should be created successfully")

		By("applying the CR in the created namespace")
		EventuallyWithOffset(1, func() error {
			_, err := kbc.Kubectl.Apply(false, "-n", namespace, "-f", sampleFile)
			return err
		}, 2*time.Minute, time.Second).ShouldNot(HaveOccurred(),
			"apply in test-webhooks ns should not fail")

		By("validating that mutating webhooks are working fine outside of the manager's namespace")
		cnt, err := kbc.Kubectl.Get(
			false,
			"-n", namespace,
			"-f", sampleFile,
			"-o", "go-template={{ .spec.count }}")
		ExpectWithOffset(1, err).NotTo(HaveOccurred())

		count, err := strconv.Atoi(cnt)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		ExpectWithOffset(1, count).To(BeNumerically("==", 5),
			"the mutating webhook should set the count to 5")

		By("removing the namespace")
		_, err = kbc.Kubectl.Command("delete", "namespace", namespace)
		Expect(err).NotTo(HaveOccurred(), "namespace should be removed successfully")

		By("validating the conversion")

		// Update the ConversionTest CR sample in v1 to set a specific `size`
		By("modifying the ConversionTest CR sample to set `size` for conversion testing")
		conversionCRFile := filepath.Join("config", "samples",
			fmt.Sprintf("%s_v1_conversiontest.yaml", kbc.Group))
		conversionCRPath, err := filepath.Abs(filepath.Join(fmt.Sprintf("e2e-%s", kbc.TestSuffix), conversionCRFile))
		Expect(err).To(Not(HaveOccurred()))

		// Edit the file to include `size` in the spec field for v1
		f, err := os.OpenFile(conversionCRPath, os.O_APPEND|os.O_WRONLY, 0o644)
		Expect(err).To(Not(HaveOccurred()))
		defer func() {
			err = f.Close()
			Expect(err).To(Not(HaveOccurred()))
		}()
		_, err = f.WriteString("\nspec:\n  size: 3")
		Expect(err).To(Not(HaveOccurred()))

		// Apply the ConversionTest Custom Resource in v1
		By("applying the modified ConversionTest CR in v1 for conversion")
		_, err = kbc.Kubectl.Apply(true, "-f", conversionCRPath)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to apply modified ConversionTest CR")

		// TODO: Add validation to check the conversion
		// the v2 should have spec.replicas == 3

		if hasMetrics {
			By("validating conversion metrics to confirm conversion operations")
			metricsOutput := getMetricsOutput(kbc)
			conversionMetric := `controller_runtime_reconcile_total{controller="conversiontest",result="success"} 1`
			ExpectWithOffset(1, metricsOutput).To(ContainSubstring(conversionMetric),
				"Expected metric for successful ConversionTest reconciliation")
		}
	}
}

func getControllerName(kbc *utils.TestContext) string {
	By("validating that the controller-manager pod is running as expected")
	var controllerPodName string
	verifyControllerUp := func() error {
		// Get pod name
		podOutput, err := kbc.Kubectl.Get(
			true,
			"pods", "-l", "control-plane=controller-manager",
			"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}"+
				"{{ \"\\n\" }}{{ end }}{{ end }}")
		ExpectWithOffset(2, err).NotTo(HaveOccurred())
		podNames := util.GetNonEmptyLines(podOutput)
		if len(podNames) != 1 {
			return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
		}
		controllerPodName = podNames[0]
		ExpectWithOffset(2, controllerPodName).Should(ContainSubstring("controller-manager"))

		// Validate pod status
		status, err := kbc.Kubectl.Get(
			true,
			"pods", controllerPodName, "-o", "jsonpath={.status.phase}")
		ExpectWithOffset(2, err).NotTo(HaveOccurred())
		if status != "Running" {
			return fmt.Errorf("controller pod in %s status", status)
		}
		return nil
	}
	defer func() {
		out, err := kbc.Kubectl.CommandInNamespace("describe", "all")
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		_, _ = fmt.Fprintln(GinkgoWriter, out)
	}()
	EventuallyWithOffset(1, verifyControllerUp, 5*time.Minute, time.Second).Should(Succeed())
	return controllerPodName
}

// getMetricsOutput return the metrics output from curl pod
func getMetricsOutput(kbc *utils.TestContext) string {
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
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
	} else {
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to check clusterrolebinding existence")
	}

	token, err := serviceAccountToken(kbc)
	ExpectWithOffset(2, err).NotTo(HaveOccurred())
	ExpectWithOffset(2, token).NotTo(BeEmpty())

	var metricsOutput string
	By("validating that the controller-manager service is available")
	_, err = kbc.Kubectl.Get(
		true,
		"service", fmt.Sprintf("e2e-%s-controller-manager-metrics-service", kbc.TestSuffix),
	)
	ExpectWithOffset(2, err).NotTo(HaveOccurred(), "Controller-manager service should exist")

	By("ensuring the service endpoint is ready")
	eventuallyCheckServiceEndpoint := func() error {
		output, err := kbc.Kubectl.Get(
			true,
			"endpoints", fmt.Sprintf("e2e-%s-controller-manager-metrics-service", kbc.TestSuffix),
			"-o", "jsonpath={.subsets[*].addresses[*].ip}",
		)
		if err != nil {
			return err
		}
		if output == "" {
			return fmt.Errorf("no endpoints found")
		}
		return nil
	}
	EventuallyWithOffset(2, eventuallyCheckServiceEndpoint, 2*time.Minute, time.Second).Should(Succeed(),
		"Service endpoint should be ready")

	By("creating a curl pod to access the metrics endpoint")
	cmdOpts := cmdOptsToCreateCurlPod(kbc, token)
	_, err = kbc.Kubectl.CommandInNamespace(cmdOpts...)
	ExpectWithOffset(2, err).NotTo(HaveOccurred())

	By("validating that the curl pod is running as expected")
	verifyCurlUp := func() error {
		status, err := kbc.Kubectl.Get(
			true,
			"pods", "curl", "-o", "jsonpath={.status.phase}")
		ExpectWithOffset(3, err).NotTo(HaveOccurred())
		if status != "Succeeded" {
			return fmt.Errorf("curl pod in %s status", status)
		}
		return nil
	}
	EventuallyWithOffset(2, verifyCurlUp, 240*time.Second, time.Second).Should(Succeed())

	By("validating that the metrics endpoint is serving as expected")
	getCurlLogs := func() string {
		metricsOutput, err = kbc.Kubectl.Logs("curl")
		ExpectWithOffset(3, err).NotTo(HaveOccurred())
		return metricsOutput
	}
	EventuallyWithOffset(2, getCurlLogs, 10*time.Second, time.Second).Should(ContainSubstring("< HTTP/1.1 200 OK"))
	removeCurlPod(kbc)
	return metricsOutput
}

func metricsShouldBeUnavailable(kbc *utils.TestContext) {
	_, err := kbc.Kubectl.Command(
		"create", "clusterrolebinding", fmt.Sprintf("metrics-%s", kbc.TestSuffix),
		fmt.Sprintf("--clusterrole=e2e-%s-metrics-reader", kbc.TestSuffix),
		fmt.Sprintf("--serviceaccount=%s:%s", kbc.Kubectl.Namespace, kbc.Kubectl.ServiceAccount))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	token, err := serviceAccountToken(kbc)
	ExpectWithOffset(2, err).NotTo(HaveOccurred())
	ExpectWithOffset(2, token).NotTo(BeEmpty())

	By("creating a curl pod to access the metrics endpoint")
	cmdOpts := cmdOptsToCreateCurlPod(kbc, token)
	_, err = kbc.Kubectl.CommandInNamespace(cmdOpts...)
	ExpectWithOffset(2, err).NotTo(HaveOccurred())

	By("validating that the curl pod fail as expected")
	verifyCurlUp := func() error {
		status, err := kbc.Kubectl.Get(
			true,
			"pods", "curl", "-o", "jsonpath={.status.phase}")
		ExpectWithOffset(3, err).NotTo(HaveOccurred())
		if status != "Failed" {
			return fmt.Errorf(
				"curl pod in %s status when should fail with an error", status)
		}
		return nil
	}
	EventuallyWithOffset(2, verifyCurlUp, 240*time.Second, time.Second).Should(Succeed())

	By("validating that the metrics endpoint is not working as expected")
	getCurlLogs := func() string {
		metricsOutput, err := kbc.Kubectl.Logs("curl")
		ExpectWithOffset(3, err).NotTo(HaveOccurred())
		return metricsOutput
	}
	EventuallyWithOffset(2, getCurlLogs, 10*time.Second, time.Second).Should(ContainSubstring("Could not resolve host"))
	removeCurlPod(kbc)
}

func cmdOptsToCreateCurlPod(kbc *utils.TestContext, token string) []string {
	// nolint:lll
	cmdOpts := []string{
		"run", "curl",
		"--restart=Never",
		"--namespace", kbc.Kubectl.Namespace,
		"--image=curlimages/curl:7.78.0",
		"--",
		"/bin/sh", "-c", fmt.Sprintf("curl -v -k -H 'Authorization: Bearer %s' https://e2e-%s-controller-manager-metrics-service.%s.svc.cluster.local:8443/metrics",
			token, kbc.TestSuffix, kbc.Kubectl.Namespace),
	}
	return cmdOpts
}

func removeCurlPod(kbc *utils.TestContext) {
	By("cleaning up the curl pod")
	_, err := kbc.Kubectl.Delete(true, "pods/curl")
	ExpectWithOffset(3, err).NotTo(HaveOccurred())
}

// serviceAccountToken provides a helper function that can provide you with a service account
// token that you can use to interact with the service. This function leverages the k8s'
// TokenRequest API in raw format in order to make it generic for all version of the k8s that
// is currently being supported in kubebuilder test infra.
// TokenRequest API returns the token in raw JWT format itself. There is no conversion required.
func serviceAccountToken(kbc *utils.TestContext) (out string, err error) {
	secretName := fmt.Sprintf("%s-token-request", kbc.Kubectl.ServiceAccount)
	tokenRequestFile := filepath.Join(kbc.Dir, secretName)
	err = os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o755))
	if err != nil {
		return out, err
	}
	var rawJson string
	Eventually(func() error {
		// Output of this is already a valid JWT token. No need to covert this from base64 to string format
		rawJson, err = kbc.Kubectl.Command(
			"create",
			"--raw", fmt.Sprintf(
				"/api/v1/namespaces/%s/serviceaccounts/%s/token",
				kbc.Kubectl.Namespace,
				kbc.Kubectl.ServiceAccount,
			),
			"-f", tokenRequestFile,
		)
		if err != nil {
			return err
		}
		var token tokenRequest
		err = json.Unmarshal([]byte(rawJson), &token)
		if err != nil {
			return err
		}
		out = token.Status.Token
		return nil
	}, time.Minute, time.Second).Should(Succeed())

	return out, err
}
