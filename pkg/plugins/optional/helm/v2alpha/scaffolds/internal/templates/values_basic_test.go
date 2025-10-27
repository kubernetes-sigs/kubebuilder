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
				DeploymentConfig: map[string]interface{}{},
			}
			valuesTemplate.ProjectName = "test-project"
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

			Expect(content).To(ContainSubstring("controllerManager:"))
			Expect(content).To(ContainSubstring("metrics:"))
			Expect(content).To(ContainSubstring("prometheus:"))
			Expect(content).To(ContainSubstring("rbacHelpers:"))
		})
	})

	Context("when project has no webhooks", func() {
		BeforeEach(func() {
			valuesTemplate = &HelmValuesBasic{
				HasWebhooks:      false,
				DeploymentConfig: map[string]interface{}{},
			}
			valuesTemplate.ProjectName = "test-project"
			err := valuesTemplate.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not include certManager configuration", func() {
			content := valuesTemplate.GetBody()

			Expect(content).NotTo(ContainSubstring("certManager:"))
		})

		It("should still include other basic sections", func() {
			content := valuesTemplate.GetBody()

			Expect(content).To(ContainSubstring("controllerManager:"))
			Expect(content).To(ContainSubstring("metrics:"))
			Expect(content).To(ContainSubstring("prometheus:"))
			Expect(content).To(ContainSubstring("rbacHelpers:"))
		})

		It("should use extracted values from DeploymentConfig", func() {
			// Test with extracted deployment config
			extractedConfig := map[string]interface{}{
				"image": map[string]interface{}{
					"repository": "custom-controller",
					"tag":        "v2.1.0",
				},
				"imagePullPolicy": "Always",
				"resources": map[string]interface{}{
					"limits": map[string]interface{}{
						"cpu":    "800m",
						"memory": "256Mi",
					},
					"requests": map[string]interface{}{
						"cpu":    "50m",
						"memory": "128Mi",
					},
				},
			}

			valuesWithConfig := &HelmValuesBasic{
				DeploymentConfig: extractedConfig,
				OutputDir:        "dist",
			}
			valuesWithConfig.ProjectName = "test-project"
			err := valuesWithConfig.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())

			content := valuesWithConfig.GetBody()

			// Should use extracted values, not defaults
			Expect(content).To(ContainSubstring("repository: custom-controller"))
			Expect(content).To(ContainSubstring("tag: v2.1.0"))
			Expect(content).To(ContainSubstring("pullPolicy: Always"))
			Expect(content).To(ContainSubstring("cpu: 800m"))
			Expect(content).To(ContainSubstring("memory: 256Mi"))
			Expect(content).To(ContainSubstring("cpu: 50m"))
			Expect(content).To(ContainSubstring("memory: 128Mi"))

			// Should NOT contain default hardcoded values
			Expect(content).NotTo(ContainSubstring("repository: controller"))
			Expect(content).NotTo(ContainSubstring("tag: latest"))
		})
	})

	Context("template path and content", func() {
		BeforeEach(func() {
			valuesTemplate = &HelmValuesBasic{
				OutputDir: "dist",
			}
			valuesTemplate.ProjectName = "test-project"
			err := valuesTemplate.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should have correct path", func() {
			path := valuesTemplate.GetPath()
			// Handle both Windows and Unix path separators
			Expect(path).To(SatisfyAny(Equal("dist/chart/values.yaml"), Equal("dist\\chart\\values.yaml")))
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
			deploymentConfig := map[string]interface{}{
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
			}

			valuesTemplate = &HelmValuesBasic{
				HasWebhooks:      false,
				DeploymentConfig: deploymentConfig,
			}
			valuesTemplate.ProjectName = "test-project"
			err := valuesTemplate.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should include deployment configuration", func() {
			content := valuesTemplate.GetBody()
			Expect(content).To(ContainSubstring("controllerManager:"))
		})
	})

	Context("rbacHelpers configuration", func() {
		BeforeEach(func() {
			valuesTemplate = &HelmValuesBasic{
				HasWebhooks: false,
			}
			valuesTemplate.ProjectName = "test-project"
			err := valuesTemplate.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should have rbacHelpers disabled by default", func() {
			content := valuesTemplate.GetBody()
			lines := strings.Split(content, "\n")
			var rbacHelpersIndex int
			for i, line := range lines {
				if strings.Contains(line, "rbacHelpers:") {
					rbacHelpersIndex = i
					break
				}
			}
			Expect(rbacHelpersIndex).To(BeNumerically(">", 0))
			Expect(lines[rbacHelpersIndex+1]).To(ContainSubstring("enable: false"))
		})
	})
})
