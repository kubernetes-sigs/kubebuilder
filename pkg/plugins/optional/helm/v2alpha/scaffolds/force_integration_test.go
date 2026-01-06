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

package scaffolds

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ = Describe("Force Flag Integration Test", func() {
	var (
		fs             machinery.Filesystem
		tmpDir         string
		manifestsFile  string
		outputDir      string
		projectConfig  config.Config
		scaffolderBase *editKustomizeScaffolder
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "helm-force-test-*")
		Expect(err).NotTo(HaveOccurred())

		err = os.Chdir(tmpDir)
		Expect(err).NotTo(HaveOccurred())

		fs = machinery.Filesystem{
			FS: afero.NewBasePathFs(afero.NewOsFs(), tmpDir),
		}

		// Create PROJECT file
		projectConfig = cfgv3.New()
		projectConfig.SetProjectName("test-project")
		projectConfig.SetDomain("example.io")

		// Setup directories - use absolute path for manifestsFile since parser uses os.Open
		manifestsFile = filepath.Join(tmpDir, "dist", "install.yaml")
		outputDir = "dist"

		// Create minimal kustomize output file using real OS filesystem (parser uses os.Open)
		kustomizeYAML := `---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
    control-plane: controller-manager
  name: test-project-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
    control-plane: controller-manager
  name: test-project-controller-manager
  namespace: test-project-system
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - name: manager
        image: controller:latest
        imagePullPolicy: IfNotPresent
        command:
        - /manager
        args:
        - --leader-elect
        - --health-probe-bind-address=:8081
        ports:
        - containerPort: 9443
          name: webhook-server
          protocol: TCP
        env:
        - name: TEST_ENV
          value: "test-value"
        resources:
          limits:
            cpu: 500m
            memory: 128Mi
          requests:
            cpu: 10m
            memory: 64Mi
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
`

		// Use OS filesystem to write manifests file since Parser uses os.Open
		err = os.MkdirAll(filepath.Dir(manifestsFile), 0o755)
		Expect(err).NotTo(HaveOccurred())
		err = os.WriteFile(manifestsFile, []byte(kustomizeYAML), 0o644)
		Expect(err).NotTo(HaveOccurred())

		scaffolderBase = &editKustomizeScaffolder{
			config:        projectConfig,
			fs:            fs,
			manifestsFile: manifestsFile,
			outputDir:     outputDir,
		}
	})

	AfterEach(func() {
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
	})

	Context("when --force flag is NOT used", func() {
		It("should NOT overwrite existing Chart.yaml, values.yaml, .helmignore, _helpers.tpl, and test-chart.yml", func() {
			// First generation with force=false
			scaffolderBase.force = false
			err := scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Define file paths (absolute paths for OS filesystem)
			chartPath := filepath.Join(tmpDir, outputDir, "chart", "Chart.yaml")
			valuesPath := filepath.Join(tmpDir, outputDir, "chart", "values.yaml")
			helmignorePath := filepath.Join(tmpDir, outputDir, "chart", ".helmignore")
			helpersPath := filepath.Join(tmpDir, outputDir, "chart", "templates", "_helpers.tpl")
			testChartPath := filepath.Join(tmpDir, ".github", "workflows", "test-chart.yml")

			// Verify files exist
			_, err = os.ReadFile(chartPath)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.ReadFile(valuesPath)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.ReadFile(helmignorePath)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.ReadFile(helpersPath)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.ReadFile(testChartPath)
			Expect(err).NotTo(HaveOccurred())

			// Modify all protected files
			customChartContent := "# CUSTOM CHART YAML\nversion: 999.0.0\n"
			customValuesContent := "# CUSTOM VALUES YAML\ncustom: value\n"
			customHelmignoreContent := "# CUSTOM HELMIGNORE\n*.custom\n"
			customHelpersContent := "# CUSTOM HELPERS TPL\n{{/* custom helper */}}\n"
			customTestChartContent := "# CUSTOM TEST CHART\nname: Custom Workflow\n"

			err = os.WriteFile(chartPath, []byte(customChartContent), 0o644)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(valuesPath, []byte(customValuesContent), 0o644)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(helmignorePath, []byte(customHelmignoreContent), 0o644)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(helpersPath, []byte(customHelpersContent), 0o644)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(testChartPath, []byte(customTestChartContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			// Second generation with force=false
			err = scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify all protected files were NOT overwritten
			chartContent, err := os.ReadFile(chartPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(chartContent)).To(Equal(customChartContent), "Chart.yaml should not be overwritten without --force")

			valuesContent, err := os.ReadFile(valuesPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(valuesContent)).To(Equal(customValuesContent), "values.yaml should not be overwritten without --force")

			helmignoreContent, err := os.ReadFile(helmignorePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(helmignoreContent)).To(Equal(customHelmignoreContent), ".helmignore should not be overwritten without --force")

			helpersContent, err := os.ReadFile(helpersPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(helpersContent)).To(Equal(customHelpersContent), "_helpers.tpl should not be overwritten without --force")

			testChartContent, err := os.ReadFile(testChartPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(testChartContent)).To(Equal(customTestChartContent), "test-chart.yml should not be overwritten without --force")
		})
	})

	Context("when --force flag IS used", func() {
		It("should overwrite all files EXCEPT Chart.yaml (which is never overwritten)", func() {
			// First generation with force=false
			scaffolderBase.force = false
			err := scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Define file paths (absolute paths for OS filesystem)
			chartPath := filepath.Join(tmpDir, outputDir, "chart", "Chart.yaml")
			valuesPath := filepath.Join(tmpDir, outputDir, "chart", "values.yaml")
			helmignorePath := filepath.Join(tmpDir, outputDir, "chart", ".helmignore")
			helpersPath := filepath.Join(tmpDir, outputDir, "chart", "templates", "_helpers.tpl")
			testChartPath := filepath.Join(tmpDir, ".github", "workflows", "test-chart.yml")

			// Modify all protected files with custom content
			customChartContent := "# CUSTOM CHART YAML\nversion: 999.0.0\n"
			customValuesContent := "# CUSTOM VALUES YAML\ncustom: value\n"
			customHelmignoreContent := "# CUSTOM HELMIGNORE\n*.custom\n"
			customHelpersContent := "# CUSTOM HELPERS TPL\n{{/* custom helper */}}\n"
			customTestChartContent := "# CUSTOM TEST CHART\nname: Custom Workflow\n"

			err = os.WriteFile(chartPath, []byte(customChartContent), 0o644)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(valuesPath, []byte(customValuesContent), 0o644)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(helmignorePath, []byte(customHelmignoreContent), 0o644)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(helpersPath, []byte(customHelpersContent), 0o644)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(testChartPath, []byte(customTestChartContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			// Second generation with force=true
			scaffolderBase.force = true
			err = scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify Chart.yaml was NOT overwritten (never overwritten, even with --force)
			chartContent, err := os.ReadFile(chartPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(chartContent)).To(Equal(customChartContent), "Chart.yaml should NEVER be overwritten, even with --force")

			valuesContent, err := os.ReadFile(valuesPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(valuesContent)).NotTo(Equal(customValuesContent), "values.yaml should be overwritten with --force")
			Expect(string(valuesContent)).To(ContainSubstring("manager:"), "values.yaml should contain manager section")

			helmignoreContent, err := os.ReadFile(helmignorePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(helmignoreContent)).NotTo(Equal(customHelmignoreContent), ".helmignore should be overwritten with --force")
			Expect(string(helmignoreContent)).To(ContainSubstring(".DS_Store"), ".helmignore should contain default patterns")

			helpersContent, err := os.ReadFile(helpersPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(helpersContent)).NotTo(Equal(customHelpersContent), "_helpers.tpl should be overwritten with --force")
			Expect(string(helpersContent)).To(ContainSubstring("test-project.name"), "_helpers.tpl should contain template helpers")

			testChartContent, err := os.ReadFile(testChartPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(testChartContent)).NotTo(Equal(customTestChartContent), "test-chart.yml should be overwritten with --force")
			Expect(string(testChartContent)).To(ContainSubstring("Test Chart"), "test-chart.yml should contain workflow name")
		})

		It("should overwrite files on first run when force=true", func() {
			// Create pre-existing custom files before any scaffold run (absolute paths)
			chartPath := filepath.Join(tmpDir, outputDir, "chart", "Chart.yaml")
			valuesPath := filepath.Join(tmpDir, outputDir, "chart", "values.yaml")

			err := os.MkdirAll(filepath.Dir(chartPath), 0o755)
			Expect(err).NotTo(HaveOccurred())

			customChartContent := "# PRE-EXISTING CHART\nversion: 0.0.1\n"
			customValuesContent := "# PRE-EXISTING VALUES\nold: data\n"

			err = os.WriteFile(chartPath, []byte(customChartContent), 0o644)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(valuesPath, []byte(customValuesContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			// First generation with force=true should overwrite
			scaffolderBase.force = true
			err = scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify Chart.yaml was NOT overwritten (never overwritten, even on first run with force)
			chartContent, err := os.ReadFile(chartPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(chartContent)).To(Equal(customChartContent), "Chart.yaml should NEVER be overwritten, even on first run with --force")

			// Verify values.yaml WAS overwritten
			valuesContent, err := os.ReadFile(valuesPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(valuesContent)).NotTo(Equal(customValuesContent), "values.yaml should be overwritten on first run with --force")
			Expect(string(valuesContent)).To(ContainSubstring("manager:"))
		})
	})

	Context("when template files are modified", func() {
		It("should verify template files exist in templates/ directory", func() {
			// First generation
			scaffolderBase.force = false
			err := scaffolderBase.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			// Verify that template directory was created with files
			templatesDir := filepath.Join(tmpDir, outputDir, "chart", "templates")
			info, err := os.Stat(templatesDir)
			Expect(err).NotTo(HaveOccurred(), "Templates directory should be created")
			Expect(info.IsDir()).To(BeTrue(), "Templates should be a directory")

			// At minimum, _helpers.tpl should exist
			helpersPath := filepath.Join(templatesDir, "_helpers.tpl")
			_, err = os.Stat(helpersPath)
			Expect(err).NotTo(HaveOccurred(), "_helpers.tpl should exist in templates/")
		})
	})
})

