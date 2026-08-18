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

package templates

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/extractor"
)

const testProjectName = "test-project"

var _ = Describe("HelmValues", func() {
	Describe("env section", func() {
		envEntry := func(name string, entry map[string]any) map[string]any {
			entry["name"] = name
			return entry
		}
		literal := func(name string, value any) map[string]any {
			return envEntry(name, map[string]any{"value": value})
		}

		valuesWithEnv := func(env ...any) string {
			values := &HelmValues{
				Extraction: &extractor.Extraction{
					Values: extractor.ValuesConfig{
						Manager: extractor.ManagerConfig{Env: env},
					},
				},
			}
			values.ProjectName = testProjectName
			return values.generateValues()
		}

		// The list is the Kubernetes shape, so it is emitted as extracted: same entries, same
		// order, valueFrom and all. Nothing is converted, so nothing can be lost in converting.
		It("should emit the source list unchanged and in order", func() {
			result := valuesWithEnv(
				literal("ZBASE", "example.com"),
				literal("APP_URL", "https://$(ZBASE)"),
				envEntry("WATCH_NAMESPACE", map[string]any{
					"valueFrom": map[string]any{
						"fieldRef": map[string]any{"fieldPath": "metadata.namespace"},
					},
				}),
			)

			Expect(result).To(ContainSubstring(`  env:
    - name: ZBASE
      value: example.com
    - name: APP_URL
      value: https://$(ZBASE)
    - name: WATCH_NAMESPACE
      valueFrom:
        fieldRef:
          fieldPath: metadata.namespace
`))
		})

		// Both keys exist even with no variables, so --set can reach them without hand-editing
		// the file first (#5489).
		DescribeTable("should always emit both keys",
			func(env []any, wantEnv string) {
				values := &HelmValues{
					Extraction: &extractor.Extraction{
						Values: extractor.ValuesConfig{Manager: extractor.ManagerConfig{Env: env}},
					},
				}
				values.ProjectName = testProjectName

				result := values.generateValues()

				Expect(result).To(ContainSubstring(wantEnv))
				Expect(result).To(ContainSubstring("  envOverrides: {}"))
			},
			Entry("no env at all", nil, "  env: []"),
			Entry("one variable", []any{literal("FOO", "bar")}, "  env:\n    - name: FOO"),
		)

		It("should not invent an envOrder key", func() {
			Expect(valuesWithEnv(literal("FOO", "bar"))).NotTo(ContainSubstring("envOrder"))
		})

		// A blank line inside a block scalar is part of the value. The generated file is assembled
		// line by line, so it is exactly the kind of place where one gets quietly dropped - and a
		// substring assertion would not notice, hence the parse.
		DescribeTable("should preserve a multiline value through the generated YAML",
			func(value string) {
				var parsed struct {
					Manager struct {
						Env []struct {
							Name  string `json:"name"`
							Value string `json:"value"`
						} `json:"env"`
					} `json:"manager"`
				}

				result := valuesWithEnv(literal("MESSAGE", value))

				Expect(yaml.Unmarshal([]byte(result), &parsed)).To(Succeed(), "generated:\n%s", result)
				Expect(parsed.Manager.Env).To(HaveLen(1))
				Expect(parsed.Manager.Env[0].Value).To(Equal(value), "generated:\n%s", result)
			},
			Entry("an interior blank line", "first line\n\nthird line"),
			Entry("several consecutive blank lines", "first\n\n\n\nlast"),
			Entry("lines that look like YAML keys", "key: not-a-field\n\nother: also-not"),
			Entry("no blank line at all", "first line\nsecond line"),
			Entry("a leading blank line", "\nsecond line"),
		)
	})

	Describe("NetworkPolicy section", func() {
		It("should default networkPolicy.enabled to false when no NetworkPolicy resources exist", func() {
			values := &HelmValues{
				Extraction: nil,
			}
			values.ProjectName = testProjectName

			result := values.generateValues()

			Expect(result).To(ContainSubstring("networkPolicy:\n  enabled: false"))
		})

		It("should set networkPolicy.enabled to true when NetworkPolicy resources are detected", func() {
			values := &HelmValues{
				Extraction: &extractor.Extraction{
					Features: extractor.FeatureSet{
						HasNetworkPolicy: true,
					},
				},
			}
			values.ProjectName = testProjectName

			result := values.generateValues()

			Expect(result).To(ContainSubstring("networkPolicy:\n  enabled: true"))
		})

		It("should set networkPolicy.enabled to false when HasNetworkPolicy is false", func() {
			values := &HelmValues{
				Extraction: &extractor.Extraction{
					Features: extractor.FeatureSet{
						HasNetworkPolicy: false,
					},
				},
			}
			values.ProjectName = testProjectName

			result := values.generateValues()

			Expect(result).To(ContainSubstring("networkPolicy:\n  enabled: false"))
		})
	})

	Describe("Prometheus section", func() {
		It("should default prometheus.enabled to false when no ServiceMonitor exists", func() {
			values := &HelmValues{Extraction: nil}
			values.ProjectName = testProjectName

			result := values.generateValues()

			Expect(result).To(ContainSubstring("prometheus:\n  enabled: false"))
		})

		It("should set prometheus.enabled to true when a ServiceMonitor is detected", func() {
			values := &HelmValues{
				Extraction: &extractor.Extraction{
					Features: extractor.FeatureSet{
						HasPrometheus: true,
					},
				},
			}
			values.ProjectName = testProjectName

			result := values.generateValues()

			Expect(result).To(ContainSubstring("prometheus:\n  enabled: true"))
		})
	})

	Describe("Metrics section", func() {
		It("should default metrics.secure to true when there is no extraction", func() {
			values := &HelmValues{Extraction: nil}
			values.ProjectName = testProjectName

			result := values.generateValues()

			Expect(result).To(ContainSubstring("  secure: true"))
		})

		It("should default metrics.secure to true when the manager does not set the arg", func() {
			values := &HelmValues{
				Extraction: &extractor.Extraction{
					Features: extractor.FeatureSet{
						HasMetrics: true,
					},
				},
			}
			values.ProjectName = testProjectName

			result := values.generateValues()

			Expect(result).To(ContainSubstring("  secure: true"))
		})

		It("should set metrics.secure to false when the manager serves plain HTTP", func() {
			insecure := false
			values := &HelmValues{
				Extraction: &extractor.Extraction{
					Features: extractor.FeatureSet{
						HasMetrics:    true,
						MetricsSecure: &insecure,
					},
				},
			}
			values.ProjectName = testProjectName

			result := values.generateValues()

			Expect(result).To(ContainSubstring("  secure: false"))
		})
	})

	Describe("RoleNamespaces rendering", func() {
		Context("when no roleNamespaces are detected", func() {
			It("should not include roleNamespaces section when Extraction is nil", func() {
				values := &HelmValues{
					Extraction: nil,
				}
				values.ProjectName = testProjectName

				result := values.generateValues()

				Expect(result).NotTo(ContainSubstring("roleNamespaces:"))
				Expect(result).To(ContainSubstring("rbac:"))
			})

			It("should not include roleNamespaces section when roleNamespaces is nil", func() {
				values := &HelmValues{
					Extraction: &extractor.Extraction{
						Features: extractor.FeatureSet{
							RoleNamespaces: nil,
						},
					},
				}
				values.ProjectName = testProjectName

				result := values.generateValues()

				Expect(result).NotTo(ContainSubstring("roleNamespaces:"))
				Expect(result).To(ContainSubstring("rbac:"))
				Expect(result).To(ContainSubstring("namespaced: false"))
				Expect(result).To(ContainSubstring("helpers:"))
			})

			It("should not include roleNamespaces section when roleNamespaces is empty", func() {
				values := &HelmValues{
					Extraction: &extractor.Extraction{
						Features: extractor.FeatureSet{
							RoleNamespaces: map[string]string{},
						},
					},
				}
				values.ProjectName = testProjectName

				result := values.generateValues()

				Expect(result).NotTo(ContainSubstring("roleNamespaces:"))
				Expect(result).To(ContainSubstring("rbac:"))
				Expect(result).To(ContainSubstring("namespaced: false"))
				Expect(result).To(ContainSubstring("helpers:"))
			})
		})

		Context("when roleNamespaces are detected", func() {
			It("should include roleNamespaces section with single mapping", func() {
				values := &HelmValues{
					Extraction: &extractor.Extraction{
						Features: extractor.FeatureSet{
							RoleNamespaces: map[string]string{
								"leader-election-role": "test-namespace",
							},
						},
					},
				}
				values.ProjectName = testProjectName

				result := values.generateValues()

				Expect(result).To(ContainSubstring("roleNamespaces:"))
				Expect(result).To(ContainSubstring(`"leader-election-role": "test-namespace"`))
				Expect(result).To(ContainSubstring("Multi-namespace RBAC role mappings"))
			})

			It("should include roleNamespaces section with multiple mappings", func() {
				values := &HelmValues{
					Extraction: &extractor.Extraction{
						Features: extractor.FeatureSet{
							RoleNamespaces: map[string]string{
								"leader-election-role": "namespace-1",
								"manager-role":         "namespace-2",
							},
						},
					},
				}
				values.ProjectName = testProjectName

				result := values.generateValues()

				Expect(result).To(ContainSubstring("roleNamespaces:"))
				Expect(result).To(ContainSubstring(`"leader-election-role": "namespace-1"`))
				Expect(result).To(ContainSubstring(`"manager-role": "namespace-2"`))
				Expect(result).To(ContainSubstring("Multi-namespace RBAC role mappings"))
			})

			It("should quote keys and values to prevent YAML type coercion", func() {
				values := &HelmValues{
					Extraction: &extractor.Extraction{
						Features: extractor.FeatureSet{
							RoleNamespaces: map[string]string{
								"role-1": "true",
								"role-2": "false",
								"123":    "numeric-namespace",
							},
						},
					},
				}
				values.ProjectName = testProjectName

				result := values.generateValues()

				// Verify values are quoted (prevent "true" -> true boolean)
				Expect(result).To(ContainSubstring(`"role-1": "true"`))
				Expect(result).To(ContainSubstring(`"role-2": "false"`))
				// Verify numeric keys are quoted (prevent 123 -> integer key)
				Expect(result).To(ContainSubstring(`"123": "numeric-namespace"`))
			})
		})
	})

	Describe("Custom ports extraction", func() {
		DescribeTable("port values emitted from detected features",
			func(metricsPort, webhookPort, healthProbePort, wantMetrics, wantWebhook, wantHealthProbe int) {
				values := &HelmValues{
					Extraction: &extractor.Extraction{
						Features: extractor.FeatureSet{
							HasMetrics:      true,
							HasWebhooks:     true,
							MetricsPort:     metricsPort,
							WebhookPort:     webhookPort,
							HealthProbePort: healthProbePort,
						},
					},
				}
				values.ProjectName = testProjectName

				result := values.generateValues()

				Expect(extractSection(result, "metrics:")).To(
					ContainSubstring(fmt.Sprintf("port: %d", wantMetrics)))
				Expect(extractSection(result, "webhook:")).To(
					ContainSubstring(fmt.Sprintf("port: %d", wantWebhook)))
				Expect(extractSection(result, "healthProbe:")).To(
					ContainSubstring(fmt.Sprintf("port: %d", wantHealthProbe)))
			},
			Entry("default ports", 0, 0, 0, 8443, 9443, 8081),
			Entry("custom metrics port", 8080, 0, 0, 8080, 9443, 8081),
			Entry("custom webhook port", 0, 9090, 0, 8443, 9090, 8081),
			Entry("custom health probe port", 0, 0, 9091, 8443, 9443, 9091),
			Entry("all custom ports", 8888, 9999, 7777, 8888, 9999, 7777),
		)

		Context("when the project has no webhooks or metrics", func() {
			It("should still emit the healthProbe section with the default port", func() {
				values := &HelmValues{
					Extraction: &extractor.Extraction{
						Features: extractor.FeatureSet{
							HasMetrics:  false,
							HasWebhooks: false,
						},
					},
				}
				values.ProjectName = testProjectName

				result := values.generateValues()

				healthProbeSection := extractSection(result, "healthProbe:")
				Expect(healthProbeSection).To(ContainSubstring("port: 8081"))
			})
		})

		Context("healthProbe placement", func() {
			It("should nest the healthProbe block under the manager section", func() {
				values := &HelmValues{}
				values.ProjectName = testProjectName

				result := values.generateValues()

				Expect(result).To(ContainSubstring("  healthProbe:\n    # Health probe server port\n    port: 8081\n"))
				Expect(result).NotTo(ContainSubstring("\nhealthProbe:"))
			})
		})
	})
})

// extractSection extracts a section from values.yaml for better error messages.
func extractSection(content, sectionName string) string {
	lines := strings.Split(content, "\n")
	var section []string
	inSection := false

	for _, line := range lines {
		if strings.Contains(line, sectionName) {
			inSection = true
		}
		if inSection {
			section = append(section, line)
			// Stop at next major section (starts at column 0, not indented)
			if len(section) > 1 && len(line) > 0 && line[0] != ' ' && line[0] != '#' {
				break
			}
			if len(section) > 20 {
				break
			}
		}
	}
	return strings.Join(section, "\n")
}
