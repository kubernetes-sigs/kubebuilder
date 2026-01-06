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

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/kustomize"
)

var _ = Describe("Extras Directory Integration Test", func() {
	var (
		fs     machinery.Filesystem
		tmpDir string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "helm-extras-test-*")
		Expect(err).NotTo(HaveOccurred())

		// Change to tmpDir so relative paths work correctly
		err = os.Chdir(tmpDir)
		Expect(err).NotTo(HaveOccurred())

		fs = machinery.Filesystem{
			FS: afero.NewBasePathFs(afero.NewOsFs(), tmpDir),
		}
	})

	AfterEach(func() {
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
	})

	Context("when converting Kustomize output with extra resources", func() {
		It("should place ConfigMap in extras directory with proper labels", func() {
			// Create a simulated kustomize output with standard resources and a ConfigMap
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
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: test-project-controller-manager
  namespace: test-project-system
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
---
apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: custom-config
  namespace: test-project-system
data:
  key1: value1
  key2: value2
---
apiVersion: v1
kind: Secret
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: custom-secret
  namespace: test-project-system
type: Opaque
data:
  password: c2VjcmV0Cg==
`

			By("writing kustomize output to a file")
			kustomizeFile := filepath.Join(tmpDir, "install.yaml")
			err := os.WriteFile(kustomizeFile, []byte(kustomizeYAML), 0o600)
			Expect(err).NotTo(HaveOccurred())

			By("parsing the kustomize output")
			parser := kustomize.NewParser(kustomizeFile)
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).NotTo(BeNil())

			By("verifying ConfigMap and Secret are in Other category")
			Expect(resources.Other).To(HaveLen(2))

			By("converting to Helm chart")
			converter := kustomize.NewChartConverter(resources, "test-project", "test-project", "dist")
			err = converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			By("verifying extras directory was created")
			extrasDir := filepath.Join("dist", "chart", "templates", "extras")
			exists, err := afero.Exists(fs.FS, extrasDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "extras directory should exist")

			By("verifying extras directory contains the ConfigMap and Secret")
			files, err := afero.ReadDir(fs.FS, extrasDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(HaveLen(2), "extras should contain ConfigMap and Secret")

			var configMapFile, secretFile string
			for _, f := range files {
				if f.Name() == "custom-config.yaml" {
					configMapFile = f.Name()
				}
				if f.Name() == "custom-secret.yaml" {
					secretFile = f.Name()
				}
			}
			Expect(configMapFile).NotTo(BeEmpty(), "ConfigMap file should exist")
			Expect(secretFile).NotTo(BeEmpty(), "Secret file should exist")

			By("verifying ConfigMap has proper Helm templating")
			configMapPath := filepath.Join(extrasDir, configMapFile)
			content, err := afero.ReadFile(fs.FS, configMapPath)
			Expect(err).NotTo(HaveOccurred())
			configMapContent := string(content)

			// Verify namespace templating
			Expect(configMapContent).To(ContainSubstring("namespace: {{ .Release.Namespace }}"),
				"ConfigMap should have templated namespace")

			// Note: Resource names without the project prefix are kept as-is
			// This allows users to have custom resource names that don't follow the project naming convention
			Expect(configMapContent).To(ContainSubstring("name: custom-config"),
				"ConfigMap name should be preserved as-is when it doesn't match project prefix")

			// Verify standard Helm labels
			Expect(configMapContent).To(ContainSubstring(`app.kubernetes.io/name: {{ include "test-project.name" . }}`),
				"ConfigMap should have app.kubernetes.io/name label")
			Expect(configMapContent).To(ContainSubstring("app.kubernetes.io/instance: {{ .Release.Name }}"),
				"ConfigMap should have app.kubernetes.io/instance label")
			Expect(configMapContent).To(ContainSubstring("app.kubernetes.io/managed-by: {{ .Release.Service }}"),
				"ConfigMap should have app.kubernetes.io/managed-by label")
			Expect(configMapContent).To(ContainSubstring(`helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}`),
				"ConfigMap should have helm.sh/chart label")

			// Verify data is preserved
			Expect(configMapContent).To(ContainSubstring("key1: value1"),
				"ConfigMap data should be preserved")
			Expect(configMapContent).To(ContainSubstring("key2: value2"),
				"ConfigMap data should be preserved")

			By("verifying Secret has proper Helm templating")
			secretPath := filepath.Join(extrasDir, secretFile)
			content, err = afero.ReadFile(fs.FS, secretPath)
			Expect(err).NotTo(HaveOccurred())
			secretContent := string(content)

			// Verify namespace templating
			Expect(secretContent).To(ContainSubstring("namespace: {{ .Release.Namespace }}"),
				"Secret should have templated namespace")

			// Note: Resource names without the project prefix are kept as-is
			Expect(secretContent).To(ContainSubstring("name: custom-secret"),
				"Secret name should be preserved as-is when it doesn't match project prefix")

			// Verify standard Helm labels
			Expect(secretContent).To(ContainSubstring(`app.kubernetes.io/name: {{ include "test-project.name" . }}`),
				"Secret should have app.kubernetes.io/name label")
			Expect(secretContent).To(ContainSubstring("app.kubernetes.io/managed-by: {{ .Release.Service }}"),
				"Secret should have app.kubernetes.io/managed-by label")

			// Verify data is preserved
			Expect(secretContent).To(ContainSubstring("password: c2VjcmV0Cg=="),
				"Secret data should be preserved")
		})

		It("should place custom Service in extras directory", func() {
			kustomizeYAML := `---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: test-project-system
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: custom-service
  namespace: test-project-system
spec:
  ports:
  - port: 8080
    targetPort: 8080
  selector:
    app: custom
`

			kustomizeFile := filepath.Join(tmpDir, "install.yaml")
			err := os.WriteFile(kustomizeFile, []byte(kustomizeYAML), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser := kustomize.NewParser(kustomizeFile)
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())

			converter := kustomize.NewChartConverter(resources, "test-project", "test-project", "dist")
			err = converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			By("verifying custom Service is in extras directory")
			extrasDir := filepath.Join("dist", "chart", "templates", "extras")
			exists, err := afero.Exists(fs.FS, extrasDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			files, err := afero.ReadDir(fs.FS, extrasDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(HaveLen(1))
			Expect(files[0].Name()).To(Equal("custom-service.yaml"))

			By("verifying Service has proper Helm templating")
			servicePath := filepath.Join(extrasDir, files[0].Name())
			content, err := afero.ReadFile(fs.FS, servicePath)
			Expect(err).NotTo(HaveOccurred())
			serviceContent := string(content)

			Expect(serviceContent).To(ContainSubstring("namespace: {{ .Release.Namespace }}"))
			Expect(serviceContent).To(ContainSubstring(`app.kubernetes.io/name: {{ include "test-project.name" . }}`))
		})

		It("should not place webhook or metrics services in extras", func() {
			kustomizeYAML := `---
apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: test-project-webhook-service
  namespace: test-project-system
spec:
  ports:
  - port: 443
    targetPort: 9443
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
  name: test-project-controller-manager-metrics-service
  namespace: test-project-system
spec:
  ports:
  - port: 8443
    targetPort: 8443
`

			kustomizeFile := filepath.Join(tmpDir, "install.yaml")
			err := os.WriteFile(kustomizeFile, []byte(kustomizeYAML), 0o600)
			Expect(err).NotTo(HaveOccurred())

			parser := kustomize.NewParser(kustomizeFile)
			resources, err := parser.Parse()
			Expect(err).NotTo(HaveOccurred())

			converter := kustomize.NewChartConverter(resources, "test-project", "test-project", "dist")
			err = converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			By("verifying extras directory was NOT created")
			extrasDir := filepath.Join("dist", "chart", "templates", "extras")
			exists, err := afero.Exists(fs.FS, extrasDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse(), "extras directory should not exist for webhook/metrics services")

			By("verifying webhook directory was created")
			webhookDir := filepath.Join("dist", "chart", "templates", "webhook")
			exists, err = afero.Exists(fs.FS, webhookDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			By("verifying metrics directory was created")
			metricsDir := filepath.Join("dist", "chart", "templates", "metrics")
			exists, err = afero.Exists(fs.FS, metricsDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})
	})
})
