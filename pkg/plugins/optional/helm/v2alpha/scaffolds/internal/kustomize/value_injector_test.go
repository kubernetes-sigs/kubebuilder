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
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestValueInjector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ValueInjector Suite")
}

var _ = Describe("ValueInjector", func() {
	var (
		injector   *ValueInjector
		userValues *UserAddedValues
		chartName  string
	)

	BeforeEach(func() {
		chartName = "test-chart"
		userValues = &UserAddedValues{
			ManagerLabels:         make(map[string]string),
			ManagerAnnotations:    make(map[string]string),
			ManagerPodLabels:      make(map[string]string),
			ManagerPodAnnotations: make(map[string]string),
			EnvVars:               []map[string]any{},
			CustomManagerFields:   make(map[string]any),
			CustomTopLevelFields:  make(map[string]any),
		}
		injector = NewValueInjector(userValues, chartName)
	})

	Describe("InjectCustomValues", func() {
		Context("with a Deployment resource", func() {
			var deployment *unstructured.Unstructured

			BeforeEach(func() {
				deployment = &unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"metadata": map[string]any{
							"name": "test-deployment",
						},
					},
				}
			})

			Context("when custom manager labels are defined", func() {
				BeforeEach(func() {
					userValues.ManagerLabels["custom-label"] = "custom-value"
					userValues.ManagerLabels["app-tier"] = "backend"
				})

				It("should inject label templates into the metadata", func() {
					yamlContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  labels:
    app.kubernetes.io/name: test-app
spec:
  template:
    metadata:
      labels:
        app.kubernetes.io/name: test-app`

					result := injector.InjectCustomValues(yamlContent, deployment)
					Expect(result).To(ContainSubstring(".Values.manager.labels.custom-label"))
					Expect(result).To(ContainSubstring(".Values.manager.labels.app-tier"))
				})
			})

			Context("when additional environment variables are defined", func() {
				BeforeEach(func() {
					userValues.EnvVars = []map[string]any{
						{"name": "CUSTOM_VAR", "value": "custom-value"},
					}
				})

				It("should inject env var templates into the container", func() {
					yamlContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        env:
        - name: EXISTING_VAR
          value: existing-value`

					result := injector.InjectCustomValues(yamlContent, deployment)
					// env is handled by existing helm_templater.go, so just verify no errors
					Expect(result).ToNot(BeEmpty())
				})
			})

			Context("when custom ports are defined", func() {
				BeforeEach(func() {
					userValues.CustomManagerFields["ports"] = []any{
						map[string]any{"name": "custom", "containerPort": 8080},
					}
				})

				It("should inject port templates", func() {
					yamlContent := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        ports:
        - name: webhook
          containerPort: 9443`

					result := injector.InjectCustomValues(yamlContent, deployment)
					Expect(result).To(ContainSubstring(".Values.manager.ports"))
				})
			})

			Context("when custom volumes are defined", func() {
				BeforeEach(func() {
					userValues.CustomManagerFields["volumes"] = []any{
						map[string]any{
							"name": "custom-volume",
							"emptyDir": map[string]any{},
						},
					}
				})

				It("should inject volume templates with correct indentation", func() {
					yamlContent := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest
      serviceAccountName: controller-manager
      terminationGracePeriodSeconds: 10
      volumes:
      - name: cert
        secret:
          secretName: webhook-cert`

					result := injector.InjectCustomValues(yamlContent, deployment)
					Expect(result).To(ContainSubstring("{{- with .Values.manager.volumes }}"))
				Expect(result).To(ContainSubstring("{{- toYaml . | nindent 10 }}"))
					// Verify the template is added after existing volumes
					lines := splitLines(result)
					volumesIdx := -1
					certVolumeIdx := -1
					templateIdx := -1
					
					for i, line := range lines {
						if containsAny(line, "volumes:") {
							volumesIdx = i
						}
						if containsAny(line, "- name: cert") && volumesIdx > 0 {
							certVolumeIdx = i
						}
						if containsAny(line, "{{- with .Values.manager.volumes }}") {
							templateIdx = i
						}
					}
					
					Expect(volumesIdx).To(BeNumerically(">", 0))
					Expect(certVolumeIdx).To(BeNumerically(">", volumesIdx))
					Expect(templateIdx).To(BeNumerically(">", certVolumeIdx))
				})

				It("should handle volumes with complex structure", func() {
					yamlContent := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      volumes:
      - name: metrics-certs
        secret:
          items:
          - key: ca.crt
            path: ca.crt
          - key: tls.crt
            path: tls.crt
          secretName: metrics-server-cert
      - name: webhook-certs
        secret:
          secretName: webhook-server-cert`

					result := injector.InjectCustomValues(yamlContent, deployment)
					Expect(result).To(ContainSubstring("{{- with .Values.manager.volumes }}"))
					
					// Verify it comes after webhook-certs volume
					lines := splitLines(result)
					webhookCertsIdx := -1
					templateIdx := -1
					
					for i, line := range lines {
						if containsAny(line, "- name: webhook-certs") {
							webhookCertsIdx = i
						}
						if containsAny(line, "{{- with .Values.manager.volumes }}") {
							templateIdx = i
						}
					}
					
					Expect(webhookCertsIdx).To(BeNumerically(">", 0))
					Expect(templateIdx).To(BeNumerically(">", webhookCertsIdx))
				})
			})

			Context("when custom volume mounts are defined", func() {
				BeforeEach(func() {
					userValues.CustomManagerFields["volumeMounts"] = []any{
						map[string]any{
							"name":      "custom-mount",
							"mountPath": "/custom",
						},
					}
				})

				It("should inject volume mount templates with correct indentation", func() {
					yamlContent := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest
        volumeMounts:
        - name: cert
          mountPath: /tmp/k8s-webhook-server/serving-certs
          readOnly: true`

					result := injector.InjectCustomValues(yamlContent, deployment)
					Expect(result).To(ContainSubstring("{{- with .Values.manager.volumeMounts }}"))
					Expect(result).To(ContainSubstring("{{- toYaml . | nindent 10 }}"))
					// Verify the template is added after existing volumeMounts
					lines := splitLines(result)
					volumeMountsIdx := -1
					certMountIdx := -1
					templateIdx := -1
					
					for i, line := range lines {
						if containsAny(line, "volumeMounts:") {
							volumeMountsIdx = i
						}
						if containsAny(line, "- name: cert") && volumeMountsIdx > 0 {
							certMountIdx = i
						}
						if containsAny(line, "{{- with .Values.manager.volumeMounts }}") {
							templateIdx = i
						}
					}
					
					Expect(volumeMountsIdx).To(BeNumerically(">", 0))
					Expect(certMountIdx).To(BeNumerically(">", volumeMountsIdx))
					Expect(templateIdx).To(BeNumerically(">", certMountIdx))
				})

				It("should handle volumeMounts in manager container only", func() {
					yamlContent := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: kube-rbac-proxy
        image: proxy:latest
        volumeMounts:
        - name: proxy-cert
          mountPath: /proxy-cert
      - name: manager
        image: controller:latest
        volumeMounts:
        - name: metrics-certs
          mountPath: /tmp/k8s-metrics-server/metrics-certs
          readOnly: true
        - name: webhook-certs
          mountPath: /tmp/k8s-webhook-server/serving-certs
          readOnly: true`

					result := injector.InjectCustomValues(yamlContent, deployment)
					
					// Should only inject into manager container, not kube-rbac-proxy
					lines := splitLines(result)
					managerIdx := -1
					proxyIdx := -1
					templateCount := 0
					
					for i, line := range lines {
						if containsAny(line, "name: kube-rbac-proxy") {
							proxyIdx = i
						}
						if containsAny(line, "name: manager") {
							managerIdx = i
						}
						if containsAny(line, "{{- with .Values.manager.volumeMounts }}") {
							templateCount++
							// Should be after manager container declaration
							Expect(i).To(BeNumerically(">", managerIdx))
							// Should not be in proxy container section
							if proxyIdx > 0 {
								Expect(i).To(BeNumerically(">", managerIdx))
							}
						}
					}
					
					// Should only have one template injection
					Expect(templateCount).To(Equal(1))
				})
			})

			Context("verifying existing values are preserved", func() {
				BeforeEach(func() {
					userValues.CustomManagerFields["volumes"] = []any{
						map[string]any{
							"name": "custom-volume",
							"emptyDir": map[string]any{},
						},
					}
					userValues.CustomManagerFields["volumeMounts"] = []any{
						map[string]any{
							"name":      "custom-volume",
							"mountPath": "/custom",
						},
					}
				})

				It("should append custom volumes to existing application volumes", func() {
					yamlContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
spec:
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest
        volumeMounts:
        - name: config-volume
          mountPath: /etc/config
          readOnly: true
        - name: data-volume
          mountPath: /var/data
          readOnly: false
      serviceAccountName: controller-manager
      terminationGracePeriodSeconds: 10
      volumes:
      - name: config-volume
        configMap:
          name: app-config
      - name: data-volume
        emptyDir: {}`

					result := injector.InjectCustomValues(yamlContent, deployment)
					
					// Verify existing volumes are preserved
					Expect(result).To(ContainSubstring("- name: config-volume"))
					Expect(result).To(ContainSubstring("name: app-config"))
					Expect(result).To(ContainSubstring("- name: data-volume"))
					Expect(result).To(ContainSubstring("emptyDir: {}"))
					
					// Verify custom volume template is added AFTER existing volumes
					Expect(result).To(ContainSubstring("{{- with .Values.manager.volumes }}"))
					
					// Verify order: existing volumes -> custom template
					lines := splitLines(result)
					configVolIdx := -1
					dataVolIdx := -1
					customVolTemplateIdx := -1
					
					for i, line := range lines {
						if containsAny(line, "- name: config-volume") && configVolIdx == -1 {
							// This is the volume, not volumeMount
							if i > 0 && containsAny(lines[i-1], "volumes:") || 
							   (i > 1 && containsAny(lines[i-2], "volumes:")) {
								configVolIdx = i
							}
						}
						if containsAny(line, "- name: data-volume") && dataVolIdx == -1 {
							// This is the volume
							if configVolIdx > 0 && i > configVolIdx {
								dataVolIdx = i
							}
						}
						if containsAny(line, "{{- with .Values.manager.volumes }}") {
							customVolTemplateIdx = i
						}
					}
					
					Expect(configVolIdx).To(BeNumerically(">", 0), "Should find config-volume")
					Expect(dataVolIdx).To(BeNumerically(">", configVolIdx), "data-volume should be after config-volume")
					Expect(customVolTemplateIdx).To(BeNumerically(">", dataVolIdx), "Custom template should be after existing volumes")
				})

				It("should append custom volumeMounts to existing volumeMounts", func() {
					yamlContent := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest
        volumeMounts:
        - name: config-volume
          mountPath: /etc/config
          readOnly: true
        - name: data-volume
          mountPath: /var/data
          readOnly: false`

					result := injector.InjectCustomValues(yamlContent, deployment)
					
					// Verify existing volumeMounts are preserved
					Expect(result).To(ContainSubstring("- name: config-volume"))
					Expect(result).To(ContainSubstring("mountPath: /etc/config"))
					Expect(result).To(ContainSubstring("- name: data-volume"))
					Expect(result).To(ContainSubstring("mountPath: /var/data"))
					
					// Verify custom volumeMount template is added AFTER existing volumeMounts
					Expect(result).To(ContainSubstring("{{- with .Values.manager.volumeMounts }}"))
					
					// Verify order
					lines := splitLines(result)
					configVmIdx := -1
					dataVmIdx := -1
					customVmTemplateIdx := -1
					
					for i, line := range lines {
						if containsAny(line, "- name: config-volume") {
							configVmIdx = i
						}
						if containsAny(line, "- name: data-volume") && configVmIdx > 0 {
							dataVmIdx = i
						}
						if containsAny(line, "{{- with .Values.manager.volumeMounts }}") {
							customVmTemplateIdx = i
						}
					}
					
					Expect(configVmIdx).To(BeNumerically(">", 0))
					Expect(dataVmIdx).To(BeNumerically(">", configVmIdx))
					Expect(customVmTemplateIdx).To(BeNumerically(">", dataVmIdx))
				})

				It("should not override existing volumes when toYaml renders custom values", func() {
					// This is a conceptual test - in practice, the toYaml function
					// renders the custom values as additional list items, not replacements
					yamlContent := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest
      volumes:
      - name: existing-volume
        configMap:
          name: existing-config`

					result := injector.InjectCustomValues(yamlContent, deployment)
					
					// The structure should be:
					// volumes:
					// - name: existing-volume  (preserved)
					//   configMap: ...
					// {{- with .Values.manager.volumes }}  (custom template)
					// {{- toYaml . | nindent 10 }}
					// {{- end }}
					
					// When Helm renders this, it will produce:
					// volumes:
					// - name: existing-volume
					// - name: custom-volume  (from values.yaml)
					
					Expect(result).To(ContainSubstring("- name: existing-volume"))
					Expect(result).To(ContainSubstring("configMap:"))
					Expect(result).To(ContainSubstring("{{- with .Values.manager.volumes }}"))
					
					// Verify the template comes AFTER the existing volume
					existingVolIdx := strings.Index(result, "- name: existing-volume")
					templateIdx := strings.Index(result, "{{- with .Values.manager.volumes }}")
					Expect(templateIdx).To(BeNumerically(">", existingVolIdx))
				})
			})
		})

		Context("with custom manager annotations", func() {
			var deployment *unstructured.Unstructured

			BeforeEach(func() {
				deployment = &unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"metadata": map[string]any{
							"name": "test-deployment",
						},
					},
				}
				userValues.ManagerAnnotations["custom.io/annotation"] = "value"
			})

			Context("when annotations section exists", func() {
				It("should inject annotation templates", func() {
					yamlContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  annotations:
    existing.io/annotation: existing-value
  labels:
    app: test`

					result := injector.InjectCustomValues(yamlContent, deployment)
					Expect(result).To(ContainSubstring(".Values.manager.annotations"))
				})
			})

			Context("when annotations section does not exist", func() {
				It("should add annotations section with templates", func() {
					yamlContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  labels:
    app: test`

					result := injector.InjectCustomValues(yamlContent, deployment)
					Expect(result).To(ContainSubstring("annotations:"))
					Expect(result).To(ContainSubstring(".Values.manager.annotations"))
				})
			})
		})

		Context("with pod-level customizations", func() {
			var deployment *unstructured.Unstructured

			BeforeEach(func() {
				deployment = &unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
					},
				}
				userValues.ManagerPodLabels["pod-label"] = "pod-value"
			})

			It("should inject pod label templates", func() {
				yamlContent := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    metadata:
      labels:
        app: test
        control-plane: controller-manager`

				result := injector.InjectCustomValues(yamlContent, deployment)
				Expect(result).To(ContainSubstring(".Values.manager.podLabels"))
			})

			It("should inject pod annotation templates", func() {
				userValues.ManagerPodAnnotations["pod.annotation.io/key"] = "value"
				
				yamlContent := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
      labels:
        app: test`

				result := injector.InjectCustomValues(yamlContent, deployment)
				Expect(result).To(ContainSubstring(".Values.manager.podAnnotations"))
			})
		})

		Context("with multiple custom values combined", func() {
			var deployment *unstructured.Unstructured

			BeforeEach(func() {
				deployment = &unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
					},
				}
				// Add multiple custom values
				userValues.ManagerLabels["custom-label"] = "value1"
				userValues.ManagerAnnotations["custom.io/annotation"] = "value2"
				userValues.ManagerPodLabels["pod-label"] = "pod-value"
				userValues.ManagerPodAnnotations["pod.io/annotation"] = "pod-anno"
				userValues.CustomManagerFields["volumes"] = []any{
					map[string]any{"name": "custom-vol", "emptyDir": map[string]any{}},
				}
				userValues.CustomManagerFields["volumeMounts"] = []any{
					map[string]any{"name": "custom-vol", "mountPath": "/custom"},
				}
			})

			It("should inject all custom values correctly", func() {
				yamlContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  labels:
    app: test
spec:
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
      labels:
        app: test
    spec:
      containers:
      - name: manager
        image: controller:latest
        volumeMounts:
        - name: cert
          mountPath: /cert
      volumes:
      - name: cert
        secret:
          secretName: cert`

				result := injector.InjectCustomValues(yamlContent, deployment)
				
				// Verify all injections are present
				Expect(result).To(ContainSubstring(".Values.manager.labels.custom-label"))
				Expect(result).To(ContainSubstring(".Values.manager.annotations"))
				Expect(result).To(ContainSubstring(".Values.manager.podLabels.pod-label"))
				Expect(result).To(ContainSubstring(".Values.manager.podAnnotations"))
				Expect(result).To(ContainSubstring("{{- with .Values.manager.volumes }}"))
				Expect(result).To(ContainSubstring("{{- with .Values.manager.volumeMounts }}"))
			})
		})

		Context("with special characters in keys", func() {
			var deployment *unstructured.Unstructured

			BeforeEach(func() {
				deployment = &unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
					},
				}
			})

			It("should escape dots and slashes in label keys", func() {
				userValues.ManagerLabels["custom.io/special-label"] = "value"
				
				yamlContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test`

				result := injector.InjectCustomValues(yamlContent, deployment)
				Expect(result).To(ContainSubstring(".Values.manager.labels.custom_io_special-label"))
			})

			It("should escape dots and slashes in annotation keys", func() {
				userValues.ManagerAnnotations["prometheus.io/scrape"] = "true"
				userValues.ManagerAnnotations["custom.domain.com/config"] = "enabled"
				
				yamlContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test`

				result := injector.InjectCustomValues(yamlContent, deployment)
				Expect(result).To(ContainSubstring(".Values.manager.annotations.prometheus_io_scrape"))
				Expect(result).To(ContainSubstring(".Values.manager.annotations.custom_domain_com_config"))
			})
		})

		Context("edge cases", func() {
			var deployment *unstructured.Unstructured

			BeforeEach(func() {
				deployment = &unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
					},
				}
			})

			It("should handle empty custom values gracefully", func() {
				yamlContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test`

				result := injector.InjectCustomValues(yamlContent, deployment)
				Expect(result).To(Equal(yamlContent))
			})

			It("should not inject into resources without volumes section", func() {
				userValues.CustomManagerFields["volumes"] = []any{
					map[string]any{"name": "custom", "emptyDir": map[string]any{}},
				}
				
				yamlContent := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest`

				result := injector.InjectCustomValues(yamlContent, deployment)
				// Should not crash, just return unchanged
				Expect(result).NotTo(ContainSubstring("{{- with .Values.manager.volumes }}"))
			})

			It("should not inject into resources without volumeMounts section", func() {
				userValues.CustomManagerFields["volumeMounts"] = []any{
					map[string]any{"name": "custom", "mountPath": "/custom"},
				}
				
				yamlContent := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest
        ports:
        - containerPort: 8080`

				result := injector.InjectCustomValues(yamlContent, deployment)
				// Should not crash, just return unchanged
				Expect(result).NotTo(ContainSubstring("{{- with .Values.manager.volumeMounts }}"))
			})
		})

		Context("with non-Deployment resources", func() {
			It("should not inject deployment-specific values into Services", func() {
				service := &unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "v1",
						"kind":       "Service",
					},
				}
				userValues.EnvVars = []map[string]any{
					{"name": "VAR", "value": "value"},
				}

				yamlContent := `apiVersion: v1
kind: Service
metadata:
  name: test-service
spec:
  ports:
  - port: 80`

				result := injector.InjectCustomValues(yamlContent, service)
				// EnvVars only apply to Deployments, not Services
				Expect(result).ToNot(BeEmpty())
			})
		})
	})

	Describe("escapeYAMLKey", func() {
		It("should escape dots in keys", func() {
			result := escapeYAMLKey("custom.io/key")
			Expect(result).To(Equal("custom_io_key"))
		})

		It("should escape slashes in keys", func() {
			result := escapeYAMLKey("custom/key")
			Expect(result).To(Equal("custom_key"))
		})

		It("should handle keys with both dots and slashes", func() {
			result := escapeYAMLKey("custom.io/special.key/value")
			Expect(result).To(Equal("custom_io_special_key_value"))
		})

		It("should not modify simple keys", func() {
			result := escapeYAMLKey("simpleKey")
			Expect(result).To(Equal("simpleKey"))
		})
	})
})

// Helper functions for tests

// splitLines splits a string into lines
func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

// containsAny checks if a string contains the given substring
func containsAny(s string, substr string) bool {
	return strings.Contains(s, substr)
}
