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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/extractor"
)

const testProjectName = "test-project"

var _ = Describe("HelmValues", func() {
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
		Context("when using default ports", func() {
			It("should use default metrics port 8443 and webhook port 9443", func() {
				values := &HelmValues{
					Extraction: &extractor.Extraction{
						Features: extractor.FeatureSet{
							HasMetrics:  true,
							HasWebhooks: true,
							MetricsPort: 0,
							WebhookPort: 0,
						},
					},
				}
				values.ProjectName = testProjectName

				result := values.generateValues()

				metricsSection := extractSection(result, "metrics:")
				Expect(metricsSection).To(ContainSubstring("port: 8443"))

				webhookSection := extractSection(result, "webhook:")
				Expect(webhookSection).To(ContainSubstring("port: 9443"))
			})
		})

		Context("when using custom metrics port", func() {
			It("should use custom metrics port 8080", func() {
				values := &HelmValues{
					Extraction: &extractor.Extraction{
						Features: extractor.FeatureSet{
							HasMetrics:  true,
							HasWebhooks: true,
							MetricsPort: 8080,
							WebhookPort: 0,
						},
					},
				}
				values.ProjectName = testProjectName

				result := values.generateValues()

				metricsSection := extractSection(result, "metrics:")
				Expect(metricsSection).To(ContainSubstring("port: 8080"))

				webhookSection := extractSection(result, "webhook:")
				Expect(webhookSection).To(ContainSubstring("port: 9443"))
			})
		})

		Context("when using custom webhook port", func() {
			It("should use custom webhook port 9090", func() {
				values := &HelmValues{
					Extraction: &extractor.Extraction{
						Features: extractor.FeatureSet{
							HasMetrics:  true,
							HasWebhooks: true,
							MetricsPort: 0,
							WebhookPort: 9090,
						},
					},
				}
				values.ProjectName = testProjectName

				result := values.generateValues()

				metricsSection := extractSection(result, "metrics:")
				Expect(metricsSection).To(ContainSubstring("port: 8443"))

				webhookSection := extractSection(result, "webhook:")
				Expect(webhookSection).To(ContainSubstring("port: 9090"))
			})
		})

		Context("when using both custom ports", func() {
			It("should use custom metrics port 8888 and webhook port 9999", func() {
				values := &HelmValues{
					Extraction: &extractor.Extraction{
						Features: extractor.FeatureSet{
							HasMetrics:  true,
							HasWebhooks: true,
							MetricsPort: 8888,
							WebhookPort: 9999,
						},
					},
				}
				values.ProjectName = testProjectName

				result := values.generateValues()

				metricsSection := extractSection(result, "metrics:")
				Expect(metricsSection).To(ContainSubstring("port: 8888"))

				webhookSection := extractSection(result, "webhook:")
				Expect(webhookSection).To(ContainSubstring("port: 9999"))
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
