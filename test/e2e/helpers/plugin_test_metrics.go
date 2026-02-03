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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

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

// GetControllerPodName validates that the controller-manager pod is running and returns its name
func GetControllerPodName(kbc *utils.TestContext) string {
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

// GetMetricsOutput returns the metrics output from curl pod
// namePrefix is the prefix for service names (e.g., "e2e-{suffix}" or "custom-operator" from fullnameOverride)
func GetMetricsOutput(controllerPodName, namePrefix string, kbc *utils.TestContext) string {
	var err error
	// All Kubebuilder projects are cluster-scoped, so use ClusterRoleBinding
	_, err = kbc.Kubectl.Command(
		"get", "clusterrolebinding", fmt.Sprintf("metrics-%s", kbc.TestSuffix),
	)
	if err != nil && strings.Contains(err.Error(), "NotFound") {
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
		"service", fmt.Sprintf("%s-controller-manager-metrics-service", namePrefix),
	)
	Expect(err).NotTo(HaveOccurred(), "Controller-manager service should exist")

	By("ensuring the service endpoint is ready")
	metricsServiceName := fmt.Sprintf("%s-controller-manager-metrics-service", namePrefix)
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
	Eventually(verifyControllerPodReady, defaultTimeout, defaultPollingInterval).Should(Succeed())

	webhookServiceName := fmt.Sprintf("%s-webhook-service", namePrefix)
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
		Eventually(checkWebhookEndpoint, defaultTimeout, defaultPollingInterval).Should(Succeed(),
			"Webhook service endpoints should be ready")
	}

	By("creating a curl pod to access the metrics endpoint")
	cmdOpts := cmdOptsToCreateCurlPod(namePrefix, kbc, token)
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

// ValidateMetricsUnavailable validates that metrics are not exposed
// namePrefix is the prefix for service names (e.g., "e2e-{suffix}" or "custom-operator" from fullnameOverride)
func ValidateMetricsUnavailable(namePrefix string, kbc *utils.TestContext) {
	// All Kubebuilder projects are cluster-scoped, so use ClusterRoleBinding
	_, err := kbc.Kubectl.Command(
		"create", "clusterrolebinding", fmt.Sprintf("metrics-%s", kbc.TestSuffix),
		fmt.Sprintf("--clusterrole=%s-metrics-reader", namePrefix),
		fmt.Sprintf("--serviceaccount=%s:%s", kbc.Kubectl.Namespace, kbc.Kubectl.ServiceAccount))
	Expect(err).NotTo(HaveOccurred())

	token, err := serviceAccountToken(kbc)
	Expect(err).NotTo(HaveOccurred())
	Expect(token).NotTo(BeEmpty())

	By("creating a curl pod to access the metrics endpoint")
	cmdOpts := cmdOptsToCreateCurlPod(namePrefix, kbc, token)
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

func cmdOptsToCreateCurlPod(namePrefix string, kbc *utils.TestContext, token string) []string {
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
					"args": ["curl -v -k -H 'Authorization: Bearer %s' https://%s-controller-manager-metrics-service.%s.svc.cluster.local:8443/metrics"],
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
    }`, token, namePrefix, kbc.Kubectl.Namespace, kbc.Kubectl.ServiceAccount),
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
	Eventually(getToken, 2*time.Minute, time.Second).Should(Succeed())

	return out, nil
}
