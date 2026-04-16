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

package kustomize

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

// ResourceCategorizer groups Kubernetes resources by their logical function, matching the config/
// directory structure used by kubebuilder. The groups are: crd, rbac, manager, metrics, webhook,
// cert-manager, prometheus, and extras.
//
// This categorization determines how resources are organized in the final Helm chart templates.
type ResourceCategorizer struct {
	resources *ParsedResources
}

// NewResourceCategorizer creates a new resource categorizer.
func NewResourceCategorizer(resources *ParsedResources) *ResourceCategorizer {
	return &ResourceCategorizer{
		resources: resources,
	}
}

// CategorizeByFunction groups resources by their logical function matching config/ directory structure.
func (c *ResourceCategorizer) CategorizeByFunction() map[string][]*unstructured.Unstructured {
	groups := make(map[string][]*unstructured.Unstructured)

	if len(c.resources.CustomResourceDefinitions) > 0 {
		groups["crd"] = c.resources.CustomResourceDefinitions
	}

	rbacResources := c.collectRBACResources()
	if len(rbacResources) > 0 {
		groups["rbac"] = rbacResources
	}

	if c.resources.Deployment != nil {
		groups["manager"] = []*unstructured.Unstructured{c.resources.Deployment}
	}

	metricsResources := c.collectMetricsResources()
	if len(metricsResources) > 0 {
		groups["metrics"] = metricsResources
	}

	webhookResources := c.collectWebhookResources()
	if len(webhookResources) > 0 {
		groups["webhook"] = webhookResources
	}

	certManagerResources := c.collectCertManagerResources()
	if len(certManagerResources) > 0 {
		groups["cert-manager"] = certManagerResources
	}

	prometheusResources := c.collectPrometheusResources()
	if len(prometheusResources) > 0 {
		groups["prometheus"] = prometheusResources
	}

	extrasResources := c.collectExtrasResources()
	if len(extrasResources) > 0 {
		groups["extras"] = extrasResources
	}

	return groups
}

// collectRBACResources gathers all RBAC-related resources.
func (c *ResourceCategorizer) collectRBACResources() []*unstructured.Unstructured {
	var rbacResources []*unstructured.Unstructured

	if c.resources.ServiceAccount != nil {
		rbacResources = append(rbacResources, c.resources.ServiceAccount)
	}

	rbacResources = append(rbacResources, c.resources.Roles...)
	rbacResources = append(rbacResources, c.resources.ClusterRoles...)
	rbacResources = append(rbacResources, c.resources.RoleBindings...)
	rbacResources = append(rbacResources, c.resources.ClusterRoleBindings...)

	return rbacResources
}

// collectWebhookResources gathers webhook-related resources.
func (c *ResourceCategorizer) collectWebhookResources() []*unstructured.Unstructured {
	var webhookResources []*unstructured.Unstructured

	webhookResources = append(webhookResources, c.resources.WebhookConfigurations...)

	for _, service := range c.resources.Services {
		if c.isWebhookService(service) {
			webhookResources = append(webhookResources, service)
		}
	}

	return webhookResources
}

// collectCertManagerResources gathers cert-manager related resources.
func (c *ResourceCategorizer) collectCertManagerResources() []*unstructured.Unstructured {
	var certManagerResources []*unstructured.Unstructured

	if c.resources.Issuer != nil {
		certManagerResources = append(certManagerResources, c.resources.Issuer)
	}

	certManagerResources = append(certManagerResources, c.resources.Certificates...)

	return certManagerResources
}

// collectMetricsResources gathers metrics-related resources.
func (c *ResourceCategorizer) collectMetricsResources() []*unstructured.Unstructured {
	var metricsResources []*unstructured.Unstructured

	for _, service := range c.resources.Services {
		if c.isMetricsService(service) {
			metricsResources = append(metricsResources, service)
		}
	}

	return metricsResources
}

// collectPrometheusResources gathers prometheus related resources.
func (c *ResourceCategorizer) collectPrometheusResources() []*unstructured.Unstructured {
	prometheusResources := make([]*unstructured.Unstructured, 0, len(c.resources.ServiceMonitors))
	prometheusResources = append(prometheusResources, c.resources.ServiceMonitors...)
	return prometheusResources
}

// isWebhookService determines if a service is webhook-related.
// It verifies the KIND is "Service" and name ends with "webhook-service" suffix.
// Suffix matching avoids false positives when project names contain "webhook".
func (c *ResourceCategorizer) isWebhookService(service *unstructured.Unstructured) bool {
	if service.GetKind() != common.KindService {
		return false
	}
	serviceName := service.GetName()
	return strings.HasSuffix(serviceName, "webhook-service")
}

// isMetricsService determines if a service is metrics-related.
// It verifies the KIND is "Service" and name ends with "metrics-service" suffix.
// Suffix matching avoids false positives when project names contain "metrics".
func (c *ResourceCategorizer) isMetricsService(service *unstructured.Unstructured) bool {
	if service.GetKind() != common.KindService {
		return false
	}
	serviceName := service.GetName()
	return strings.HasSuffix(serviceName, "metrics-service")
}

// collectExtrasResources gathers uncategorized resources that don't fit standard categories.
func (c *ResourceCategorizer) collectExtrasResources() []*unstructured.Unstructured {
	var extrasResources []*unstructured.Unstructured

	for _, service := range c.resources.Services {
		if !c.isWebhookService(service) && !c.isMetricsService(service) {
			extrasResources = append(extrasResources, service)
		}
	}

	extrasResources = append(extrasResources, c.resources.Other...)

	return extrasResources
}
