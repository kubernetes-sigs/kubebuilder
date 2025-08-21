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
	suiteSetupMarker = "e2e-setup"
	testMarker       = "e2e-tests"
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
	}
}

// GetCodeFragments implements file.Inserter
func (f *APITestUpdater) GetCodeFragments() machinery.CodeFragmentsMap {
	if !f.WireController {
		return nil
	}

	fragments := make(machinery.CodeFragmentsMap, 1)
	
	// Add the manager deployment tests when controllers are added
	fragments[machinery.NewMarkerFor(f.GetPath(), testMarker)] = []string{managerTestsCode}
	
	return fragments
}

const managerTestsCode = `
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

	It("should ensure the metrics service is available", func() {
		By("creating a ClusterRoleBinding for the service account to allow access to metrics")
		cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
			fmt.Sprintf("--clusterrole=%s-metrics-reader", {{ .ProjectName | quote }}),
			fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
		)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

		By("getting the service account token")
		token, err := serviceAccountToken()
		Expect(err).NotTo(HaveOccurred())
		Expect(token).ToNot(BeEmpty())

		By("waiting for the metrics service to be available")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err := cmd.CombinedOutput()
			g.Expect(err).NotTo(HaveOccurred())
		}).Should(Succeed())
	})

	AfterAll(func() {
		By("cleaning up the controller")
		cmd := exec.Command("make", "undeploy")
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