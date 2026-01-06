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

package templates

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ = Describe("HelmValuesBasic", func() {
	var valuesTemplate *HelmValuesBasic

	Context("when project has webhooks", func() {
		BeforeEach(func() {
			valuesTemplate = &HelmValuesBasic{
				HasWebhooks:      true,
				DeploymentConfig: map[string]any{},
			}
			valuesTemplate.InjectProjectName("test-project")
			err := valuesTemplate.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should include certManager configuration", func() {
			content := valuesTemplate.GetBody()

			Expect(content).To(ContainSubstring("certManager:"))
			Expect(content).To(ContainSubstring("enable: true"))
		})

		It("should include all basic sections", func() {
			content := valuesTemplate.GetBody()

			Expect(content).To(ContainSubstring("manager:"))
			Expect(content).To(ContainSubstring("args: []"))
			Expect(content).To(ContainSubstring("env: []"))
			Expect(content).To(ContainSubstring("metrics:"))
			Expect(content).To(ContainSubstring("prometheus:"))
			Expect(content).To(ContainSubstring("rbacHelpers:"))
			Expect(content).To(ContainSubstring("imagePullSecrets: []"))
		})
	})

	Context("when project has no webhooks", func() {
		BeforeEach(func() {
			valuesTemplate = &HelmValuesBasic{
				HasWebhooks:      false,
				DeploymentConfig: map[string]any{},
			}
			valuesTemplate.InjectProjectName("test-project")
			err := valuesTemplate.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not include certManager configuration", func() {
			content := valuesTemplate.GetBody()

			Expect(content).To(ContainSubstring("certManager:"))
			Expect(content).To(ContainSubstring("enable: false"))
		})

		It("should still include other basic sections", func() {
			content := valuesTemplate.GetBody()

			Expect(content).To(ContainSubstring("manager:"))
			Expect(content).To(ContainSubstring("args: []"))
			Expect(content).To(ContainSubstring("metrics:"))
			Expect(content).To(ContainSubstring("prometheus:"))
			Expect(content).To(ContainSubstring("rbacHelpers:"))
			Expect(content).To(ContainSubstring("imagePullSecrets: []"))
		})
	})

	Context("template path and content", func() {
		BeforeEach(func() {
			valuesTemplate = &HelmValuesBasic{
				OutputDir: "dist",
			}
			valuesTemplate.InjectProjectName("test-project")
			err := valuesTemplate.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should have correct path", func() {
			Expect(valuesTemplate.GetPath()).To(Equal("dist/chart/values.yaml"))
		})

		It("should implement Builder interface", func() {
			var builder machinery.Builder = valuesTemplate
			Expect(builder).NotTo(BeNil())
		})

		It("should have correct file permissions", func() {
			info := valuesTemplate.GetIfExistsAction()
			Expect(info).To(Equal(machinery.SkipFile))
		})
	})

	Context("with deployment configuration", func() {
		BeforeEach(func() {
			deploymentConfig := map[string]any{
				"args": []any{
					"--leader-elect",
				},
				"env": []any{
					map[string]any{
						"name":  "TEST_ENV",
						"value": "test-value",
					},
				},
				"image": map[string]any{
					"repository": "example.com/custom-controller",
					"tag":        "v1.2.3",
					"pullPolicy": "Always",
				},
				"resources": map[string]any{
					"limits": map[string]any{
						"cpu":    "100m",
						"memory": "128Mi",
					},
				},
			}

			valuesTemplate = &HelmValuesBasic{
				HasWebhooks:      false,
				DeploymentConfig: deploymentConfig,
			}
			valuesTemplate.InjectProjectName("test-project")
			err := valuesTemplate.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should include deployment configuration", func() {
			content := valuesTemplate.GetBody()
			Expect(content).To(ContainSubstring("args:"))
			Expect(content).To(ContainSubstring("- --leader-elect"))
			Expect(content).To(ContainSubstring("env:"))
			Expect(content).To(ContainSubstring("name: TEST_ENV"))
			Expect(content).To(ContainSubstring("value: test-value"))
			Expect(content).To(ContainSubstring("repository: example.com/custom-controller"))
			Expect(content).To(ContainSubstring("tag: v1.2.3"))
			Expect(content).To(ContainSubstring("pullPolicy: Always"))
			Expect(content).To(ContainSubstring("resources:"))
			Expect(content).To(ContainSubstring("cpu: 100m"))
			Expect(content).To(ContainSubstring("memory: 128Mi"))
			Expect(content).To(ContainSubstring("affinity: {}"))
			Expect(content).To(ContainSubstring("nodeSelector: {}"))
			Expect(content).To(ContainSubstring("tolerations: []"))
		})
	})

	Context("with nodeSelector, affinity and tolerations configuration", func() {
		BeforeEach(func() {
			deploymentConfig := map[string]any{
				"podNodeSelector": map[string]string{
					"kubernetes.io/os": "linux",
				},
				"podTolerations": []map[string]string{
					{
						"key":      "key1",
						"operator": "Equal",
						"effect":   "NoSchedule",
					},
				},
				"podAffinity": map[string]any{
					"nodeAffinity": map[string]any{
						"requiredDuringSchedulingIgnoredDuringExecution": map[string]any{
							"nodeSelectorTerms": []any{
								map[string]any{
									"matchExpressions": []any{
										map[string]any{
											"key":      "topology.kubernetes.io/zone",
											"operator": "In",
											"values":   []string{"antarctica-east1", "antarctica-east2"},
										},
									},
								},
							},
						},
					},
				},
			}

			valuesTemplate = &HelmValuesBasic{
				HasWebhooks:      false,
				DeploymentConfig: deploymentConfig,
			}
			valuesTemplate.InjectProjectName("test-project")
			err := valuesTemplate.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should include default values", func() {
			content := valuesTemplate.GetBody()
			Expect(content).To(ContainSubstring(`  ## Manager pod's node selector
  ##
  nodeSelector:
    kubernetes.io/os: linux`))

			Expect(content).To(ContainSubstring(`  ## Manager pod's tolerations
  ##
  tolerations:
    - effect: NoSchedule
      key: key1
      operator: Equal`))

			Expect(content).To(ContainSubstring(`  ## Manager pod's affinity
  ##
  affinity:
    nodeAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
                - matchExpressions:
                    - key: topology.kubernetes.io/zone
                      operator: In
                      values:
                        - antarctica-east1
                        - antarctica-east2`))
		})
	})

	Context("with multiple imagePullSecrets", func() {
		BeforeEach(func() {
			valuesTemplate = &HelmValuesBasic{
				DeploymentConfig: map[string]any{
					"imagePullSecrets": []any{
						map[string]any{
							"name": "test-secret",
						},
						map[string]any{
							"name": "test-secret2",
						},
					},
				},
			}
			valuesTemplate.InjectProjectName("test-private-project")
			err := valuesTemplate.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should render multiple imagePullSecrets", func() {
			content := valuesTemplate.GetBody()
			Expect(content).To(ContainSubstring("imagePullSecrets:"))
			Expect(content).To(ContainSubstring("- name: test-secret"))
			Expect(content).To(ContainSubstring("- name: test-secret2"))
		})
	})

	Context("with complex env variables", func() {
		BeforeEach(func() {
			valuesTemplate = &HelmValuesBasic{
				DeploymentConfig: map[string]any{
					"env": []any{
						map[string]any{
							"name": "POD_NAMESPACE",
							"valueFrom": map[string]any{
								"fieldRef": map[string]any{
									"fieldPath": "metadata.namespace",
								},
							},
						},
					},
				},
			}
			valuesTemplate.InjectProjectName("test-project")
			err := valuesTemplate.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should render nested env configuration", func() {
			content := valuesTemplate.GetBody()
			Expect(content).To(ContainSubstring("env:"))
			Expect(content).To(ContainSubstring("name: POD_NAMESPACE"))
			Expect(content).To(ContainSubstring("valueFrom:"))
			Expect(content).To(ContainSubstring("fieldRef:"))
			Expect(content).To(ContainSubstring("fieldPath: metadata.namespace"))
		})
	})

	Context("rbacHelpers configuration", func() {
		BeforeEach(func() {
			valuesTemplate = &HelmValuesBasic{
				HasWebhooks: false,
			}
			valuesTemplate.InjectProjectName("test-project")
			err := valuesTemplate.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should have rbacHelpers disabled by default", func() {
			content := valuesTemplate.GetBody()
			Expect(content).To(ContainSubstring("rbacHelpers:"))
			Expect(content).To(ContainSubstring("enable: false"))
		})
	})

	Context("Port configuration", func() {
		Context("with default ports", func() {
			BeforeEach(func() {
				valuesTemplate = &HelmValuesBasic{
					HasWebhooks:      true,
					HasMetrics:       true,
					DeploymentConfig: map[string]any{},
				}
				valuesTemplate.InjectProjectName("test-project")
				err := valuesTemplate.SetTemplateDefaults()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should include default webhook port", func() {
				content := valuesTemplate.GetBody()
				Expect(content).To(ContainSubstring("webhook:"))
				Expect(content).To(ContainSubstring("enable: true"))
				Expect(content).To(ContainSubstring("port: 9443"))
			})

			It("should include certManager enabled", func() {
				content := valuesTemplate.GetBody()
				Expect(content).To(ContainSubstring("certManager:"))
				Expect(content).To(ContainSubstring("enable: true"))
			})

			It("should include default metrics port", func() {
				content := valuesTemplate.GetBody()
				Expect(content).To(ContainSubstring("metrics:"))
				Expect(content).To(ContainSubstring("enable: true"))
				Expect(content).To(ContainSubstring("port: 8443"))
			})

			It("should not expose health port in values", func() {
				content := valuesTemplate.GetBody()
				Expect(content).NotTo(ContainSubstring("healthPort"))
			})
		})

		Context("with custom ports extracted from deployment", func() {
			BeforeEach(func() {
				deploymentConfig := map[string]any{
					"webhookPort": 9444,
					"metricsPort": 9090,
				}

				valuesTemplate = &HelmValuesBasic{
					HasWebhooks:      true,
					HasMetrics:       true,
					DeploymentConfig: deploymentConfig,
				}
				valuesTemplate.InjectProjectName("test-project")
				err := valuesTemplate.SetTemplateDefaults()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should use custom webhook port", func() {
				content := valuesTemplate.GetBody()
				Expect(content).To(ContainSubstring("webhook:"))
				Expect(content).To(ContainSubstring("port: 9444"))
				Expect(content).NotTo(ContainSubstring("port: 9443"))
			})

			It("should use custom metrics port", func() {
				content := valuesTemplate.GetBody()
				Expect(content).To(ContainSubstring("metrics:"))
				Expect(content).To(ContainSubstring("port: 9090"))
				Expect(content).NotTo(ContainSubstring("port: 8443"))
			})

			It("should not expose health port", func() {
				content := valuesTemplate.GetBody()
				Expect(content).NotTo(ContainSubstring("healthPort"))
			})
		})

		Context("with partial custom ports", func() {
			BeforeEach(func() {
				deploymentConfig := map[string]any{
					"metricsPort": 9090,
					// webhookPort not provided - should use default
				}

				valuesTemplate = &HelmValuesBasic{
					HasWebhooks:      true,
					HasMetrics:       true,
					DeploymentConfig: deploymentConfig,
				}
				valuesTemplate.InjectProjectName("test-project")
				err := valuesTemplate.SetTemplateDefaults()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should use custom metrics port and default webhook port", func() {
				content := valuesTemplate.GetBody()
				Expect(content).To(ContainSubstring("port: 9090")) // custom metrics
				Expect(content).To(ContainSubstring("port: 9443")) // default webhook
			})

			It("should not expose health port", func() {
				content := valuesTemplate.GetBody()
				Expect(content).NotTo(ContainSubstring("healthPort"))
			})
		})

		Context("with no webhooks but with metrics", func() {
			BeforeEach(func() {
				valuesTemplate = &HelmValuesBasic{
					HasWebhooks:      false,
					HasMetrics:       true,
					DeploymentConfig: map[string]any{},
				}
				valuesTemplate.InjectProjectName("test-project")
				err := valuesTemplate.SetTemplateDefaults()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should not include webhook configuration", func() {
				content := valuesTemplate.GetBody()
				Expect(content).NotTo(ContainSubstring("webhook:"))
			})

			It("should include metrics port configuration", func() {
				content := valuesTemplate.GetBody()
				Expect(content).To(ContainSubstring("metrics:"))
				Expect(content).To(ContainSubstring("port: 8443"))
				Expect(content).NotTo(ContainSubstring("healthPort"))
			})
		})

		Context("port configuration structure", func() {
			BeforeEach(func() {
				valuesTemplate = &HelmValuesBasic{
					HasWebhooks:      true,
					HasMetrics:       true,
					DeploymentConfig: map[string]any{},
				}
				valuesTemplate.InjectProjectName("test-project")
				err := valuesTemplate.SetTemplateDefaults()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have ports under webhook section", func() {
				content := valuesTemplate.GetBody()
				lines := strings.Split(content, "\n")

				var webhookLine, portLine int
				for i, line := range lines {
					if strings.Contains(line, "webhook:") && !strings.Contains(line, "#") {
						webhookLine = i
					}
					if webhookLine > 0 && i > webhookLine && strings.Contains(line, "port:") &&
						strings.Contains(line, "9443") {
						portLine = i
						break
					}
				}

				Expect(webhookLine).To(BeNumerically(">", 0))
				Expect(portLine).To(BeNumerically(">", webhookLine))
				Expect(portLine - webhookLine).To(BeNumerically("<=", 3))
			})

			It("should have port under metrics section", func() {
				content := valuesTemplate.GetBody()
				lines := strings.Split(content, "\n")

				var metricsLine, portLine int
				for i, line := range lines {
					if strings.Contains(line, "metrics:") && !strings.Contains(line, "#") {
						metricsLine = i
					}
					if metricsLine > 0 && i > metricsLine {
						if strings.Contains(line, "port:") && strings.Contains(line, "8443") {
							portLine = i
						}
					}
				}

				Expect(metricsLine).To(BeNumerically(">", 0))
				Expect(portLine).To(BeNumerically(">", metricsLine))
			})
		})
	})
})
