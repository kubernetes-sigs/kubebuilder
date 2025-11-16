//go:build integration

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

package scaffolds

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// These tests verify the base scaffolding that both init and edit create.
// Both scaffolders generate RuntimeManifest, ResourcesManifest, and CustomMetricsConfig.
var _ = Describe("Base Scaffolds (Init & Edit)", func() {
	var (
		fs         machinery.Filesystem
		scaffolder *initScaffolder
		tmpDir     string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "grafana-scaffolds-test-*")
		Expect(err).NotTo(HaveOccurred())

		// Change to tmpDir so relative paths work correctly
		err = os.Chdir(tmpDir)
		Expect(err).NotTo(HaveOccurred())

		fs = machinery.Filesystem{
			FS: afero.NewBasePathFs(afero.NewOsFs(), tmpDir),
		}
		scaffolder = &initScaffolder{}
		scaffolder.InjectFS(fs)
	})

	AfterEach(func() {
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
	})

	Context("when scaffolding grafana manifests", func() {
		It("should create controller-runtime metrics dashboard", func() {
			err := scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			By("verifying the dashboard file exists")
			runtimePath := filepath.Join("grafana", "controller-runtime-metrics.json")
			Expect(fileExistsScaffolds(runtimePath)).To(BeTrue())

			By("verifying the dashboard contains controller-runtime metrics")
			content, err := os.ReadFile(runtimePath)
			Expect(err).NotTo(HaveOccurred())
			contentStr := string(content)
			Expect(contentStr).To(ContainSubstring("controller_runtime"))
			Expect(contentStr).To(ContainSubstring("reconcile"))
			Expect(contentStr).To(ContainSubstring("workqueue"))
		})

		It("should create controller-resources metrics dashboard", func() {
			err := scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			By("verifying the dashboard file exists")
			resourcesPath := filepath.Join("grafana", "controller-resources-metrics.json")
			Expect(fileExistsScaffolds(resourcesPath)).To(BeTrue())

			By("verifying the dashboard contains CPU and memory metrics")
			content, err := os.ReadFile(resourcesPath)
			Expect(err).NotTo(HaveOccurred())
			contentStr := string(content)
			Expect(contentStr).To(ContainSubstring("CPU"))
			Expect(contentStr).To(ContainSubstring("Memory"))
		})

		It("should create custom metrics config template", func() {
			err := scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			By("verifying the config file exists")
			configPath := filepath.Join("grafana", "custom-metrics", "config.yaml")
			Expect(fileExistsScaffolds(configPath)).To(BeTrue())

			By("verifying the config has proper structure")
			content, err := os.ReadFile(configPath)
			Expect(err).NotTo(HaveOccurred())
			contentStr := string(content)
			Expect(contentStr).To(ContainSubstring("customMetrics:"))
			Expect(contentStr).To(ContainSubstring("# Example:"))
			Expect(contentStr).To(ContainSubstring("metric:"))
			Expect(contentStr).To(ContainSubstring("type:"))
		})

		It("should create valid JSON dashboards", func() {
			err := scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			By("verifying runtime dashboard is valid JSON")
			runtimePath := filepath.Join("grafana", "controller-runtime-metrics.json")
			content, err := os.ReadFile(runtimePath)
			Expect(err).NotTo(HaveOccurred())
			contentStr := string(content)
			Expect(contentStr).To(HavePrefix("{"))
			Expect(contentStr).To(HaveSuffix("}\n"))
			Expect(contentStr).To(ContainSubstring(`"__inputs"`))
			Expect(contentStr).To(ContainSubstring(`"panels"`))
		})

		It("should configure Prometheus datasource", func() {
			err := scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			By("verifying Prometheus datasource configuration")
			runtimePath := filepath.Join("grafana", "controller-runtime-metrics.json")
			content, err := os.ReadFile(runtimePath)
			Expect(err).NotTo(HaveOccurred())
			contentStr := string(content)
			Expect(contentStr).To(ContainSubstring("DS_PROMETHEUS"))
			Expect(contentStr).To(ContainSubstring(`"type": "datasource"`))
			Expect(contentStr).To(ContainSubstring(`"pluginId": "prometheus"`))
		})

		It("should include template variables", func() {
			err := scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			By("verifying template variables exist")
			runtimePath := filepath.Join("grafana", "controller-runtime-metrics.json")
			content, err := os.ReadFile(runtimePath)
			Expect(err).NotTo(HaveOccurred())
			contentStr := string(content)
			Expect(contentStr).To(ContainSubstring(`"templating"`))
			Expect(contentStr).To(ContainSubstring(`"name": "namespace"`))
			Expect(contentStr).To(ContainSubstring(`"name": "job"`))
		})
	})
})

func fileExistsScaffolds(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
