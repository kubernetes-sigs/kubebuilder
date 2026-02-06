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

var _ = Describe("Chart.yaml Never Overwrite Test", func() {
	var (
		fs            machinery.Filesystem
		tmpDir        string
		manifestsFile string
		outputDir     string
		projectConfig config.Config
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "helm-chart-never-overwrite-*")
		Expect(err).NotTo(HaveOccurred())

		err = os.Chdir(tmpDir)
		Expect(err).NotTo(HaveOccurred())

		fs = machinery.Filesystem{
			FS: afero.NewBasePathFs(afero.NewOsFs(), tmpDir),
		}

		projectConfig = cfgv3.New()
		projectConfig.SetProjectName("test-project")
		projectConfig.SetDomain("example.io")

		manifestsFile = filepath.Join(tmpDir, "dist", "install.yaml")
		outputDir = "dist"

		// Create minimal kustomize output
		kustomizeYAML := `---
apiVersion: v1
kind: Namespace
metadata:
  name: test-project-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
  namespace: test-project-system
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest
`

		err = os.MkdirAll(filepath.Dir(manifestsFile), 0o755)
		Expect(err).NotTo(HaveOccurred())
		err = os.WriteFile(manifestsFile, []byte(kustomizeYAML), 0o644)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
	})

	It("should NEVER overwrite Chart.yaml even with --force=true", func() {
		scaffolder := &editKustomizeScaffolder{
			config:        projectConfig,
			fs:            fs,
			force:         false,
			manifestsFile: manifestsFile,
			outputDir:     outputDir,
		}

		// First scaffold
		err := scaffolder.Scaffold()
		Expect(err).NotTo(HaveOccurred())

		chartPath := filepath.Join(tmpDir, outputDir, "chart", "Chart.yaml")

		// Read initial Chart.yaml
		initialContent, err := os.ReadFile(chartPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(initialContent)).To(ContainSubstring("name: test-project"))
		Expect(string(initialContent)).To(ContainSubstring("version: 0.1.0"))

		// Customize Chart.yaml with user version
		customChartYAML := `apiVersion: v2
name: test-project
description: My custom description
type: application
version: 1.2.3
appVersion: "1.2.3"
icon: "https://mycompany.com/icon.png"
maintainers:
  - name: John Doe
    email: john@example.com
`
		err = os.WriteFile(chartPath, []byte(customChartYAML), 0o644)
		Expect(err).NotTo(HaveOccurred())

		// Scaffold again WITHOUT force
		err = scaffolder.Scaffold()
		Expect(err).NotTo(HaveOccurred())

		// Verify Chart.yaml was NOT overwritten
		content, err := os.ReadFile(chartPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal(customChartYAML), "Chart.yaml should not be overwritten without --force")
		Expect(string(content)).To(ContainSubstring("version: 1.2.3"))
		Expect(string(content)).To(ContainSubstring("John Doe"))

		// Scaffold again WITH force=true
		scaffolder.force = true
		err = scaffolder.Scaffold()
		Expect(err).NotTo(HaveOccurred())

		// Verify Chart.yaml STILL was NOT overwritten (never overwritten even with force)
		content, err = os.ReadFile(chartPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal(customChartYAML), "Chart.yaml should NEVER be overwritten, even with --force")
		Expect(string(content)).To(ContainSubstring("version: 1.2.3"))
		Expect(string(content)).To(ContainSubstring("John Doe"))
		Expect(string(content)).To(ContainSubstring("My custom description"))
	})

	It("should preserve Chart.yaml on initial scaffold if file already exists", func() {
		// Create Chart.yaml before any scaffolding
		chartPath := filepath.Join(tmpDir, outputDir, "chart", "Chart.yaml")
		err := os.MkdirAll(filepath.Dir(chartPath), 0o755)
		Expect(err).NotTo(HaveOccurred())

		preexistingChart := `apiVersion: v2
name: my-existing-chart
version: 99.99.99
`
		err = os.WriteFile(chartPath, []byte(preexistingChart), 0o644)
		Expect(err).NotTo(HaveOccurred())

		// First scaffold with force=true
		scaffolder := &editKustomizeScaffolder{
			config:        projectConfig,
			fs:            fs,
			force:         true,
			manifestsFile: manifestsFile,
			outputDir:     outputDir,
		}

		err = scaffolder.Scaffold()
		Expect(err).NotTo(HaveOccurred())

		// Verify Chart.yaml was preserved even on first scaffold with force
		content, err := os.ReadFile(chartPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal(preexistingChart), "Chart.yaml should be preserved even on first scaffold with --force")
		Expect(string(content)).To(ContainSubstring("my-existing-chart"))
		Expect(string(content)).To(ContainSubstring("99.99.99"))
	})
})
