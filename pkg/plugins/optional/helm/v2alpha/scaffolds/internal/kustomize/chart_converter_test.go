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
		converter = NewChartConverter(resources, "test-project", "test-project", "dist")
	})

	Context("NewChartConverter", func() {
		It("should create a converter with correct properties", func() {
			Expect(converter.resources).To(Equal(resources))
			Expect(converter.detectedPrefix).To(Equal("test-project"))
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
			containers := []any{
				map[string]any{
					"name":            "manager",
					"image":           "controller:latest",
					"imagePullPolicy": "IfNotPresent",
					"args": []any{
						"--metrics-bind-address=:8443",
						"--leader-elect",
						"--custom-flag=value",
						"--health-probe-bind-address=:8081",
						"--webhook-cert-path=/tmp/k8s-webhook-server/serving-certs",
					},
					"env": []any{
						map[string]any{
							"name":  "TEST_ENV",
							"value": "test-value",
						},
					},
					"resources": map[string]any{
						"limits": map[string]any{
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
			Expect(config).To(HaveKey("image"))
			Expect(config).To(HaveKey("resources"))
			Expect(config).To(HaveKey("args"))

			imageConfig, ok := config["image"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(imageConfig["repository"]).To(Equal("controller"))
			Expect(imageConfig["tag"]).To(Equal("latest"))
			Expect(imageConfig["pullPolicy"]).To(Equal("IfNotPresent"))

			args, ok := config["args"].([]any)
			Expect(ok).To(BeTrue())
			Expect(args).To(ContainElement("--leader-elect"))
			Expect(args).To(ContainElement("--custom-flag=value"))
			Expect(args).NotTo(ContainElement("--metrics-bind-address=:8443"))
			Expect(args).NotTo(ContainElement("--health-probe-bind-address=:8081"))
		})

		It("should extract port configurations from args", func() {
			// Set up deployment with port-related args
			containers := []any{
				map[string]any{
					"name":  "manager",
					"image": "controller:latest",
					"args": []any{
						"--metrics-bind-address=:8443",
						"--health-probe-bind-address=:8081",
						"--leader-elect",
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

			Expect(config).To(HaveKey("metricsPort"))
			Expect(config["metricsPort"]).To(Equal(8443))
			Expect(config).NotTo(HaveKey("healthPort"))
		})

		It("should extract webhook port from container ports", func() {
			// Set up deployment with webhook container port
			containers := []any{
				map[string]any{
					"name":  "manager",
					"image": "controller:latest",
					"ports": []any{
						map[string]any{
							"containerPort": int64(9443),
							"name":          "webhook-server",
							"protocol":      "TCP",
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

			Expect(config).To(HaveKey("webhookPort"))
			Expect(config["webhookPort"]).To(Equal(9443))
		})

		It("should extract custom port values", func() {
			// Set up deployment with custom ports
			containers := []any{
				map[string]any{
					"name":  "manager",
					"image": "controller:latest",
					"args": []any{
						"--metrics-bind-address=:9090",
						"--health-probe-bind-address=:9091",
					},
					"ports": []any{
						map[string]any{
							"containerPort": int64(9444),
							"name":          "webhook-server",
							"protocol":      "TCP",
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

			Expect(config["metricsPort"]).To(Equal(9090))
			Expect(config["healthPort"]).To(BeNil())
			Expect(config["webhookPort"]).To(Equal(9444))
		})

		It("should extract imagePullSecrets", func() {
			// Set up deployment with image pull secrets
			containers := []any{
				map[string]any{
					"name":  "manager",
					"image": "controller:latest",
				},
			}
			imagePullSecrets := []any{
				map[string]any{
					"name": "test-secret",
				},
			}
			// Set the image pull secrets
			err := unstructured.SetNestedSlice(
				resources.Deployment.Object,
				imagePullSecrets,
				"spec", "template", "spec", "imagePullSecrets",
			)
			Expect(err).NotTo(HaveOccurred())
			// Set the containers
			err = unstructured.SetNestedSlice(
				resources.Deployment.Object,
				containers,
				"spec", "template", "spec", "containers",
			)
			Expect(err).NotTo(HaveOccurred())

			config := converter.ExtractDeploymentConfig()
			Expect(config).To(HaveKey("imagePullSecrets"))
			Expect(config["imagePullSecrets"]).To(Equal(imagePullSecrets))
		})

		It("should handle deployment without containers", func() {
			config := converter.ExtractDeploymentConfig()
			Expect(config).To(BeEmpty())
		})
	})

	Context("extractPortFromArg", func() {
		It("should extract port from :PORT format", func() {
			port := extractPortFromArg("--metrics-bind-address=:8443")
			Expect(port).To(Equal(8443))
		})

		It("should extract port from 0.0.0.0:PORT format", func() {
			port := extractPortFromArg("--metrics-bind-address=0.0.0.0:8443")
			Expect(port).To(Equal(8443))
		})

		It("should extract port from HOST:PORT format", func() {
			port := extractPortFromArg("--health-probe-bind-address=localhost:8081")
			Expect(port).To(Equal(8081))
		})

		It("should return 0 for invalid formats", func() {
			port := extractPortFromArg("--invalid-arg")
			Expect(port).To(Equal(0))

			port = extractPortFromArg("--no-equals:8443")
			Expect(port).To(Equal(0))

			port = extractPortFromArg("--port=invalid")
			Expect(port).To(Equal(0))
		})

		It("should return 0 for out-of-range ports", func() {
			port := extractPortFromArg("--port=:0")
			Expect(port).To(Equal(0))

			port = extractPortFromArg("--port=:99999")
			Expect(port).To(Equal(0))
		})
	})

	Context("Extras Directory", func() {
		It("should place ConfigMap in extras directory", func() {
			// Create a ConfigMap that doesn't fit standard categories
			configMap := &unstructured.Unstructured{}
			configMap.SetAPIVersion("v1")
			configMap.SetKind("ConfigMap")
			configMap.SetName("custom-config")
			configMap.SetNamespace("test-project-system")
			configMap.Object["metadata"] = map[string]any{
				"name":      "custom-config",
				"namespace": "test-project-system",
				"labels": map[string]any{
					"app.kubernetes.io/name":       "test-project",
					"app.kubernetes.io/managed-by": "kustomize",
				},
			}
			configMap.Object["data"] = map[string]any{
				"key": "value",
			}

			resources.Other = []*unstructured.Unstructured{configMap}

			err := converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			// Verify extras directory was created
			exists, err := afero.Exists(fs.FS, "dist/chart/templates/extras")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			// Verify ConfigMap file was created
			files, err := afero.ReadDir(fs.FS, "dist/chart/templates/extras")
			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(HaveLen(1))
			Expect(files[0].Name()).To(ContainSubstring("custom-config"))

			// Read the ConfigMap file and verify it has Helm templating
			content, err := afero.ReadFile(fs.FS, filepath.Join("dist/chart/templates/extras", files[0].Name()))
			Expect(err).NotTo(HaveOccurred())
			contentStr := string(content)

			// Verify Helm templates are applied
			Expect(contentStr).To(ContainSubstring("{{ .Release.Namespace }}"))
			Expect(contentStr).To(ContainSubstring("app.kubernetes.io/name:"))
			Expect(contentStr).To(ContainSubstring("app.kubernetes.io/managed-by:"))
		})

		It("should place custom Service in extras directory", func() {
			// Create a custom Service that is neither webhook nor metrics
			customService := &unstructured.Unstructured{}
			customService.SetAPIVersion("v1")
			customService.SetKind("Service")
			customService.SetName("custom-service")
			customService.SetNamespace("test-project-system")
			customService.Object["metadata"] = map[string]any{
				"name":      "custom-service",
				"namespace": "test-project-system",
				"labels": map[string]any{
					"app.kubernetes.io/name":       "test-project",
					"app.kubernetes.io/managed-by": "kustomize",
				},
			}
			customService.Object["spec"] = map[string]any{
				"ports": []any{
					map[string]any{
						"port":       8080,
						"targetPort": 8080,
					},
				},
			}

			resources.Services = []*unstructured.Unstructured{customService}

			err := converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			// Verify extras directory was created
			exists, err := afero.Exists(fs.FS, "dist/chart/templates/extras")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			// Verify Service file was created in extras
			files, err := afero.ReadDir(fs.FS, "dist/chart/templates/extras")
			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(HaveLen(1))
			Expect(files[0].Name()).To(ContainSubstring("custom-service"))
		})

		It("should place Secret in extras directory", func() {
			// Create a Secret
			secret := &unstructured.Unstructured{}
			secret.SetAPIVersion("v1")
			secret.SetKind("Secret")
			secret.SetName("custom-secret")
			secret.SetNamespace("test-project-system")
			secret.Object["metadata"] = map[string]any{
				"name":      "custom-secret",
				"namespace": "test-project-system",
				"labels": map[string]any{
					"app.kubernetes.io/name":       "test-project",
					"app.kubernetes.io/managed-by": "kustomize",
				},
			}
			secret.Object["data"] = map[string]any{
				"password": "c2VjcmV0",
			}

			resources.Other = []*unstructured.Unstructured{secret}

			err := converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			// Verify extras directory was created
			exists, err := afero.Exists(fs.FS, "dist/chart/templates/extras")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			// Verify Secret file was created
			files, err := afero.ReadDir(fs.FS, "dist/chart/templates/extras")
			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(HaveLen(1))
			Expect(files[0].Name()).To(ContainSubstring("custom-secret"))

			// Read the Secret file and verify it has Helm templating
			content, err := afero.ReadFile(fs.FS, filepath.Join("dist/chart/templates/extras", files[0].Name()))
			Expect(err).NotTo(HaveOccurred())
			contentStr := string(content)

			// Verify Helm templates are applied
			Expect(contentStr).To(ContainSubstring("{{ .Release.Namespace }}"))
		})

		It("should handle multiple extras resources", func() {
			// Create multiple extras resources
			configMap := &unstructured.Unstructured{}
			configMap.SetAPIVersion("v1")
			configMap.SetKind("ConfigMap")
			configMap.SetName("config1")
			configMap.SetNamespace("test-project-system")
			configMap.Object["metadata"] = map[string]any{
				"name":      "config1",
				"namespace": "test-project-system",
			}

			secret := &unstructured.Unstructured{}
			secret.SetAPIVersion("v1")
			secret.SetKind("Secret")
			secret.SetName("secret1")
			secret.SetNamespace("test-project-system")
			secret.Object["metadata"] = map[string]any{
				"name":      "secret1",
				"namespace": "test-project-system",
			}

			customService := &unstructured.Unstructured{}
			customService.SetAPIVersion("v1")
			customService.SetKind("Service")
			customService.SetName("custom-svc")
			customService.SetNamespace("test-project-system")
			customService.Object["metadata"] = map[string]any{
				"name":      "custom-svc",
				"namespace": "test-project-system",
			}

			resources.Other = []*unstructured.Unstructured{configMap, secret}
			resources.Services = []*unstructured.Unstructured{customService}

			err := converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			// Verify all three files were created
			files, err := afero.ReadDir(fs.FS, "dist/chart/templates/extras")
			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(HaveLen(3))
		})

		It("should apply standard Helm labels to extras resources", func() {
			// Create a ConfigMap
			configMap := &unstructured.Unstructured{}
			configMap.SetAPIVersion("v1")
			configMap.SetKind("ConfigMap")
			configMap.SetName("test-config")
			configMap.SetNamespace("test-system")
			configMap.Object["metadata"] = map[string]any{
				"name":      "test-config",
				"namespace": "test-system",
				"labels": map[string]any{
					"app.kubernetes.io/name":       "test-project",
					"app.kubernetes.io/managed-by": "kustomize",
				},
			}

			resources.Other = []*unstructured.Unstructured{configMap}

			err := converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			// Read the ConfigMap file
			files, err := afero.ReadDir(fs.FS, "dist/chart/templates/extras")
			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(HaveLen(1))

			content, err := afero.ReadFile(fs.FS, filepath.Join("dist/chart/templates/extras", files[0].Name()))
			Expect(err).NotTo(HaveOccurred())
			contentStr := string(content)

			// Verify all standard Helm labels are present
			Expect(contentStr).To(ContainSubstring("app.kubernetes.io/name: {{ include \"test-project.name\" . }}"))
			Expect(contentStr).To(ContainSubstring("app.kubernetes.io/instance: {{ .Release.Name }}"))
			Expect(contentStr).To(ContainSubstring("app.kubernetes.io/managed-by: {{ .Release.Service }}"))
			Expect(contentStr).To(ContainSubstring(
				`helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}`))
		})

		It("should not place webhook or metrics services in extras", func() {
			// Create webhook service
			webhookService := &unstructured.Unstructured{}
			webhookService.SetAPIVersion("v1")
			webhookService.SetKind("Service")
			webhookService.SetName("test-project-webhook-service")
			webhookService.SetNamespace("test-project-system")
			webhookService.Object["metadata"] = map[string]any{
				"name":      "test-project-webhook-service",
				"namespace": "test-project-system",
			}

			// Create metrics service
			metricsService := &unstructured.Unstructured{}
			metricsService.SetAPIVersion("v1")
			metricsService.SetKind("Service")
			metricsService.SetName("test-project-controller-manager-metrics-service")
			metricsService.SetNamespace("test-project-system")
			metricsService.Object["metadata"] = map[string]any{
				"name":      "test-project-controller-manager-metrics-service",
				"namespace": "test-project-system",
			}

			resources.Services = []*unstructured.Unstructured{webhookService, metricsService}

			err := converter.WriteChartFiles(fs)
			Expect(err).NotTo(HaveOccurred())

			// Verify extras directory was not created (webhook/metrics go to their own dirs)
			exists, err := afero.Exists(fs.FS, "dist/chart/templates/extras")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())

			// Verify webhook directory was created
			exists, err = afero.Exists(fs.FS, "dist/chart/templates/webhook")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			// Verify metrics directory was created
			exists, err = afero.Exists(fs.FS, "dist/chart/templates/metrics")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})
	})
})
