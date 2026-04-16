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

package extractor

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("DeploymentExtractor", func() {
	Describe("ExtractDeploymentConfig replicas handling", func() {
		var (
			deployment *unstructured.Unstructured
			extractor  *DeploymentExtractor
		)

		BeforeEach(func() {
			deployment = &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]any{
						"name": "test-deployment",
					},
					"spec": map[string]any{
						"template": map[string]any{
							"spec": map[string]any{
								"containers": []any{
									map[string]any{
										"name":  "manager",
										"image": "controller:latest",
									},
								},
							},
						},
					},
				},
			}
			extractor = &DeploymentExtractor{}
		})

		Context("when replicas is not set", func() {
			It("should return nil for replicas", func() {
				result := extractor.ExtractDeploymentConfig(deployment)
				Expect(result.Manager.Replicas).To(BeNil())
			})
		})

		Context("when replicas is set to 0 (scale-to-zero)", func() {
			It("should preserve the zero value", func() {
				deployment.Object["spec"].(map[string]any)["replicas"] = int64(0)

				result := extractor.ExtractDeploymentConfig(deployment)

				Expect(result.Manager.Replicas).NotTo(BeNil())
				Expect(*result.Manager.Replicas).To(Equal(0))
			})
		})

		Context("when replicas is set to 1", func() {
			It("should extract replicas value as 1", func() {
				deployment.Object["spec"].(map[string]any)["replicas"] = int64(1)

				result := extractor.ExtractDeploymentConfig(deployment)

				Expect(result.Manager.Replicas).NotTo(BeNil())
				Expect(*result.Manager.Replicas).To(Equal(1))
			})
		})

		Context("when replicas is set to 3", func() {
			It("should extract replicas value as 3", func() {
				deployment.Object["spec"].(map[string]any)["replicas"] = int64(3)

				result := extractor.ExtractDeploymentConfig(deployment)

				Expect(result.Manager.Replicas).NotTo(BeNil())
				Expect(*result.Manager.Replicas).To(Equal(3))
			})
		})
	})

	Describe("ExtractPortFromArg", func() {
		Context("when argument has valid port format", func() {
			It("should extract port from :PORT format", func() {
				port := ExtractPortFromArg("--metrics-bind-address=:8443")
				Expect(port).To(Equal(8443))
			})

			It("should extract port from HOST:PORT format", func() {
				port := ExtractPortFromArg("--metrics-bind-address=0.0.0.0:8080")
				Expect(port).To(Equal(8080))
			})

			It("should extract port from localhost:PORT format", func() {
				port := ExtractPortFromArg("--metrics-bind-address=127.0.0.1:9090")
				Expect(port).To(Equal(9090))
			})

			It("should extract port from IPv6 format", func() {
				port := ExtractPortFromArg("--webhook-port=[::1]:9443")
				Expect(port).To(Equal(9443))
			})
		})

		Context("when argument has invalid format", func() {
			It("should return 0 for missing equals sign", func() {
				port := ExtractPortFromArg("--metrics-bind-address:8443")
				Expect(port).To(Equal(0))
			})

			It("should return 0 for non-numeric port", func() {
				port := ExtractPortFromArg("--metrics-bind-address=:abc")
				Expect(port).To(Equal(0))
			})

			It("should return 0 for port out of valid range (too low)", func() {
				port := ExtractPortFromArg("--metrics-bind-address=:0")
				Expect(port).To(Equal(0))
			})

			It("should return 0 for port out of valid range (too high)", func() {
				port := ExtractPortFromArg("--metrics-bind-address=:99999")
				Expect(port).To(Equal(0))
			})

			It("should return 0 for empty port", func() {
				port := ExtractPortFromArg("--metrics-bind-address=:")
				Expect(port).To(Equal(0))
			})

			It("should return 0 for missing port value", func() {
				port := ExtractPortFromArg("--metrics-bind-address=")
				Expect(port).To(Equal(0))
			})
		})

		Context("when handling edge cases", func() {
			It("should handle port at minimum valid value", func() {
				port := ExtractPortFromArg("--metrics-bind-address=:1")
				Expect(port).To(Equal(1))
			})

			It("should handle port at maximum valid value", func() {
				port := ExtractPortFromArg("--metrics-bind-address=:65535")
				Expect(port).To(Equal(65535))
			})

			It("should handle multiple colons (IPv6)", func() {
				port := ExtractPortFromArg("--metrics-bind-address=::1:8443")
				Expect(port).To(Equal(8443))
			})
		})
	})
})
