/*
Copyright 2024 The Kubernetes Authors.

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

package helpers

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

const (
	// defaultTimeout is the default timeout for Eventually checks
	defaultTimeout = 3 * time.Minute
	// defaultPollingInterval is the default polling interval for Eventually checks
	defaultPollingInterval = time.Second
)

// InstallMethod defines how the project will be deployed
type InstallMethod string

const (
	// InstallMethodKustomize uses `make deploy` (default)
	InstallMethodKustomize InstallMethod = "kustomize"
	// InstallMethodInstaller uses `build-installer` and applies dist/install.yaml
	InstallMethodInstaller InstallMethod = "installer"
	// InstallMethodHelm uses Helm chart installation
	InstallMethodHelm InstallMethod = "helm"
)

// RunOptions configures the Run test execution
type RunOptions struct {
	// HasWebhook indicates if webhooks are enabled
	HasWebhook bool
	// HasMetrics indicates if metrics are enabled
	HasMetrics bool
	// HasNetworkPolicies indicates if network policies are enabled
	HasNetworkPolicies bool
	// InstallMethod specifies how to install the project
	InstallMethod InstallMethod
	// HelmFullnameOverride sets fullnameOverride for Helm installations (only for InstallMethodHelm)
	HelmFullnameOverride string
	// SkipChartGeneration skips build-installer and chart generation (chart already prepared externally)
	SkipChartGeneration bool
}

// Run executes common e2e tests for a scaffolded project.
// This function is shared between go/v4 and helm/v2-alpha plugin tests.
func Run(kbc *utils.TestContext, opts RunOptions) {
	var controllerPodName string
	var err error

	// Determine the name prefix for resources
	// If fullnameOverride is set, use that; otherwise use e2e-{suffix}
	namePrefix := fmt.Sprintf("e2e-%s", kbc.TestSuffix)
	if opts.HelmFullnameOverride != "" {
		namePrefix = opts.HelmFullnameOverride
	}

	// For Helm installations with fullnameOverride, update ServiceAccount name
	if opts.InstallMethod == InstallMethodHelm {
		kbc.Kubectl.ServiceAccount = namePrefix + "-controller-manager"
	}

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

	// Deploy based on installation method
	switch opts.InstallMethod {
	case InstallMethodKustomize:
		By("deploying the controller-manager via make deploy")
		cmd := exec.Command("make", "deploy", "IMG="+kbc.ImageName)
		_, err = kbc.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

	case InstallMethodInstaller:
		By("building the installer")
		err = kbc.Make("build-installer", "IMG="+kbc.ImageName)
		Expect(err).NotTo(HaveOccurred())

		By("deploying the controller-manager with the installer")
		_, err = kbc.Kubectl.Apply(true, "-f", "dist/install.yaml")
		Expect(err).NotTo(HaveOccurred())

	case InstallMethodHelm:
		if !opts.SkipChartGeneration {
			By("building the installer manifest for helm chart generation")
			err = kbc.Make("build-installer", "IMG="+kbc.ImageName)
			Expect(err).NotTo(HaveOccurred(), "Failed to build installer manifest")

			By("building the helm-chart")
			err = kbc.EditHelmPlugin()
			Expect(err).NotTo(HaveOccurred(), "Failed to edit helm plugin")
		}

		By("deploying the controller-manager via Helm")
		err = kbc.HelmInstallRelease()
		Expect(err).NotTo(HaveOccurred(), "Failed to install Helm release")
	}

	By("Checking controllerManager and getting the name of the Pod")
	controllerPodName = GetControllerPodName(kbc)

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

	if opts.HasNetworkPolicies {
		if opts.HasMetrics {
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

		if opts.HasWebhook {
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

	if opts.HasWebhook {
		By("validating that cert-manager has provisioned the certificate Secret")

		verifyWebhookCert := func(g Gomega) {
			var output string
			output, err = kbc.Kubectl.Get(
				true,
				"secrets", "webhook-server-cert")
			g.Expect(err).ToNot(HaveOccurred(), "webhook-server-cert should exist in the namespace")
			g.Expect(output).To(ContainSubstring("webhook-server-cert"))
		}

		Eventually(verifyWebhookCert, defaultTimeout, defaultPollingInterval).Should(Succeed())

		By("validating that the mutating|validating webhooks have the CA injected")
		verifyCAInjection := func(g Gomega) {
			// Determine the prefix for webhook configurations
			// If fullnameOverride is set, use that; otherwise use e2e-{suffix}
			var namePrefix string
			if opts.HelmFullnameOverride != "" {
				namePrefix = opts.HelmFullnameOverride
			} else {
				namePrefix = fmt.Sprintf("e2e-%s", kbc.TestSuffix)
			}

			var mwhOutput, vwhOutput string
			mwhOutput, err = kbc.Kubectl.Get(
				false,
				"mutatingwebhookconfigurations.admissionregistration.k8s.io",
				fmt.Sprintf("%s-mutating-webhook-configuration", namePrefix),
				"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
			g.Expect(err).NotTo(HaveOccurred())
			// check that ca should be long enough, because there may be a place holder "\n"
			g.Expect(len(mwhOutput)).To(BeNumerically(">", 10))

			vwhOutput, err = kbc.Kubectl.Get(
				false,
				"validatingwebhookconfigurations.admissionregistration.k8s.io",
				fmt.Sprintf("%s-validating-webhook-configuration", namePrefix),
				"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
			g.Expect(err).NotTo(HaveOccurred())
			// check that ca should be long enough, because there may be a place holder "\n"
			g.Expect(len(vwhOutput)).To(BeNumerically(">", 10))
		}

		Eventually(verifyCAInjection, defaultTimeout, defaultPollingInterval).Should(Succeed())

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
		Eventually(verifyCAInjection, defaultTimeout, defaultPollingInterval).Should(Succeed(),
			"CA injection validation failed")
	}

	By("creating an instance of the CR")
	// currently controller-runtime doesn't provide a readiness probe, we retry a few times
	// we can change it to probe the readiness endpoint after CR supports it.
	sampleFile := filepath.Join("config", "samples",
		fmt.Sprintf("%s_%s_%s.yaml", kbc.Group, kbc.Version, strings.ToLower(kbc.Kind)))
	sampleFilePath, err := filepath.Abs(filepath.Join(fmt.Sprintf("e2e-%s", kbc.TestSuffix), sampleFile))
	Expect(err).To(Not(HaveOccurred()))

	// Add a field to the sample CR for testing
	err = util.ReplaceInFile(sampleFilePath, "# TODO(user): Add fields here", "foo: bar")
	Expect(err).To(Not(HaveOccurred()))

	applySample := func(g Gomega) {
		g.Expect(kbc.Kubectl.Apply(true, "-f", sampleFile)).
			Error().NotTo(HaveOccurred())
	}
	Eventually(applySample, defaultTimeout, defaultPollingInterval).Should(Succeed())

	if opts.HasMetrics {
		By("checking the metrics values to validate that the created resource object gets reconciled")
		metricsOutput := GetMetricsOutput(controllerPodName, namePrefix, kbc)
		Expect(metricsOutput).To(ContainSubstring(fmt.Sprintf(
			`controller_runtime_reconcile_total{controller="%s",result="success"} 1`,
			strings.ToLower(kbc.Kind),
		)))
	}

	if !opts.HasMetrics {
		By("validating the metrics endpoint is not working as expected")
		ValidateMetricsUnavailable(namePrefix, kbc)
	}

	if opts.HasWebhook {
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

	if opts.HasWebhook {
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

		// Note: Webhooks are cluster-scoped and validate/mutate CRs in ALL namespaces,
		// even in namespace-scoped managers. The manager won't reconcile CRs outside
		// its WATCH_NAMESPACE, but webhooks will still enforce validation/mutation rules.
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
		}, defaultTimeout, defaultPollingInterval).Should(Succeed())

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
		}, defaultTimeout, defaultPollingInterval).Should(Succeed())

		if opts.HasMetrics {
			By("validating conversion metrics to confirm conversion operations")
			metricsOutput := GetMetricsOutput(controllerPodName, namePrefix, kbc)
			conversionMetric := `controller_runtime_reconcile_total{controller="conversiontest",result="success"} 1`
			Expect(metricsOutput).To(ContainSubstring(conversionMetric),
				"Expected metric for successful ConversionTest reconciliation")
		}
	}
}
