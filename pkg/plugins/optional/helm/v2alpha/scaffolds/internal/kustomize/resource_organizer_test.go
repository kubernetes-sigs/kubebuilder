//go:build integration

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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("ResourceOrganizer", func() {
	var (
		organizer *ResourceOrganizer
		resources *ParsedResources
	)

	BeforeEach(func() {
		resources = &ParsedResources{
			CustomResourceDefinitions: make([]*unstructured.Unstructured, 0),
			Roles:                     make([]*unstructured.Unstructured, 0),
			ClusterRoles:              make([]*unstructured.Unstructured, 0),
			RoleBindings:              make([]*unstructured.Unstructured, 0),
			ClusterRoleBindings:       make([]*unstructured.Unstructured, 0),
			Services:                  make([]*unstructured.Unstructured, 0),
			Certificates:              make([]*unstructured.Unstructured, 0),
			WebhookConfigurations:     make([]*unstructured.Unstructured, 0),
			ServiceMonitors:           make([]*unstructured.Unstructured, 0),
			Other:                     make([]*unstructured.Unstructured, 0),
		}
	})

	Context("when organizing resources by function", func() {
		It("should categorize webhook services correctly", func() {
			webhookService := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]any{
						"name":      "controller-manager-webhook-service",
						"namespace": "test-system",
					},
				},
			}
			resources.Services = append(resources.Services, webhookService)

			organizer = NewResourceOrganizer(resources)
			groups := organizer.OrganizeByFunction()

			Expect(groups).To(HaveKey("webhook"))
			Expect(groups["webhook"]).To(ContainElement(webhookService))
			Expect(groups).NotTo(HaveKey("extras"))
		})

		It("should categorize metrics services correctly", func() {
			metricsService := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]any{
						"name":      "controller-manager-metrics-service",
						"namespace": "test-system",
					},
				},
			}
			resources.Services = append(resources.Services, metricsService)

			organizer = NewResourceOrganizer(resources)
			groups := organizer.OrganizeByFunction()

			Expect(groups).To(HaveKey("metrics"))
			Expect(groups["metrics"]).To(ContainElement(metricsService))
			Expect(groups).NotTo(HaveKey("extras"))
		})

		It("should categorize manual services as extras", func() {
			manualService := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]any{
						"name":      "my-alert-service",
						"namespace": "test-system",
					},
				},
			}
			resources.Services = append(resources.Services, manualService)

			organizer = NewResourceOrganizer(resources)
			groups := organizer.OrganizeByFunction()

			Expect(groups).To(HaveKey("extras"))
			Expect(groups["extras"]).To(ContainElement(manualService))
			Expect(groups).NotTo(HaveKey("webhook"))
			Expect(groups).NotTo(HaveKey("metrics"))
		})

		It("should categorize multiple services correctly", func() {
			webhookService := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]any{
						"name":      "webhook-service",
						"namespace": "test-system",
					},
				},
			}
			metricsService := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]any{
						"name":      "metrics-service",
						"namespace": "test-system",
					},
				},
			}
			manualService := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]any{
						"name":      "custom-app-service",
						"namespace": "test-system",
					},
				},
			}
			resources.Services = append(resources.Services, webhookService, metricsService, manualService)

			organizer = NewResourceOrganizer(resources)
			groups := organizer.OrganizeByFunction()

			Expect(groups).To(HaveKey("webhook"))
			Expect(groups["webhook"]).To(ContainElement(webhookService))

			Expect(groups).To(HaveKey("metrics"))
			Expect(groups["metrics"]).To(ContainElement(metricsService))

			Expect(groups).To(HaveKey("extras"))
			Expect(groups["extras"]).To(ContainElement(manualService))
			Expect(groups["extras"]).To(HaveLen(1))
		})

		It("should not create extras group when all services are categorized", func() {
			webhookService := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]any{
						"name":      "webhook-service",
						"namespace": "test-system",
					},
				},
			}
			metricsService := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]any{
						"name":      "metrics-service",
						"namespace": "test-system",
					},
				},
			}
			resources.Services = append(resources.Services, webhookService, metricsService)

			organizer = NewResourceOrganizer(resources)
			groups := organizer.OrganizeByFunction()

			Expect(groups).NotTo(HaveKey("extras"))
		})

		It("should categorize ConfigMaps as extras", func() {
			configMap := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]any{
						"name":      "my-config",
						"namespace": "test-system",
					},
				},
			}
			resources.Other = append(resources.Other, configMap)

			organizer = NewResourceOrganizer(resources)
			groups := organizer.OrganizeByFunction()

			Expect(groups).To(HaveKey("extras"))
			Expect(groups["extras"]).To(ContainElement(configMap))
		})

		It("should categorize Secrets as extras", func() {
			secret := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]any{
						"name":      "my-secret",
						"namespace": "test-system",
					},
				},
			}
			resources.Other = append(resources.Other, secret)

			organizer = NewResourceOrganizer(resources)
			groups := organizer.OrganizeByFunction()

			Expect(groups).To(HaveKey("extras"))
			Expect(groups["extras"]).To(ContainElement(secret))
		})

		It("should categorize mixed extras resources correctly", func() {
			manualService := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]any{
						"name":      "custom-service",
						"namespace": "test-system",
					},
				},
			}
			configMap := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]any{
						"name":      "my-config",
						"namespace": "test-system",
					},
				},
			}
			secret := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]any{
						"name":      "my-secret",
						"namespace": "test-system",
					},
				},
			}

			resources.Services = append(resources.Services, manualService)
			resources.Other = append(resources.Other, configMap, secret)

			organizer = NewResourceOrganizer(resources)
			groups := organizer.OrganizeByFunction()

			Expect(groups).To(HaveKey("extras"))
			Expect(groups["extras"]).To(HaveLen(3))
			Expect(groups["extras"]).To(ContainElement(manualService))
			Expect(groups["extras"]).To(ContainElement(configMap))
			Expect(groups["extras"]).To(ContainElement(secret))
		})

		It("should categorize all standard resource types correctly", func() {
			deployment := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]any{
						"name":      "controller-manager",
						"namespace": "test-system",
					},
				},
			}
			resources.Deployment = deployment

			serviceAccount := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "ServiceAccount",
					"metadata": map[string]any{
						"name":      "controller-manager",
						"namespace": "test-system",
					},
				},
			}
			resources.ServiceAccount = serviceAccount

			clusterRole := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "rbac.authorization.k8s.io/v1",
					"kind":       "ClusterRole",
					"metadata": map[string]any{
						"name": "manager-role",
					},
				},
			}
			resources.ClusterRoles = append(resources.ClusterRoles, clusterRole)

			organizer = NewResourceOrganizer(resources)
			groups := organizer.OrganizeByFunction()

			Expect(groups).To(HaveKey("manager"))
			Expect(groups["manager"]).To(ContainElement(deployment))

			Expect(groups).To(HaveKey("rbac"))
			Expect(groups["rbac"]).To(ContainElement(serviceAccount))
			Expect(groups["rbac"]).To(ContainElement(clusterRole))
		})
	})

	Context("when checking service types", func() {
		BeforeEach(func() {
			organizer = NewResourceOrganizer(resources)
		})

		It("should identify webhook services by name", func() {
			webhookService := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]any{
						"name": "webhook-service",
					},
				},
			}
			Expect(organizer.isWebhookService(webhookService)).To(BeTrue())
		})

		It("should identify metrics services by name", func() {
			metricsService := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]any{
						"name": "metrics-service",
					},
				},
			}
			Expect(organizer.isMetricsService(metricsService)).To(BeTrue())
		})

		It("should not identify generic services as webhook or metrics", func() {
			genericService := &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]any{
						"name": "my-app-service",
					},
				},
			}
			Expect(organizer.isWebhookService(genericService)).To(BeFalse())
			Expect(organizer.isMetricsService(genericService)).To(BeFalse())
		})
	})
})

