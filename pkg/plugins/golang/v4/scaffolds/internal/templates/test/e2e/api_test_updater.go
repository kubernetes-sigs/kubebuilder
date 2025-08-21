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

package e2e

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Inserter = &APITestUpdater{}

const (
	suiteSetupMarker  = "e2e-setup"
	testMarker        = "e2e-tests"
	helperMarker      = "e2e-helper-functions"
)

// APITestUpdater updates e2e tests to insert controller/manager tests when APIs are added
type APITestUpdater struct {
	machinery.RepositoryMixin
	machinery.ProjectNameMixin
	machinery.ResourceMixin

	// WireController indicates whether to inject controller tests
	WireController bool
}

// GetPath implements file.Builder
func (*APITestUpdater) GetPath() string {
	return filepath.Join("test", "e2e", "e2e_test.go")
}

// GetIfExistsAction implements file.Builder
func (*APITestUpdater) GetIfExistsAction() machinery.IfExistsAction {
	return machinery.OverwriteFile
}

// GetMarkers implements file.Inserter
func (f *APITestUpdater) GetMarkers() []machinery.Marker {
	return []machinery.Marker{
		machinery.NewMarkerFor(f.GetPath(), testMarker),
		machinery.NewMarkerFor(f.GetPath(), helperMarker),
	}
}

// GetCodeFragments implements file.Inserter
func (f *APITestUpdater) GetCodeFragments() machinery.CodeFragmentsMap {
	if !f.WireController {
		return nil
	}

	fragments := make(machinery.CodeFragmentsMap, 2)
	
	// Add the manager deployment tests when controllers are added
	fragments[machinery.NewMarkerFor(f.GetPath(), testMarker)] = []string{managerTestsCode}
	
	// Add helper functions when controllers are added
	fragments[machinery.NewMarkerFor(f.GetPath(), helperMarker)] = []string{helperFunctionsCode}
	
	return fragments
}

const managerTestsCode = `
	// The following imports are needed for the manager tests:
	// import (
	//     "encoding/json"
	//     "fmt" 
	//     "os"
	//     "os/exec"
	//     "path/filepath"
	//     "time"
	//     "{{ .Repo }}/test/utils"
	// )

	// projectImage is the name of the image which will be build and loaded
	// with the code source changes to be tested.
	var projectImage = "{{ .ProjectName }}:v0.0.1"

	// SetDefaultEventuallyTimeout sets the default timeout for Eventually assertions
	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, err := utils.Run(cmd)
		ExpectWithOffset(2, err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")

		By("validating that the controller-manager pod is running")
		verifyControllerUp := func(g Gomega) {
			cmd := exec.Command("kubectl", "get",
				"pods", "-l", "control-plane=controller-manager",
				"-o", "go-template={{ range .items }}{{ range .status.conditions }}{{ if and (eq .type \"Ready\") (eq .status \"True\") }}{{ printf \"ok\" }}{{ end }}{{ end }}{{ end }}",
				"-n", namespace,
			)
			output, _ := cmd.CombinedOutput()
			g.Expect(string(output)).To(Equal("ok"))
		}
		Eventually(verifyControllerUp).WithTimeout(5 * time.Minute).WithPolling(time.Second).Should(Succeed())

		By("getting the controller-manager pod name")
		cmd = exec.Command("kubectl", "get",
			"pods", "-l", "control-plane=controller-manager",
			"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}{{ \"\\n\" }}{{ end }}{{ end }}",
			"-n", namespace,
		)
		podOutput, err := cmd.CombinedOutput()
		ExpectWithOffset(2, err).NotTo(HaveOccurred(), "Failed to get controller-manager pod name")
		controllerPodName = string(podOutput)
		ExpectWithOffset(2, controllerPodName).ToNot(BeEmpty(), "Controller pod name should not be empty")
	})

	It("should have a running controller-manager pod", func() {
		By("checking that the controller-manager pod is running")
		cmd := exec.Command("kubectl", "get", "pods", controllerPodName, "-n", namespace, "-o", "jsonpath={.status.phase}")
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(output)).To(Equal("Running"), "Controller pod should be in Running phase")
	})

	It("should ensure the metrics endpoint is serving metrics", func() {
		By("creating a ClusterRoleBinding for the service account to allow access to metrics")
		cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
			"--clusterrole={{ .ProjectName }}-metrics-reader",
			fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
		)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

		By("validating that the metrics service is available")
		cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

		By("getting the service account token")
		token, err := serviceAccountToken()
		Expect(err).NotTo(HaveOccurred())
		Expect(token).NotTo(BeEmpty())

		By("waiting for the metrics endpoint to be ready")
		verifyMetricsEndpointReady := func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "endpoints", metricsServiceName, "-n", namespace)
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(ContainSubstring("8443"), "Metrics endpoint is not ready")
		}
		Eventually(verifyMetricsEndpointReady).Should(Succeed())

		By("verifying that the controller manager is serving the metrics server")
		verifyMetricsServerStarted := func(g Gomega) {
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(ContainSubstring("controller-runtime.metrics\\tServing metrics server"),
				"Metrics server not yet started")
		}
		Eventually(verifyMetricsServerStarted).Should(Succeed())

		By("creating the curl-metrics pod to access the metrics endpoint")
		cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
			"--namespace", namespace,
			"--image=curlimages/curl:latest",
			"--overrides",
			fmt.Sprintf(` + "`" + `{
				"spec": {
					"containers": [{
						"name": "curl",
						"image": "curlimages/curl:latest",
						"command": ["/bin/sh", "-c"],
						"args": ["curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics"],
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
			}` + "`" + `, token, metricsServiceName, namespace, serviceAccountName))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

		By("waiting for the curl-metrics pod to complete.")
		verifyCurlUp := func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
				"-o", "jsonpath={.status.phase}",
				"-n", namespace)
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
		}
		Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

		By("getting the metrics by checking curl-metrics logs")
		metricsOutput := getMetricsOutput()
		Expect(metricsOutput).To(ContainSubstring(
			"controller_runtime_reconcile_total",
		))
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("removing the ClusterRoleBinding")
		cmd = exec.Command("kubectl", "delete", "clusterrolebinding", metricsRoleBindingName)
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})
`

// helperFunctionsCode contains helper functions needed by the manager tests
const helperFunctionsCode = `
// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = ` + "`" + `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}` + "`" + `

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() string {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	metricsOutput, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
	Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
	return metricsOutput
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string ` + "`json:\"token\"`" + `
	} ` + "`json:\"status\"`" + `
}
`