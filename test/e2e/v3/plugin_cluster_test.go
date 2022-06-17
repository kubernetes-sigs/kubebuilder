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

package v3

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v3/test/e2e/utils"
)

var _ = Describe("kubebuilder", func() {
	Context("project version 3", func() {
		var (
			kbc *utils.TestContext
			sat bool
		)

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())

			By("installing the Prometheus operator")
			Expect(kbc.InstallPrometheusOperManager()).To(Succeed())

			sat = false
		})

		AfterEach(func() {
			By("clean up API objects created during the test")
			kbc.CleanupManifests(filepath.Join("config", "default"))

			By("uninstalling the Prometheus manager bundle")
			kbc.UninstallPrometheusOperManager()

			By("removing controller image and working dir")
			kbc.Destroy()
		})

		Context("plugin go.kubebuilder.io/v2", func() {
			// Use cert-manager with v1beta2 CRs.
			BeforeEach(func() {
				// Skip if cluster version >= 1.22 because pre v1 CRDs and webhooks no longer exist.
				if srvVer := kbc.K8sVersion.ServerVersion; srvVer.GetMajorInt() >= 1 && srvVer.GetMinorInt() >= 22 {
					Skip(fmt.Sprintf("cluster version %s does not support pre v1 CRDs or webhooks", srvVer.GitVersion))
				}
				By("installing the v1beta2 cert-manager bundle")
				Expect(kbc.InstallCertManager(true)).To(Succeed())
			})
			AfterEach(func() {
				// Skip if cluster version >= 1.22 because pre v1 CRDs and webhooks no longer exist.
				if srvVer := kbc.K8sVersion.ServerVersion; srvVer.GetMajorInt() >= 1 && srvVer.GetMinorInt() >= 22 {
					Skip(fmt.Sprintf("cluster version %s does not support pre v1 CRDs or webhooks", srvVer.GitVersion))
				}
				By("uninstalling the v1beta2 cert-manager bundle")
				kbc.UninstallCertManager(true)
			})

			It("should generate a runnable project go/v2 with default SA", func() {
				// go/v3 uses a unqiue-per-project service account name,
				// while go/v2 still uses "default".
				tmp := kbc.Kubectl.ServiceAccount
				kbc.Kubectl.ServiceAccount = "default"
				defer func() { kbc.Kubectl.ServiceAccount = tmp }()
				GenerateV2(kbc)
				Run(kbc, sat)
			})
		})

		Context("plugin go.kubebuilder.io/v3", func() {
			// Use cert-manager with v1 CRs.
			BeforeEach(func() {
				By("installing the cert-manager bundle")
				Expect(kbc.InstallCertManager(false)).To(Succeed())
			})
			AfterEach(func() {
				By("uninstalling the cert-manager bundle")
				kbc.UninstallCertManager(false)
			})

			It("should generate a runnable project go/v3 with v1 CRDs and Webhooks", func() {
				// Skip if cluster version < 1.16, when v1 CRDs and webhooks did not exist.
				// Skip if cluster version < 1.19, because securityContext.seccompProfile only works from 1.19
				// Otherwise, unknown field "seccompProfile" in io.k8s.api.core.v1.PodSecurityContext will be faced
				if srvVer := kbc.K8sVersion.ServerVersion; srvVer.GetMajorInt() <= 1 && srvVer.GetMinorInt() < 19 {
					Skip(fmt.Sprintf("cluster version %s does not support v1 CRDs or webhooks"+
						"and securityContext.seccompProfile", srvVer.GitVersion))
				}

				GenerateV3(kbc, "v1")

				// only if running on Kubernetes >= 1.24 do we need to generate the ServiceAccount token Secret
				// TODO: Remove this once a better implementation using something like the TokenRequest API
				// is used in the e2e tests
				if srvVer := kbc.K8sVersion.ServerVersion; srvVer.GetMajorInt() == 1 && srvVer.GetMinorInt() >= 24 {
					sat = true
				}

				Run(kbc, sat)
			})
			It("should generate a runnable project with the golang base plugin v3 and kustomize v4-alpha", func() {
				// Skip if cluster version < 1.16, when v1 CRDs and webhooks did not exist.
				// Skip if cluster version < 1.19, because securityContext.seccompProfile only works from 1.19
				// Otherwise, unknown field "seccompProfile" in io.k8s.api.core.v1.PodSecurityContext will be faced
				if srvVer := kbc.K8sVersion.ServerVersion; srvVer.GetMajorInt() <= 1 && srvVer.GetMinorInt() < 19 {
					Skip(fmt.Sprintf("cluster version %s does not support v1 CRDs or webhooks "+
						"and securityContext.seccompProfile", srvVer.GitVersion))
				}

				GenerateV3WithKustomizeV2(kbc, "v1")

				// only if running on Kubernetes >= 1.24 do we need to generate the ServiceAccount token Secret
				// TODO: Remove this once a better implementation using something like the TokenRequest API
				// is used in the e2e tests
				if srvVer := kbc.K8sVersion.ServerVersion; srvVer.GetMajorInt() == 1 && srvVer.GetMinorInt() >= 24 {
					sat = true
				}

				Run(kbc, sat)
			})
			It("should generate a runnable project with v1beta1 CRDs and Webhooks", func() {
				// Skip if cluster version < 1.15, when `.spec.preserveUnknownFields` was not a v1beta1 CRD field.
				// Skip if cluster version < 1.19, because securityContext.seccompProfile only works from 1.19
				// Otherwise, unknown field "seccompProfile" in io.k8s.api.core.v1.PodSecurityContext will be faced
				// Skip if cluster version >= 1.22 because pre v1 CRDs and webhooks no longer exist.
				if srvVer := kbc.K8sVersion.ServerVersion; srvVer.GetMajorInt() <= 1 && srvVer.GetMinorInt() < 19 ||
					srvVer.GetMajorInt() <= 1 && srvVer.GetMinorInt() >= 22 {
					Skip(fmt.Sprintf("cluster version %s does not support project defaults "+
						"and securityContext.seccompProfile", srvVer.GitVersion))
				}

				GenerateV3(kbc, "v1beta1")
				Run(kbc, sat)
			})

			It("should generate a runnable project go/v3 with v1 CRDs and Webhooks and the deployImagePlugin", func() {
				// Skip if cluster version < 1.16, when v1 CRDs and webhooks did not exist.
				// Skip if cluster version < 1.19, because securityContext.seccompProfile only works from 1.19
				// Otherwise, unknown field "seccompProfile" in io.k8s.api.core.v1.PodSecurityContext will be faced
				if srvVer := kbc.K8sVersion.ServerVersion; srvVer.GetMajorInt() <= 1 && srvVer.GetMinorInt() < 19 {
					Skip(fmt.Sprintf("cluster version %s does not support v1 CRDs or webhooks"+
						"and securityContext.seccompProfile", srvVer.GitVersion))
				}

				GenerateV3WithDeployImage(kbc)

				// only if running on Kubernetes >= 1.24 do we need to generate the ServiceAccount token Secret
				// TODO: Remove this once a better implementation using something like the TokenRequest API
				// is used in the e2e tests
				if srvVer := kbc.K8sVersion.ServerVersion; srvVer.GetMajorInt() == 1 && srvVer.GetMinorInt() >= 24 {
					sat = true
				}

				// THe following command will test/check if the Operator itself is deployed
				// TODO: we need have a test here where we check if the Deployment with the
				// image will work well as when we change the CR spec from 1 to 3 for example
				// that will scale more instances of the Deployment/Pod created for the
				// Operand image will work as should be
				Run(kbc, sat)
			})
		})
	})
})

// Run runs a set of e2e tests for a scaffolded project defined by a TestContext.
func Run(kbc *utils.TestContext, sat bool) {
	var controllerPodName string
	var err error

	By("updating the go.mod")
	err = kbc.Tidy()
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("building the controller image")
	err = kbc.Make("docker-build", "IMG="+kbc.ImageName)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("loading the controller docker image into the kind cluster")
	err = kbc.LoadImageToKindCluster()
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	// NOTE: If you want to run the test against a GKE cluster, you will need to grant yourself permission.
	// Otherwise, you may see "... is forbidden: attempt to grant extra privileges"
	// $ kubectl create clusterrolebinding myname-cluster-admin-binding \
	// --clusterrole=cluster-admin --user=myname@mycompany.com
	// https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control
	By("deploying the controller-manager")
	err = kbc.Make("deploy", "IMG="+kbc.ImageName)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

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

	if sat {
		ServiceAccountToken(kbc)
	}

	_ = curlMetrics(kbc)

	By("validating that cert-manager has provisioned the certificate Secret")
	EventuallyWithOffset(1, func() error {
		_, err := kbc.Kubectl.Get(
			true,
			"secrets", "webhook-server-cert")
		return err
	}, time.Minute, time.Second).Should(Succeed())

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

	By("creating an instance of the CR")
	// currently controller-runtime doesn't provide a readiness probe, we retry a few times
	// we can change it to probe the readiness endpoint after CR supports it.
	sampleFile := filepath.Join("config", "samples",
		fmt.Sprintf("%s_%s_%s.yaml", kbc.Group, kbc.Version, strings.ToLower(kbc.Kind)))

	sampleFilePath, err := filepath.Abs(filepath.Join(fmt.Sprintf("e2e-%s", kbc.TestSuffix), sampleFile))
	Expect(err).To(Not(HaveOccurred()))

	f, err := os.OpenFile(sampleFilePath, os.O_APPEND|os.O_WRONLY, 0644)
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

// curlMetrics curl's the /metrics endpoint, returning all logs once a 200 status is returned.
func curlMetrics(kbc *utils.TestContext) string {
	By("reading the metrics token")
	// Filter token query by service account in case more than one exists in a namespace.
	// TODO: Parsing a token this way is not ideal or best practice and should not be used.
	// Instead, a TokenRequest should be created to get a token to use for this step.
	query := fmt.Sprintf(`{.items[?(@.metadata.annotations.kubernetes\.io/service-account\.name=="%s")].data.token}`,
		kbc.Kubectl.ServiceAccount,
	)
	b64Token, err := kbc.Kubectl.Get(true, "secrets", "-o=jsonpath="+query)
	ExpectWithOffset(2, err).NotTo(HaveOccurred())
	token, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64Token))
	ExpectWithOffset(2, err).NotTo(HaveOccurred())
	ExpectWithOffset(2, len(token)).To(BeNumerically(">", 0))

	By("creating a curl pod")
	cmdOpts := []string{
		"run", "curl", "--image=curlimages/curl:7.68.0", "--restart=OnFailure", "--",
		"curl", "-v", "-k", "-H", fmt.Sprintf(`Authorization: Bearer %s`, token),
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

// TODO: Remove this when and other related functions when
// tests have been changed to use a better implementation.
// ServiceAccountToken creates the ServiceAccount token Secret.
// This is to be used when Kubernetes cluster version is >= 1.24.
// In k8s 1.24+ a ServiceAccount token Secret is no longer
// automatically generated
func ServiceAccountToken(kbc *utils.TestContext) {
	// As of Kubernetes 1.24 a ServiceAccount no longer has a ServiceAccount token
	// secret autogenerated. We have to create it manually here
	By("Creating the ServiceAccount token")
	secretFile, err := createServiceAccountToken(kbc.Kubectl.ServiceAccount, kbc.Dir)
	Expect(err).NotTo(HaveOccurred())
	Eventually(func() error {
		_, err = kbc.Kubectl.Apply(true, "-f", secretFile)
		return err
	}, time.Minute, time.Second).Should(Succeed())
}

var saSecretTemplate = `---
apiVersion: v1
kind: Secret
type: kubernetes.io/service-account-token
metadata:
  name: %s
  annotations:
    kubernetes.io/service-account.name: "%s"
`

// createServiceAccountToken writes a service account token secret to a file.
// It returns a string to the file or an error if it fails to write the file
func createServiceAccountToken(name string, dir string) (string, error) {
	secretName := name + "-secret"
	fileName := dir + "/" + secretName + ".yaml"
	err := os.WriteFile(fileName, []byte(fmt.Sprintf(saSecretTemplate, secretName, name)), 0777)
	if err != nil {
		return "", err
	}

	return fileName, nil
}
