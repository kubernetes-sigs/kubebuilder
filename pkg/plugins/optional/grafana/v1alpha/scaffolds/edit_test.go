//go:build integration
// +build integration

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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/grafana/v1alpha/scaffolds/internal/templates"
)

var _ = Describe("Edit Scaffolder", func() {
	var (
		fs         machinery.Filesystem
		scaffolder *editScaffolder
		tmpDir     string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "grafana-test-*")
		Expect(err).NotTo(HaveOccurred())

		// Change to tmpDir so relative paths work correctly
		err = os.Chdir(tmpDir)
		Expect(err).NotTo(HaveOccurred())

		fs = machinery.Filesystem{
			FS: afero.NewBasePathFs(afero.NewOsFs(), tmpDir),
		}
		scaffolder = &editScaffolder{}
		scaffolder.InjectFS(fs)
	})

	AfterEach(func() {
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
	})

	Describe("configReader", func() {
		It("should parse valid custom metrics config", func() {
			configContent := `---
customMetrics:
  - metric: foo_bar
    type: counter
  - metric: baz_qux
    type: histogram
`
			reader := strings.NewReader(configContent)
			items, err := configReader(reader)
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(2))
			Expect(items[0].Metric).To(Equal("foo_bar"))
			Expect(items[0].Type).To(Equal("counter"))
			Expect(items[1].Metric).To(Equal("baz_qux"))
			Expect(items[1].Type).To(Equal("histogram"))
		})

		It("should handle empty config", func() {
			configContent := `---
customMetrics:
`
			reader := strings.NewReader(configContent)
			items, err := configReader(reader)
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(BeEmpty())
		})

		It("should return error for invalid YAML", func() {
			configContent := `invalid: yaml: content:`
			reader := strings.NewReader(configContent)
			_, err := configReader(reader)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("validateCustomMetricItems", func() {
		It("should filter out items missing required fields", func() {
			items := []templates.CustomMetricItem{
				{Metric: "valid_metric", Type: "counter"},
				{Metric: "", Type: "gauge"}, // Missing metric
				{Metric: "another_valid", Type: "histogram"},
				{Type: "counter"}, // Missing metric
			}

			validated := validateCustomMetricItems(items)
			Expect(validated).To(HaveLen(2))
			Expect(validated[0].Metric).To(Equal("valid_metric"))
			Expect(validated[1].Metric).To(Equal("another_valid"))
		})

		It("should fill missing expr for counter type", func() {
			items := []templates.CustomMetricItem{
				{Metric: "foo_bar", Type: "counter"},
			}

			validated := validateCustomMetricItems(items)
			Expect(validated).To(HaveLen(1))
			Expect(validated[0].Expr).To(ContainSubstring("sum(rate(foo_bar"))
			Expect(validated[0].Expr).To(ContainSubstring(`{job=\"$job\", namespace=\"$namespace\"}`))
		})

		It("should fill missing expr for histogram type", func() {
			items := []templates.CustomMetricItem{
				{Metric: "foo_bar", Type: "histogram"},
			}

			validated := validateCustomMetricItems(items)
			Expect(validated).To(HaveLen(1))
			Expect(validated[0].Expr).To(ContainSubstring("histogram_quantile(0.90"))
			Expect(validated[0].Expr).To(ContainSubstring("foo_bar"))
		})

		It("should fill missing expr for gauge type", func() {
			items := []templates.CustomMetricItem{
				{Metric: "foo_bar", Type: "gauge"},
			}

			validated := validateCustomMetricItems(items)
			Expect(validated).To(HaveLen(1))
			Expect(validated[0].Expr).To(Equal("foo_bar"))
		})

		It("should not override existing expr", func() {
			customExpr := "my_custom_expr"
			items := []templates.CustomMetricItem{
				{Metric: "foo_bar", Type: "counter", Expr: customExpr},
			}

			validated := validateCustomMetricItems(items)
			Expect(validated).To(HaveLen(1))
			Expect(validated[0].Expr).To(Equal(customExpr))
		})
	})

	Describe("hasFields", func() {
		It("should return true when expr exists", func() {
			item := templates.CustomMetricItem{Expr: "some_expr"}
			Expect(hasFields(item)).To(BeTrue())
		})

		It("should return true when metric and valid type exist", func() {
			validTypes := []string{"counter", "gauge", "histogram"}
			for _, t := range validTypes {
				item := templates.CustomMetricItem{Metric: "foo_bar", Type: t}
				Expect(hasFields(item)).To(BeTrue(), "Expected type %s to be valid", t)
			}
		})

		It("should return false when metric is missing", func() {
			item := templates.CustomMetricItem{Type: "counter"}
			Expect(hasFields(item)).To(BeFalse())
		})

		It("should return false when type is invalid", func() {
			item := templates.CustomMetricItem{Metric: "foo_bar", Type: "invalid"}
			Expect(hasFields(item)).To(BeFalse())
		})

		It("should return false when both metric and expr are missing", func() {
			item := templates.CustomMetricItem{Type: "counter"}
			Expect(hasFields(item)).To(BeFalse())
		})
	})

	Describe("fillMissingUnit", func() {
		It("should detect seconds unit", func() {
			items := []templates.CustomMetricItem{
				{Metric: "foo_seconds"},
				{Metric: "bar_duration"},
			}

			for _, item := range items {
				filled := fillMissingUnit(item)
				Expect(filled.Unit).To(Equal("s"))
			}
		})

		It("should detect bytes unit", func() {
			item := templates.CustomMetricItem{Metric: "foo_bytes"}
			filled := fillMissingUnit(item)
			Expect(filled.Unit).To(Equal("bytes"))
		})

		It("should detect percent unit", func() {
			item := templates.CustomMetricItem{Metric: "foo_ratio"}
			filled := fillMissingUnit(item)
			Expect(filled.Unit).To(Equal("percent"))
		})

		It("should default to none for unknown units", func() {
			item := templates.CustomMetricItem{Metric: "foo_bar"}
			filled := fillMissingUnit(item)
			Expect(filled.Unit).To(Equal("none"))
		})

		It("should not override existing unit", func() {
			item := templates.CustomMetricItem{Metric: "foo_bar", Unit: "custom"}
			filled := fillMissingUnit(item)
			Expect(filled.Unit).To(Equal("custom"))
		})
	})

	Describe("Scaffold", func() {
		Context("when initializing a project with grafana plugin", func() {
			It("should scaffold the default grafana manifests", func() {
				err := scaffolder.Scaffold()
				Expect(err).NotTo(HaveOccurred())

				By("creating the controller-runtime metrics dashboard")
				runtimePath := filepath.Join("grafana", "controller-runtime-metrics.json")
				Expect(fileExists(runtimePath)).To(BeTrue())
				content, err := os.ReadFile(runtimePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).To(ContainSubstring("controller_runtime"))

				By("creating the controller resources metrics dashboard")
				resourcesPath := filepath.Join("grafana", "controller-resources-metrics.json")
				Expect(fileExists(resourcesPath)).To(BeTrue())

				By("creating the custom metrics config template")
				configPath := configFilePath
				Expect(fileExists(configPath)).To(BeTrue())
				content, err = os.ReadFile(configPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).To(ContainSubstring("customMetrics:"))
				Expect(string(content)).To(ContainSubstring("# Example:"))
			})
		})

		Context("when editing a project with custom metrics", func() {
			BeforeEach(func() {
				// First scaffold to create initial structure
				err := scaffolder.Scaffold()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should generate custom metrics dashboard for counter and histogram of same metric", func() {
				By("updating the config with counter and histogram for same metric name")
				configContent := `---
customMetrics:
  - metric: foo_bar
    type: counter
  - metric: foo_bar
    type: histogram
`
				configPath := configFilePath
				err := os.WriteFile(configPath, []byte(configContent), 0o644)
				Expect(err).NotTo(HaveOccurred())

				By("running scaffold again to generate the custom metrics dashboard")
				scaffolder2 := &editScaffolder{}
				scaffolder2.InjectFS(fs)
				err = scaffolder2.Scaffold()
				Expect(err).NotTo(HaveOccurred())

				By("verifying the custom metrics dashboard was created")
				dashPath := filepath.Join("grafana", "custom-metrics", "custom-metrics-dashboard.json")
				Expect(fileExists(dashPath)).To(BeTrue())

				By("verifying the dashboard contains the counter expression")
				content, err := os.ReadFile(dashPath)
				Expect(err).NotTo(HaveOccurred())
				contentStr := string(content)
				expectedCounter := `sum(rate(foo_bar{job=\"$job\", namespace=\"$namespace\"}[5m])) by (instance, pod)`
				Expect(contentStr).To(ContainSubstring(expectedCounter),
					"Dashboard should contain counter expression")

				By("verifying the dashboard contains the histogram expression")
				expectedHistogram := `histogram_quantile(0.90, sum by(instance, le) ` +
					`(rate(foo_bar{job=\"$job\", namespace=\"$namespace\"}[5m])))`
				Expect(contentStr).To(ContainSubstring(expectedHistogram),
					"Dashboard should contain histogram expression")
			})

			It("should generate dashboard with multiple different metrics", func() {
				By("configuring multiple different metrics")
				configContent := `---
customMetrics:
  - metric: http_requests_total
    type: counter
  - metric: memory_usage_bytes
    type: gauge
  - metric: request_duration_seconds
    type: histogram
`
				configPath := configFilePath
				err := os.WriteFile(configPath, []byte(configContent), 0o644)
				Expect(err).NotTo(HaveOccurred())

				By("generating the dashboard")
				scaffolder2 := &editScaffolder{}
				scaffolder2.InjectFS(fs)
				err = scaffolder2.Scaffold()
				Expect(err).NotTo(HaveOccurred())

				By("verifying all metrics are present in the dashboard")
				dashPath := filepath.Join("grafana", "custom-metrics", "custom-metrics-dashboard.json")
				content, err := os.ReadFile(dashPath)
				Expect(err).NotTo(HaveOccurred())
				contentStr := string(content)

				Expect(contentStr).To(ContainSubstring("http_requests_total"))
				Expect(contentStr).To(ContainSubstring("memory_usage_bytes"))
				Expect(contentStr).To(ContainSubstring("request_duration_seconds"))
				Expect(contentStr).To(ContainSubstring("histogram_quantile"))
			})

			It("should handle metrics with custom expressions", func() {
				By("configuring metrics with custom expressions")
				configContent := `---
customMetrics:
  - metric: custom_metric
    type: counter
    expr: 'my_custom_expression{label="value"}'
    unit: custom_unit
`
				configPath := configFilePath
				err := os.WriteFile(configPath, []byte(configContent), 0o644)
				Expect(err).NotTo(HaveOccurred())

				By("generating the dashboard")
				scaffolder2 := &editScaffolder{}
				scaffolder2.InjectFS(fs)
				err = scaffolder2.Scaffold()
				Expect(err).NotTo(HaveOccurred())

				By("verifying the custom expression is used")
				dashPath := filepath.Join("grafana", "custom-metrics", "custom-metrics-dashboard.json")
				content, err := os.ReadFile(dashPath)
				Expect(err).NotTo(HaveOccurred())
				contentStr := string(content)

				Expect(contentStr).To(ContainSubstring(`my_custom_expression{label="value"}`))
				Expect(contentStr).To(ContainSubstring("custom_unit"))
			})

			It("should auto-detect units from metric names", func() {
				By("configuring metrics with unit-indicating names")
				configContent := `---
customMetrics:
  - metric: response_time_seconds
    type: gauge
  - metric: memory_bytes
    type: gauge
  - metric: error_ratio
    type: gauge
`
				configPath := configFilePath
				err := os.WriteFile(configPath, []byte(configContent), 0o644)
				Expect(err).NotTo(HaveOccurred())

				By("generating the dashboard")
				scaffolder2 := &editScaffolder{}
				scaffolder2.InjectFS(fs)
				err = scaffolder2.Scaffold()
				Expect(err).NotTo(HaveOccurred())

				By("verifying units were auto-detected")
				dashPath := filepath.Join("grafana", "custom-metrics", "custom-metrics-dashboard.json")
				content, err := os.ReadFile(dashPath)
				Expect(err).NotTo(HaveOccurred())
				contentStr := string(content)

				// The dashboard should contain the auto-detected units
				Expect(contentStr).To(ContainSubstring(`"unit": "s"`))       // seconds
				Expect(contentStr).To(ContainSubstring(`"unit": "bytes"`))   // bytes
				Expect(contentStr).To(ContainSubstring(`"unit": "percent"`)) // percent
			})

			It("should skip invalid metrics and only process valid ones", func() {
				By("configuring a mix of valid and invalid metrics")
				configContent := `---
customMetrics:
  - metric: valid_counter
    type: counter
  - metric: ""
    type: gauge
  - type: histogram
  - metric: another_valid
    type: gauge
  - metric: invalid_type
    type: unknown
`
				configPath := configFilePath
				err := os.WriteFile(configPath, []byte(configContent), 0o644)
				Expect(err).NotTo(HaveOccurred())

				By("generating the dashboard")
				scaffolder2 := &editScaffolder{}
				scaffolder2.InjectFS(fs)
				err = scaffolder2.Scaffold()
				Expect(err).NotTo(HaveOccurred())

				By("verifying dashboard was created with only valid metrics")
				dashPath := filepath.Join("grafana", "custom-metrics", "custom-metrics-dashboard.json")
				if fileExists(dashPath) {
					content, err := os.ReadFile(dashPath)
					Expect(err).NotTo(HaveOccurred())
					contentStr := string(content)

					Expect(contentStr).To(ContainSubstring("valid_counter"))
					Expect(contentStr).To(ContainSubstring("another_valid"))
					Expect(contentStr).NotTo(ContainSubstring("invalid_type"))
				}
			})

			It("should handle metrics ending with _info suffix specially", func() {
				By("configuring a metric with _info suffix")
				configContent := `---
customMetrics:
  - metric: build_info
    type: gauge
`
				configPath := configFilePath
				err := os.WriteFile(configPath, []byte(configContent), 0o644)
				Expect(err).NotTo(HaveOccurred())

				By("generating the dashboard")
				scaffolder2 := &editScaffolder{}
				scaffolder2.InjectFS(fs)
				err = scaffolder2.Scaffold()
				Expect(err).NotTo(HaveOccurred())

				By("verifying dashboard was created with _info metric using table visualization")
				dashPath := filepath.Join("grafana", "custom-metrics", "custom-metrics-dashboard.json")
				if fileExists(dashPath) {
					content, err := os.ReadFile(dashPath)
					Expect(err).NotTo(HaveOccurred())
					contentStr := string(content)

					Expect(contentStr).To(ContainSubstring("build_info"))
					Expect(contentStr).To(ContainSubstring(`"type": "table"`))
				}
			})
		})

		Context("when no custom metrics are configured", func() {
			It("should not create custom metrics dashboard", func() {
				By("scaffolding with default config")
				err := scaffolder.Scaffold()
				Expect(err).NotTo(HaveOccurred())

				By("verifying custom metrics dashboard was not created")
				dashPath := filepath.Join("grafana", "custom-metrics", "custom-metrics-dashboard.json")
				Expect(fileExists(dashPath)).To(BeFalse())

				By("verifying the config template was created")
				configPath := configFilePath
				Expect(fileExists(configPath)).To(BeTrue())
			})
		})

		Context("when config file has only empty metrics", func() {
			It("should not create custom metrics dashboard", func() {
				By("creating empty config")
				configContent := `---
customMetrics: []
`
				err := os.MkdirAll(filepath.Join("grafana", "custom-metrics"), 0o755)
				Expect(err).NotTo(HaveOccurred())

				configPath := configFilePath
				err = os.WriteFile(configPath, []byte(configContent), 0o644)
				Expect(err).NotTo(HaveOccurred())

				By("scaffolding")
				err = scaffolder.Scaffold()
				Expect(err).NotTo(HaveOccurred())

				By("verifying custom metrics dashboard was not created")
				dashPath := filepath.Join("grafana", "custom-metrics", "custom-metrics-dashboard.json")
				Expect(fileExists(dashPath)).To(BeFalse())
			})
		})
	})
})

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
