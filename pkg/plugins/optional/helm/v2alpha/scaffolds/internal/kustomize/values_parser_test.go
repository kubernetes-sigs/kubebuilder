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
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestValuesParser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ValuesParser Suite")
}

var _ = Describe("ValuesParser", func() {
	var (
		parser    *ValuesParser
		tempDir   string
		chartDir  string
		valuesPath string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "values-parser-test-*")
		Expect(err).NotTo(HaveOccurred())
		
		chartDir = filepath.Join(tempDir, "chart")
		err = os.MkdirAll(chartDir, 0o755)
		Expect(err).NotTo(HaveOccurred())
		
		valuesPath = filepath.Join(chartDir, "values.yaml")
		parser = NewValuesParser(tempDir)
	})

	AfterEach(func() {
		if tempDir != "" {
			_ = os.RemoveAll(tempDir)
		}
	})

	Describe("ParseExistingValues", func() {
		Context("when values.yaml does not exist", func() {
			It("should return empty values without error", func() {
				parsed, err := parser.ParseExistingValues()
				Expect(err).NotTo(HaveOccurred())
				Expect(parsed).NotTo(BeNil())
				Expect(parsed.Raw).To(BeEmpty())
				Expect(parsed.Path).To(Equal(valuesPath))
			})
		})

		Context("when values.yaml exists with standard values", func() {
			BeforeEach(func() {
				valuesContent := `manager:
  replicas: 1
  image:
    repository: controller
    tag: latest
    pullPolicy: IfNotPresent
  env: []
  
crd:
  enable: true
  keep: true`
				err := os.WriteFile(valuesPath, []byte(valuesContent), 0o644)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should parse the values successfully", func() {
				parsed, err := parser.ParseExistingValues()
				Expect(err).NotTo(HaveOccurred())
				Expect(parsed).NotTo(BeNil())
				Expect(parsed.Raw).To(HaveKey("manager"))
				Expect(parsed.Raw).To(HaveKey("crd"))
			})
		})

		Context("when values.yaml contains invalid YAML", func() {
			BeforeEach(func() {
				invalidYAML := `manager:
  replicas: 1
  image: [this is, not: valid: yaml`
				err := os.WriteFile(valuesPath, []byte(invalidYAML), 0o644)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return an error", func() {
				_, err := parser.ParseExistingValues()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("DetectUserAddedValues", func() {
		Context("when values contain only standard fields", func() {
			It("should not detect any user-added values", func() {
				parsed := &ParsedValues{
					Raw: map[string]any{
						"manager": map[string]any{
							"replicas": 1,
							"image": map[string]any{
								"repository": "controller",
								"tag":        "latest",
							},
						},
						"crd": map[string]any{
							"enable": true,
						},
					},
				}

				userValues := parser.DetectUserAddedValues(parsed)
				Expect(userValues).NotTo(BeNil())
				Expect(userValues.ManagerLabels).To(BeEmpty())
				Expect(userValues.ManagerAnnotations).To(BeEmpty())
				Expect(userValues.CustomManagerFields).To(BeEmpty())
				Expect(userValues.CustomTopLevelFields).To(BeEmpty())
			})
		})

		Context("when values contain custom manager labels", func() {
			It("should detect the custom labels", func() {
				parsed := &ParsedValues{
					Raw: map[string]any{
						"manager": map[string]any{
							"replicas": 1,
							"labels": map[string]any{
								"custom-label":  "custom-value",
								"another-label": "another-value",
							},
						},
					},
				}

				userValues := parser.DetectUserAddedValues(parsed)
				Expect(userValues.ManagerLabels).To(HaveLen(2))
				Expect(userValues.ManagerLabels).To(HaveKeyWithValue("custom-label", "custom-value"))
				Expect(userValues.ManagerLabels).To(HaveKeyWithValue("another-label", "another-value"))
			})
		})

		Context("when values contain custom manager annotations", func() {
			It("should detect the custom annotations", func() {
				parsed := &ParsedValues{
					Raw: map[string]any{
						"manager": map[string]any{
							"replicas": 1,
							"annotations": map[string]any{
								"custom.io/annotation": "value",
							},
						},
					},
				}

				userValues := parser.DetectUserAddedValues(parsed)
				Expect(userValues.ManagerAnnotations).To(HaveLen(1))
				Expect(userValues.ManagerAnnotations).To(HaveKeyWithValue("custom.io/annotation", "value"))
			})
		})

		Context("when values contain custom manager fields", func() {
			It("should detect custom fields beyond standard ones", func() {
				parsed := &ParsedValues{
					Raw: map[string]any{
						"manager": map[string]any{
							"replicas":        1,
							"customField":     "customValue",
							"additionalPorts": []any{map[string]any{"name": "custom", "port": 8080}},
						},
					},
				}

				userValues := parser.DetectUserAddedValues(parsed)
				Expect(userValues.CustomManagerFields).To(HaveLen(2))
				Expect(userValues.CustomManagerFields).To(HaveKey("customField"))
				Expect(userValues.CustomManagerFields).To(HaveKey("additionalPorts"))
			})
		})

		Context("when values contain custom top-level fields", func() {
			It("should detect custom top-level fields", func() {
				parsed := &ParsedValues{
					Raw: map[string]any{
						"manager": map[string]any{
							"replicas": 1,
						},
						"customSection": map[string]any{
							"enabled": true,
						},
						"myCustomConfig": "myValue",
					},
				}

				userValues := parser.DetectUserAddedValues(parsed)
				Expect(userValues.CustomTopLevelFields).To(HaveLen(2))
				Expect(userValues.CustomTopLevelFields).To(HaveKey("customSection"))
				Expect(userValues.CustomTopLevelFields).To(HaveKey("myCustomConfig"))
			})
		})

		Context("when values contain custom pod labels and annotations", func() {
			It("should detect pod-level customizations", func() {
				parsed := &ParsedValues{
					Raw: map[string]any{
						"manager": map[string]any{
							"replicas": 1,
							"podLabels": map[string]any{
								"pod-label": "pod-value",
							},
							"podAnnotations": map[string]any{
								"pod.annotation.io/key": "pod-annotation-value",
							},
						},
					},
				}

				userValues := parser.DetectUserAddedValues(parsed)
				Expect(userValues.ManagerPodLabels).To(HaveLen(1))
				Expect(userValues.ManagerPodLabels).To(HaveKeyWithValue("pod-label", "pod-value"))
				Expect(userValues.ManagerPodAnnotations).To(HaveLen(1))
				Expect(userValues.ManagerPodAnnotations).To(HaveKeyWithValue("pod.annotation.io/key", "pod-annotation-value"))
			})
		})

		Context("when values contain environment variables", func() {
			It("should detect env vars", func() {
				parsed := &ParsedValues{
					Raw: map[string]any{
						"manager": map[string]any{
							"replicas": 1,
							"env": []any{
								map[string]any{
									"name":  "CUSTOM_VAR",
									"value": "custom-value",
								},
								map[string]any{
									"name":  "ANOTHER_VAR",
									"value": "another-value",
								},
							},
						},
					},
				}

				userValues := parser.DetectUserAddedValues(parsed)
				Expect(userValues.EnvVars).To(HaveLen(2))
				Expect(userValues.EnvVars[0]).To(HaveKeyWithValue("name", "CUSTOM_VAR"))
				Expect(userValues.EnvVars[1]).To(HaveKeyWithValue("name", "ANOTHER_VAR"))
			})
		})
	})

	Describe("GetValuePath", func() {
		It("should retrieve nested values correctly", func() {
			values := map[string]any{
				"manager": map[string]any{
					"image": map[string]any{
						"repository": "myrepo",
						"tag":        "v1.0.0",
					},
				},
			}

			val, found := GetValuePath(values, "manager.image.repository")
			Expect(found).To(BeTrue())
			Expect(val).To(Equal("myrepo"))

			val, found = GetValuePath(values, "manager.image.tag")
			Expect(found).To(BeTrue())
			Expect(val).To(Equal("v1.0.0"))

			_, found = GetValuePath(values, "manager.nonexistent")
			Expect(found).To(BeFalse())
		})

		It("should handle top-level values", func() {
			values := map[string]any{
				"topLevel": "value",
			}

			val, found := GetValuePath(values, "topLevel")
			Expect(found).To(BeTrue())
			Expect(val).To(Equal("value"))
		})
	})
})
