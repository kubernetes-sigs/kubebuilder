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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"

	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"

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
			By("clean up API objects created during the test")
			kbc.CleanupManifests(filepath.Join("config", "default"))

			By("removing controller image and working dir")
			kbc.Destroy()
		})
		It("should generate a runnable project", func() {
			kbc.IsRestricted = false
			GenerateV4(kbc)
			Run(kbc, true, false, true)
		})
		It("should generate a runnable project with the Installer", func() {
			kbc.IsRestricted = false
			GenerateV4(kbc)
			Run(kbc, false, true, true)
		})
		It("should generate a runnable project without metrics exposed", func() {
			kbc.IsRestricted = false
			GenerateV4WithoutMetrics(kbc)
			Run(kbc, true, false, false)
		})
		It("should generate a runnable project with the manager running "+
			"as restricted and without webhooks", func() {
			kbc.IsRestricted = true
			GenerateV4WithoutWebhooks(kbc)
			Run(kbc, false, false, true)
		})
	})
})

// Run runs a set of e2e tests for a scaffolded project defined by a TestContext.
func Run(kbc *utils.TestContext, hasWebhook, isToUseInstaller, hasMetrics bool) {
	var controllerPodName string
	var err error

	By("creating manager namespace")
	err = kbc.CreateManagerNamespace()
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("labeling all namespaces to warn about restricted")
	err = kbc.LabelAllNamespacesToWarnAboutRestricted()
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("updating the go.mod")
	err = kbc.Tidy()
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("run make manifests")
	err = kbc.Make("manifests")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("run make generate")
	err = kbc.Make("generate")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("building the controller image")
	err = kbc.Make("docker-build", "IMG="+kbc.ImageName)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("loading the controller docker image into the kind cluster")
	err = kbc.LoadImageToKindCluster()
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	var output []byte
	if !isToUseInstaller {
		By("deploying the controller-manager")
		cmd := exec.Command("make", "deploy", "IMG="+kbc.ImageName)
		output, err = kbc.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
	} else {
		By("building the installer")
		err = kbc.Make("build-installer", "IMG="+kbc.ImageName)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())

		By("deploying the controller-manager with the installer")
		_, err = kbc.Kubectl.Apply(true, "-f", "dist/install.yaml")
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
	}

	if kbc.IsRestricted {
		By("validating that manager Pod/container(s) are restricted")
		ExpectWithOffset(1, output).NotTo(ContainSubstring("Warning: would violate PodSecurity"))
	}

	By("validating that the controller-manager pod is running as expected")
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
		fmt.Fprintln(GinkgoWriter, out)
	}()
	EventuallyWithOffset(1, verifyControllerUp, time.Minute, time.Second).Should(Succeed())

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

	By("validating the metrics endpoint")
	_ = curlMetrics(kbc, hasMetrics)

	if hasWebhook {
		By("validating that cert-manager has provisioned the certificate Secret")
		EventuallyWithOffset(1, func() error {
			_, err := kbc.Kubectl.Get(
				true,
				"secrets", "webhook-server-cert")
			return err
		}, time.Minute, time.Second).Should(Succeed())
	}

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

	if hasWebhook {
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
		metricsOutput := curlMetrics(kbc, hasMetrics)
		ExpectWithOffset(1, metricsOutput).To(ContainSubstring(fmt.Sprintf(
			`controller_runtime_reconcile_total{controller="%s",result="success"} 1`,
			strings.ToLower(kbc.Kind),
		)))
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

}

// curlMetrics curl's the /metrics endpoint, returning all logs once a 200 status is returned.
func curlMetrics(kbc *utils.TestContext, hasMetrics bool) string {
	By("validating that the controller-manager service is available")
	_, err := kbc.Kubectl.Get(
		true,
		"service", fmt.Sprintf("e2e-%s-controller-manager-metrics-service", kbc.TestSuffix),
	)
	ExpectWithOffset(2, err).NotTo(HaveOccurred(), "Controller-manager service should exist")

	By("validating that the controller-manager deployment is ready")
	verifyDeploymentReady := func() error {
		output, err := kbc.Kubectl.Get(
			true,
			"deployment", fmt.Sprintf("e2e-%s-controller-manager", kbc.TestSuffix),
			"-o", "jsonpath={.status.readyReplicas}",
		)
		if err != nil {
			return err
		}
		readyReplicas, _ := strconv.Atoi(output)
		if readyReplicas < 1 {
			return fmt.Errorf("expected at least 1 ready replica, got %d", readyReplicas)
		}
		return nil
	}
	EventuallyWithOffset(2, verifyDeploymentReady, 240*time.Second, time.Second).Should(Succeed(),
		"Deployment is not ready")

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
	// nolint:lll
	cmdOpts := []string{
		"run", "curl",
		"--restart=Never",
		"--namespace", kbc.Kubectl.Namespace,
		"--image=curlimages/curl:7.78.0",
		"--",
		"/bin/sh", "-c", fmt.Sprintf("curl -v -k http://e2e-%s-controller-manager-metrics-service.%s.svc.cluster.local:8080/metrics",
			kbc.TestSuffix, kbc.Kubectl.Namespace),
	}
	_, err = kbc.Kubectl.CommandInNamespace(cmdOpts...)
	ExpectWithOffset(2, err).NotTo(HaveOccurred())

	var metricsOutput string
	if hasMetrics {
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
	} else {
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
			metricsOutput, err = kbc.Kubectl.Logs("curl")
			ExpectWithOffset(3, err).NotTo(HaveOccurred())
			return metricsOutput
		}
		EventuallyWithOffset(2, getCurlLogs, 10*time.Second, time.Second).Should(ContainSubstring("Connection refused"))
	}
	By("cleaning up the curl pod")
	_, err = kbc.Kubectl.Delete(true, "pods/curl")
	ExpectWithOffset(3, err).NotTo(HaveOccurred())

	return metricsOutput
}
