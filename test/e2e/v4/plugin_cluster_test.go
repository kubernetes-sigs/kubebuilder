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

	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo/v2"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v3/test/e2e/utils"
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

			By("installing the cert-manager bundle")
			Expect(kbc.InstallCertManager()).To(Succeed())

			By("installing the Prometheus operator")
			Expect(kbc.InstallPrometheusOperManager()).To(Succeed())
		})

		AfterEach(func() {
			By("clean up API objects created during the test")
			kbc.CleanupManifests(filepath.Join("config", "default"))

			By("uninstalling the Prometheus manager bundle")
			kbc.UninstallPrometheusOperManager()

			By("uninstalling the cert-manager bundle")
			kbc.UninstallCertManager()

			By("removing controller image and working dir")
			kbc.Destroy()
		})
		It("should generate a runnable project"+
			" with restricted pods", func() {
			kbc.IsRestricted = true
			GenerateV4(kbc)
			Run(kbc, true, false)
		})
		It("should generate a runnable project without webhooks"+
			" with restricted pods", func() {
			kbc.IsRestricted = true
			GenerateV4WithoutWebhooks(kbc)
			Run(kbc, false, false)
		})
		It("should generate a runnable project"+
			" with the Installer", func() {
			GenerateV4(kbc)
			Run(kbc, false, true)
		})
	})
})

// Run runs a set of e2e tests for a scaffolded project defined by a TestContext.
func Run(kbc *utils.TestContext, hasWebhook, isToUseInstaller bool) {
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
		// NOTE: If you want to run the test against a GKE cluster, you will need to grant yourself permission.
		// Otherwise, you may see "... is forbidden: attempt to grant extra privileges"
		// $ kubectl create clusterrolebinding myname-cluster-admin-binding \
		// --clusterrole=cluster-admin --user=myname@mycompany.com
		// https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control
		By("deploying the controller-manager")

		cmd := exec.Command("make", "deploy", "IMG="+kbc.ImageName)
		output, err = kbc.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
	} else {
		By("building the installer")
		err = kbc.Make("build-installer", "IMG="+kbc.ImageName)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())

		// NOTE: If you want to run the test against a GKE cluster, you will need to grant yourself permission.
		// Otherwise, you may see "... is forbidden: attempt to grant extra privileges"
		// $ kubectl create clusterrolebinding myname-cluster-admin-binding \
		// --clusterrole=cluster-admin --user=myname@mycompany.com
		// https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control
		By("deploying the controller-manager with the installer")

		_, err = kbc.Kubectl.Apply(true, "-f", "dist/install.yaml")
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
	}

	if kbc.IsRestricted && !isToUseInstaller {
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

	By("granting permissions to access the metrics")
	_, err = kbc.Kubectl.Command(
		"create", "clusterrolebinding", fmt.Sprintf("metrics-%s", kbc.TestSuffix),
		fmt.Sprintf("--clusterrole=e2e-%s-metrics-reader", kbc.TestSuffix),
		fmt.Sprintf("--serviceaccount=%s:%s", kbc.Kubectl.Namespace, kbc.Kubectl.ServiceAccount))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	_ = curlMetrics(kbc)

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

	By("applying the CRD Editor Role")
	crdEditorRole := filepath.Join("config", "rbac",
		fmt.Sprintf("%s_editor_role.yaml", strings.ToLower(kbc.Kind)))
	EventuallyWithOffset(1, func() error {
		_, err = kbc.Kubectl.Apply(true, "-f", crdEditorRole)
		return err
	}, time.Minute, time.Second).Should(Succeed())

	By("applying the CRD Viewer Role")
	crdViewerRole := filepath.Join("config", "rbac", fmt.Sprintf("%s_viewer_role.yaml", strings.ToLower(kbc.Kind)))
	EventuallyWithOffset(1, func() error {
		_, err = kbc.Kubectl.Apply(true, "-f", crdViewerRole)
		return err
	}, time.Minute, time.Second).Should(Succeed())

	By("validating that the created resource object gets reconciled in the controller")
	metricsOutput := curlMetrics(kbc)
	ExpectWithOffset(1, metricsOutput).To(ContainSubstring(fmt.Sprintf(
		`controller_runtime_reconcile_total{controller="%s",result="success"} 1`,
		strings.ToLower(kbc.Kind),
	)))

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
func curlMetrics(kbc *utils.TestContext) string {
	By("reading the metrics token")
	// Filter token query by service account in case more than one exists in a namespace.
	token, err := ServiceAccountToken(kbc)
	ExpectWithOffset(2, err).NotTo(HaveOccurred())
	ExpectWithOffset(2, len(token)).To(BeNumerically(">", 0))

	By("creating a curl pod")
	cmdOpts := []string{
		"run", "curl", "--image=curlimages/curl:7.68.0", "--restart=OnFailure", "--",
		"curl", "-v", "-k", "-H", fmt.Sprintf(`Authorization: Bearer %s`, strings.TrimSpace(token)),
		fmt.Sprintf("https://e2e-%s-controller-manager-metrics-service.%s.svc:8443/metrics",
			kbc.TestSuffix, kbc.Kubectl.Namespace),
	}
	_, err = kbc.Kubectl.CommandInNamespace(cmdOpts...)
	ExpectWithOffset(2, err).NotTo(HaveOccurred())

	By("validating that the curl pod is running as expected")
	verifyCurlUp := func() error {
		// Validate pod status
		status, err := kbc.Kubectl.Get(
			true,
			"pods", "curl", "-o", "jsonpath={.status.phase}")
		ExpectWithOffset(3, err).NotTo(HaveOccurred())
		if status != "Completed" && status != "Succeeded" {
			return fmt.Errorf("curl pod in %s status", status)
		}
		return nil
	}
	EventuallyWithOffset(2, verifyCurlUp, 240*time.Second, time.Second).Should(Succeed())

	By("validating that the metrics endpoint is serving as expected")
	var metricsOutput string
	getCurlLogs := func() string {
		metricsOutput, err = kbc.Kubectl.Logs("curl")
		ExpectWithOffset(3, err).NotTo(HaveOccurred())
		return metricsOutput
	}
	EventuallyWithOffset(2, getCurlLogs, 10*time.Second, time.Second).Should(ContainSubstring("< HTTP/2 200"))

	By("cleaning up the curl pod")
	_, err = kbc.Kubectl.Delete(true, "pods/curl")
	ExpectWithOffset(3, err).NotTo(HaveOccurred())

	return metricsOutput
}

// ServiceAccountToken provides a helper function that can provide you with a service account
// token that you can use to interact with the service. This function leverages the k8s'
// TokenRequest API in raw format in order to make it generic for all version of the k8s that
// is currently being supported in kubebuilder test infra.
// TokenRequest API returns the token in raw JWT format itself. There is no conversion required.
func ServiceAccountToken(kbc *utils.TestContext) (out string, err error) {
	By("Creating the ServiceAccount token")
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
