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

package kustomize

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ = Describe("ChartConverter", func() {
	var (
		converter *ChartConverter
		resources *ParsedResources
		fs        machinery.Filesystem
	)

	BeforeEach(func() {
		// Create test resources
		resources = &ParsedResources{}

		// Add a test deployment
		deployment := &unstructured.Unstructured{}
		deployment.SetAPIVersion("apps/v1")
		deployment.SetKind("Deployment")
		deployment.SetName("test-controller")
		deployment.SetNamespace("test-system")

		// Set deployment spec
		err := unstructured.SetNestedField(deployment.Object, int64(1), "spec", "replicas")
		Expect(err).NotTo(HaveOccurred())

		resources.Deployment = deployment

		// Create filesystem
		fs = machinery.Filesystem{FS: afero.NewMemMapFs()}

		// Create converter
		converter = NewChartConverter(resources, "test-project", "dist")
	})

	Context("NewChartConverter", func() {
		It("should create a converter with correct properties", func() {
			Expect(converter.resources).To(Equal(resources))
			Expect(converter.projectName).To(Equal("test-project"))
			Expect(converter.outputDir).To(Equal("dist"))
		})
	})

	Context("WriteChartFiles", func() {
		It("should write chart files to filesystem", func() {
			// Add some resources to test with
			serviceAccount := &unstructured.Unstructured{}
			serviceAccount.SetAPIVersion("v1")
			serviceAccount.SetKind("ServiceAccount")
			serviceAccount.SetName("test-sa")
			serviceAccount.SetNamespace("test-system")
			resources.ServiceAccount = serviceAccount

			// Add RBAC resources to test rbac directory creation
			clusterRole := &unstructured.Unstructured{}
			clusterRole.SetAPIVersion("rbac.authorization.k8s.io/v1")
			clusterRole.SetKind("ClusterRole")
			clusterRole.SetName("test-role")
			resources.ClusterRoles = []*unstructured.Unstructured{clusterRole}

			err := converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			exists, err := afero.Exists(fs.FS, "dist/chart/templates/manager")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			exists, err = afero.Exists(fs.FS, "dist/chart/templates/rbac")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("should deduplicate identical resources within a group", func() {
			// Prepare two identical Services in the metrics group
			metricsSvc1 := &unstructured.Unstructured{}
			metricsSvc1.SetAPIVersion("v1")
			metricsSvc1.SetKind("Service")
			metricsSvc1.SetName("test-project-controller-manager-metrics-service")
			metricsSvc1.SetNamespace("test-system")

			metricsSvc2 := &unstructured.Unstructured{}
			metricsSvc2.SetAPIVersion("v1")
			metricsSvc2.SetKind("Service")
			metricsSvc2.SetName("test-project-controller-manager-metrics-service")
			metricsSvc2.SetNamespace("test-system")

			// Add both to resources; organizer will place them into the metrics group
			resources.Services = append(resources.Services, metricsSvc1, metricsSvc2)

			// Write chart files
			err := converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			// Expect only one file to be written for the metrics service after de-duplication
			metricsDir := filepath.Join("dist", "chart", "templates", "metrics")
			files, err := afero.ReadDir(fs.FS, metricsDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(HaveLen(1), "expected only one metrics service file after deduplication")
		})
	})

	Context("ExtractDeploymentConfig", func() {
		It("should extract deployment configuration correctly", func() {
			// Set up deployment with environment variables
			containers := []interface{}{
				map[string]interface{}{
					"name":  "manager",
					"image": "controller:latest",
					"env": []interface{}{
						map[string]interface{}{
							"name":  "TEST_ENV",
							"value": "test-value",
						},
					},
					"resources": map[string]interface{}{
						"limits": map[string]interface{}{
							"cpu":    "100m",
							"memory": "128Mi",
						},
					},
				},
			}

			err := unstructured.SetNestedSlice(
				resources.Deployment.Object,
				containers,
				"spec", "template", "spec", "containers",
			)
			Expect(err).NotTo(HaveOccurred())

			config := converter.ExtractDeploymentConfig()

			Expect(config).NotTo(BeNil())
			Expect(config).To(HaveKey("env"))
			Expect(config).To(HaveKey("resources"))

			// Verify image extraction
			Expect(config).To(HaveKey("image"))
			imageConfig := config["image"].(map[string]interface{})
			Expect(imageConfig["repository"]).To(Equal("controller"))
			Expect(imageConfig["tag"]).To(Equal("latest"))
		})

		It("should handle deployment without containers", func() {
			config := converter.ExtractDeploymentConfig()
			Expect(config).To(BeEmpty())
		})
	})
})
