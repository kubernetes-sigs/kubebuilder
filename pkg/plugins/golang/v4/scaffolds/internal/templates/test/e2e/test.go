/*
Copyright 2022 The Kubernetes Authors.

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
	"bytes"
	"fmt"
	log "log/slog"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var (
	_ machinery.Template = &Test{}
	_ machinery.Inserter = &WebhookTestUpdater{}
)

const (
	webhookChecksMarker           = "e2e-webhooks-checks"
	metricsWebhookReadinessMarker = "e2e-metrics-webhooks-readiness"
)

// Test defines the basic setup for the e2e test
type Test struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
	machinery.RepositoryMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults set defaults for this template
func (f *Test) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("test", "e2e", "e2e_test.go")
	}

	// This is where the template body is defined with markers
	f.TemplateBody = testCodeTemplate

	return nil
}

// WebhookTestUpdater updates e2e_test.go to insert additional webhook validation tests
type WebhookTestUpdater struct {
	machinery.RepositoryMixin
	machinery.ProjectNameMixin
	machinery.ResourceMixin
	WireWebhook bool
}

// GetPath implements file.Builder
func (*WebhookTestUpdater) GetPath() string {
	return filepath.Join("test", "e2e", "e2e_test.go")
}

// GetIfExistsAction implements file.Builder
func (*WebhookTestUpdater) GetIfExistsAction() machinery.IfExistsAction {
	return machinery.OverwriteFile // Ensures only the marker is replaced
}

// GetMarkers implements file.Inserter
func (f *WebhookTestUpdater) GetMarkers() []machinery.Marker {
	return []machinery.Marker{
		machinery.NewMarkerFor(f.GetPath(), webhookChecksMarker),
		machinery.NewMarkerFor(f.GetPath(), metricsWebhookReadinessMarker),
	}
}

// GetCodeFragments implements file.Inserter
func (f *WebhookTestUpdater) GetCodeFragments() machinery.CodeFragmentsMap {
	if !f.WireWebhook {
		return nil
	}

	filePath := f.GetPath()

	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Warn("Unable to read file", "file", filePath, "error", err)
		log.Warn("Webhook test code injection will be skipped for this file.")
		log.Warn("This typically occurs when the file was removed and is missing.")
		log.Warn("If you intend to scaffold webhook tests, ensure the file and its markers exist.")
		return nil
	}

	codeFragments := machinery.CodeFragmentsMap{}
	markers := f.GetMarkers()

	for _, marker := range markers {
		markerStr := marker.String()
		if !bytes.Contains(content, []byte(markerStr)) {
			log.Warn("Marker not found in file, skipping webhook test code injection",
				"marker", markerStr,
				"file_path", filePath)
			continue // skip this marker
		}

		switch {
		case strings.Contains(markerStr, webhookChecksMarker):
			var fragments []string
			fragments = append(fragments, webhookChecksFragment)

			if f.Resource != nil && f.Resource.HasDefaultingWebhook() {
				mutatingWebhookCode := fmt.Sprintf(mutatingWebhookChecksFragment, f.ProjectName)
				fragments = append(fragments, mutatingWebhookCode)
			}

			if f.Resource != nil && f.Resource.HasValidationWebhook() {
				validatingWebhookCode := fmt.Sprintf(validatingWebhookChecksFragment, f.ProjectName)
				fragments = append(fragments, validatingWebhookCode)
			}

			if f.Resource != nil && f.Resource.HasConversionWebhook() {
				conversionWebhookCode := fmt.Sprintf(
					conversionWebhookChecksFragment,
					f.Resource.Kind,
					f.Resource.Plural+"."+f.Resource.Group+"."+f.Resource.Domain,
				)
				fragments = append(fragments, conversionWebhookCode)
			}

			if len(fragments) > 0 {
				codeFragments[marker] = fragments
			}
		case strings.Contains(markerStr, metricsWebhookReadinessMarker):
			webhookServiceName := fmt.Sprintf("%s-webhook-service", f.ProjectName)
			fragments := []string{fmt.Sprintf(metricsWebhookReadinessFragment, webhookServiceName)}
			codeFragments[marker] = fragments
		}
	}

	if len(codeFragments) == 0 {
		return nil
	}

	return codeFragments
}

const webhookChecksFragment = `It("should provisioned cert-manager", func() {
	By("validating that cert-manager has the certificate Secret")
	verifyCertManager := func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "secrets", "webhook-server-cert", "-n", namespace)
		_, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
	}
	Eventually(verifyCertManager).Should(Succeed())
})

`

const mutatingWebhookChecksFragment = `It("should have CA injection for mutating webhooks", func() {
	By("checking CA injection for mutating webhooks")
	verifyCAInjection := func(g Gomega) {
		cmd := exec.Command("kubectl", "get",
			"mutatingwebhookconfigurations.admissionregistration.k8s.io",
			"%s-mutating-webhook-configuration",
			"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
		mwhOutput, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(len(mwhOutput)).To(BeNumerically(">", 10))
	}
	Eventually(verifyCAInjection).Should(Succeed())
})

`

const validatingWebhookChecksFragment = `It("should have CA injection for validating webhooks", func() {
	By("checking CA injection for validating webhooks")
	verifyCAInjection := func(g Gomega) {
		cmd := exec.Command("kubectl", "get",
			"validatingwebhookconfigurations.admissionregistration.k8s.io",
			"%s-validating-webhook-configuration",
			"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
		vwhOutput, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(len(vwhOutput)).To(BeNumerically(">", 10))
	}
	Eventually(verifyCAInjection).Should(Succeed())
})

`

const conversionWebhookChecksFragment = `It("should have CA injection for %[1]s conversion webhook", func() {
	By("checking CA injection for %[1]s conversion webhook")
	verifyCAInjection := func(g Gomega) {
		cmd := exec.Command("kubectl", "get",
			"customresourcedefinitions.apiextensions.k8s.io",
			"%[2]s",
			"-o", "go-template={{ .spec.conversion.webhook.clientConfig.caBundle }}")
		vwhOutput, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(len(vwhOutput)).To(BeNumerically(">", 10))
	}
	Eventually(verifyCAInjection).Should(Succeed())
})

`

const metricsWebhookReadinessFragment = `By("waiting for the webhook service endpoints to be ready")
	verifyWebhookEndpointsReady := func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "endpointslices.discovery.k8s.io", "-n", namespace,
			"-l", "kubernetes.io/service-name=%s",
			"-o", "jsonpath={range .items[*]}{range .endpoints[*]}{.addresses[*]}{end}{end}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred(), "Webhook endpoints should exist")
		g.Expect(output).ShouldNot(BeEmpty(), "Webhook endpoints not yet ready")
	}
	Eventually(verifyWebhookEndpointsReady, 3*time.Minute, time.Second).Should(Succeed())

`

var testCodeTemplate = `//go:build e2e
// +build e2e

{{ .Boilerplate }}

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"{{ .Repo }}/test/utils"
)

// namespace where the project is deployed in
const namespace = "{{ .ProjectName }}-system"
// serviceAccountName created for the project
const serviceAccountName = "{{ .ProjectName }}-controller-manager"
// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "{{ .ProjectName }}-controller-manager-metrics-service"
// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "{{ .ProjectName }}-metrics-binding"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("labeling the namespace to enforce the restricted security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
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

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				// Get the name of the controller-manager pod
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{"{{"}} range .items {{"}}"}}" +
					"{{"{{"}} if not .metadata.deletionTimestamp {{"}}"}}" +
					"{{"{{"}} .metadata.name {{"}}"}}"+
					"{{"{{"}} \"\\n\" {{"}}"}}{{"{{"}} end {{"}}"}}{{"{{"}} end {{"}}"}}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				// Validate the pod's status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole={{ .ProjectName}}-metrics-reader",
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

			By("ensuring the controller pod is ready")
			verifyControllerPodReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pod", controllerPodName, "-n", namespace,
					"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("True"), "Controller pod not ready")
			}
			Eventually(verifyControllerPodReady, 3*time.Minute, time.Second).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("Serving metrics server"),
 					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted, 3*time.Minute, time.Second).Should(Succeed())

			// +kubebuilder:scaffold:e2e-metrics-webhooks-readiness

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
			Eventually(verifyCurlUp, 5 * time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			verifyMetricsAvailable := func(g Gomega) {
				metricsOutput, err := getMetricsOutput()
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
				g.Expect(metricsOutput).NotTo(BeEmpty())
				g.Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
			}
			Eventually(verifyMetricsAvailable, 2*time.Minute).Should(Succeed())
		})

		// +kubebuilder:scaffold:e2e-webhooks-checks

		// TODO: Customize the e2e test suite with scenarios specific to your project.
		// Consider applying sample/CR(s) and check their status and/or verifying
		// the reconciliation by using the metrics, i.e.:
		// metricsOutput, err := getMetricsOutput()
		// Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
		// Expect(metricsOutput).To(ContainSubstring(
		//    fmt.Sprintf(` + "`controller_runtime_reconcile_total{controller=\"%s\",result=\"success\"} 1`" + `,
		//    strings.ToLower(<Kind>),
		// ))
	})
})

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
func getMetricsOutput() (string, error) {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	return utils.Run(cmd)
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string ` + "`json:\"token\"`" + `
	} ` + "`json:\"status\"`" + `
}
`
