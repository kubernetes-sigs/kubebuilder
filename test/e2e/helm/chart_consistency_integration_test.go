//go:build integration

/*
Copyright 2026 The Kubernetes Authors.

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

package helm_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

func TestHelmChartConsistency(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Helm Chart Consistency Integration Suite")
}

// This test validates that Helm chart templates work correctly when:
// 1. The Chart.yaml name differs from the project name
// 2. Kustomize uses a custom namePrefix different from the project name
//
// The key requirement is that all templates use the SAME helper function names
// (based on ProjectName) so the chart can be successfully rendered.
var _ = Describe("Helm Chart Template Consistency", func() {
	var kbc *utils.TestContext

	BeforeEach(func() {
		var err error
		kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(kbc.Prepare()).To(Succeed())
	})

	AfterEach(func() {
		kbc.Destroy()
	})

	It("should use consistent helper names across all templates", func() {
		By("initializing a project with a specific name")
		err := kbc.Init(
			"--plugins", "go/v4",
			"--project-version", "3",
			"--domain", kbc.Domain,
			"--project-name", "testproject",
		)
		Expect(err).NotTo(HaveOccurred())

		By("creating an API with webhooks")
		err = kbc.CreateAPI(
			"--group", kbc.Group,
			"--version", kbc.Version,
			"--kind", kbc.Kind,
			"--resource",
			"--controller",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		err = kbc.CreateWebhook(
			"--group", kbc.Group,
			"--version", kbc.Version,
			"--kind", kbc.Kind,
			"--defaulting",
			"--programmatic-validation",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("generating manifests and Helm chart")
		err = kbc.Make("manifests", "build-installer")
		Expect(err).NotTo(HaveOccurred())

		err = kbc.Edit("--plugins=helm/v2-alpha")
		Expect(err).NotTo(HaveOccurred())

		chartPath := filepath.Join(kbc.Dir, "dist", "chart")
		Expect(chartPath).To(BeADirectory())

		By("verifying all templates use project-name-based helpers")
		helperPrefix := "testproject"

		// Check _helpers.tpl defines helpers with project name
		helpersContent, err := os.ReadFile(filepath.Join(chartPath, "templates", "_helpers.tpl"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(helpersContent)).To(ContainSubstring(`define "` + helperPrefix + `.name"`))
		Expect(string(helpersContent)).To(ContainSubstring(`define "` + helperPrefix + `.resourceName"`))

		// Check test template uses the same helper names
		testContent, err := os.ReadFile(filepath.Join(chartPath, "templates", "tests", "test-connection.yaml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(testContent)).To(ContainSubstring(`include "` + helperPrefix + `.resourceName"`))
		Expect(string(testContent)).To(ContainSubstring(`include "` + helperPrefix + `.name"`))

		// Check ServiceMonitor if it exists
		serviceMonitorPath := filepath.Join(chartPath, "templates", "monitoring", "servicemonitor.yaml")
		if _, err := os.Stat(serviceMonitorPath); err == nil {
			smContent, err := os.ReadFile(serviceMonitorPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(smContent)).To(ContainSubstring(`include "` + helperPrefix + `.resourceName"`))
		}

		// Check converted RBAC templates use the same helpers
		rbacContent, err := os.ReadFile(filepath.Join(chartPath, "templates", "rbac", "controller-manager.yaml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(rbacContent)).To(ContainSubstring(`include "` + helperPrefix + `.resourceName"`))

		By("validating chart can be rendered with helm template")
		cmd := exec.Command("helm", "template", "test-release", chartPath)
		cmd.Dir = kbc.Dir
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), "helm template should succeed: %s", string(output))

		By("changing Chart.yaml name and verifying it still renders")
		chartYamlPath := filepath.Join(chartPath, "Chart.yaml")
		chartContent, err := os.ReadFile(chartYamlPath)
		Expect(err).NotTo(HaveOccurred())
		modifiedContent := strings.ReplaceAll(string(chartContent), "name: testproject", "name: different-name")
		err = os.WriteFile(chartYamlPath, []byte(modifiedContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		cmd = exec.Command("helm", "template", "test-release", chartPath)
		cmd.Dir = kbc.Dir
		output, err = cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(),
			"helm template should still succeed with changed Chart.yaml name: %s", string(output))

		// Verify the output has expected naming patterns
		renderedYaml := string(output)
		Expect(renderedYaml).To(ContainSubstring("test-release-different-name-"))
	})

	It("should work correctly when kustomize uses custom namePrefix", func() {
		By("initializing a project")
		err := kbc.Init(
			"--plugins", "go/v4",
			"--project-version", "3",
			"--domain", kbc.Domain,
		)
		Expect(err).NotTo(HaveOccurred())

		By("creating an API with webhooks")
		err = kbc.CreateAPI(
			"--group", kbc.Group,
			"--version", kbc.Version,
			"--kind", kbc.Kind,
			"--resource",
			"--controller",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		err = kbc.CreateWebhook(
			"--group", kbc.Group,
			"--version", kbc.Version,
			"--kind", kbc.Kind,
			"--defaulting",
			"--programmatic-validation",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("adding custom namePrefix to kustomization.yaml")
		kustomizationPath := filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml")
		content, err := os.ReadFile(kustomizationPath)
		Expect(err).NotTo(HaveOccurred())

		// Prepend custom namePrefix
		modifiedContent := "namePrefix: custom-prefix-\n" + string(content)
		err = os.WriteFile(kustomizationPath, []byte(modifiedContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		By("generating manifests and Helm chart")
		err = kbc.Make("manifests", "build-installer")
		Expect(err).NotTo(HaveOccurred())

		err = kbc.Edit("--plugins=helm/v2-alpha")
		Expect(err).NotTo(HaveOccurred())

		chartPath := filepath.Join(kbc.Dir, "dist", "chart")
		projectName := "e2e-" + kbc.TestSuffix

		By("verifying _helpers.tpl uses project name (not kustomize prefix)")
		helpersContent, err := os.ReadFile(filepath.Join(chartPath, "templates", "_helpers.tpl"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(helpersContent)).To(ContainSubstring(`define "` + projectName + `.name"`))
		Expect(string(helpersContent)).To(ContainSubstring(`define "` + projectName + `.resourceName"`))

		By("verifying test template uses project name helpers (not kustomize prefix)")
		testContent, err := os.ReadFile(filepath.Join(chartPath, "templates", "tests", "test-connection.yaml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(testContent)).To(ContainSubstring(`include "` + projectName + `.resourceName"`))
		// Should NOT use the kustomize prefix in helper calls
		Expect(string(testContent)).NotTo(ContainSubstring(`include "custom-prefix.resourceName"`))

		By("verifying chart renders successfully")
		cmd := exec.Command("helm", "template", "test-release", chartPath)
		cmd.Dir = kbc.Dir
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), "helm template should succeed: %s", string(output))
	})
})
