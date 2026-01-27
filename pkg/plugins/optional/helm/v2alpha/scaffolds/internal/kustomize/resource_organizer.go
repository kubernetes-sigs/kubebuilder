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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ResourceOrganizer groups Kubernetes resources by their logical function
type ResourceOrganizer struct {
	resources *ParsedResources
}

// NewResourceOrganizer creates a new resource organizer
func NewResourceOrganizer(resources *ParsedResources) *ResourceOrganizer {
	return &ResourceOrganizer{
		resources: resources,
	}
}

// OrganizeByFunction groups resources by their logical function matching config/ directory structure
func (o *ResourceOrganizer) OrganizeByFunction() map[string][]*unstructured.Unstructured {
	groups := make(map[string][]*unstructured.Unstructured)

	// CRDs - Custom Resource Definitions
	if len(o.resources.CustomResourceDefinitions) > 0 {
		groups["crd"] = o.resources.CustomResourceDefinitions
	}

	// RBAC - Role-Based Access Control resources
	rbacResources := o.collectRBACResources()
	if len(rbacResources) > 0 {
		groups["rbac"] = rbacResources
	}

	// Manager - Deployment and related resources
	if o.resources.Deployment != nil {
		groups["manager"] = []*unstructured.Unstructured{o.resources.Deployment}
	}

	// Metrics - Metrics services and related resources
	metricsResources := o.collectMetricsResources()
	if len(metricsResources) > 0 {
		groups["metrics"] = metricsResources
	}

	// Webhook - Webhook configurations and webhook services
	webhookResources := o.collectWebhookResources()
	if len(webhookResources) > 0 {
		groups["webhook"] = webhookResources
	}

	// Cert-manager - Certificate issuers and related resources
	certManagerResources := o.collectCertManagerResources()
	if len(certManagerResources) > 0 {
		groups["cert-manager"] = certManagerResources
	}

	// Prometheus - Prometheus ServiceMonitors and monitoring resources
	prometheusResources := o.collectPrometheusResources()
	if len(prometheusResources) > 0 {
		groups["prometheus"] = prometheusResources
	}

	// Extras - Uncategorized resources (services, configmaps, secrets, etc. not fitting above categories)
	// This includes both uncategorized services and all resources from the "Other" category
	extrasResources := o.collectExtrasResources()
	if len(extrasResources) > 0 {
		groups["extras"] = extrasResources
	}

	return groups
}

// collectRBACResources gathers all RBAC-related resources
func (o *ResourceOrganizer) collectRBACResources() []*unstructured.Unstructured {
	var rbacResources []*unstructured.Unstructured

	// Service account
	if o.resources.ServiceAccount != nil {
		rbacResources = append(rbacResources, o.resources.ServiceAccount)
	}

	// Roles and bindings
	rbacResources = append(rbacResources, o.resources.Roles...)
	rbacResources = append(rbacResources, o.resources.ClusterRoles...)
	rbacResources = append(rbacResources, o.resources.RoleBindings...)
	rbacResources = append(rbacResources, o.resources.ClusterRoleBindings...)

	return rbacResources
}

// collectWebhookResources gathers webhook-related resources
func (o *ResourceOrganizer) collectWebhookResources() []*unstructured.Unstructured {
	var webhookResources []*unstructured.Unstructured

	// Webhook configurations (ValidatingWebhookConfiguration, MutatingWebhookConfiguration)
	webhookResources = append(webhookResources, o.resources.WebhookConfigurations...)

	// Webhook services (services containing "webhook" in the name)
	for _, service := range o.resources.Services {
		if o.isWebhookService(service) {
			webhookResources = append(webhookResources, service)
		}
	}

	return webhookResources
}

// collectCertManagerResources gathers cert-manager related resources
func (o *ResourceOrganizer) collectCertManagerResources() []*unstructured.Unstructured {
	var certManagerResources []*unstructured.Unstructured

	// Certificate issuers
	if o.resources.Issuer != nil {
		certManagerResources = append(certManagerResources, o.resources.Issuer)
	}

	// Certificates (both webhook and metrics certificates are cert-manager resources)
	certManagerResources = append(certManagerResources, o.resources.Certificates...)

	return certManagerResources
}

// collectMetricsResources gathers metrics-related resources
func (o *ResourceOrganizer) collectMetricsResources() []*unstructured.Unstructured {
	var metricsResources []*unstructured.Unstructured

	// Metrics services (services containing "metrics" in the name)
	for _, service := range o.resources.Services {
		if o.isMetricsService(service) {
			metricsResources = append(metricsResources, service)
		}
	}

	return metricsResources
}

// collectPrometheusResources gathers prometheus related resources
func (o *ResourceOrganizer) collectPrometheusResources() []*unstructured.Unstructured {
	prometheusResources := make([]*unstructured.Unstructured, 0, len(o.resources.ServiceMonitors))

	// ServiceMonitors
	prometheusResources = append(prometheusResources, o.resources.ServiceMonitors...)

	return prometheusResources
}

// isWebhookService determines if a service is webhook-related
func (o *ResourceOrganizer) isWebhookService(service *unstructured.Unstructured) bool {
	serviceName := service.GetName()
	return strings.Contains(serviceName, "webhook")
}

// isMetricsService determines if a service is metrics-related
func (o *ResourceOrganizer) isMetricsService(service *unstructured.Unstructured) bool {
	serviceName := service.GetName()
	return strings.Contains(serviceName, "metrics")
}

// collectExtrasResources gathers uncategorized resources that don't fit standard categories
func (o *ResourceOrganizer) collectExtrasResources() []*unstructured.Unstructured {
	var extrasResources []*unstructured.Unstructured

	// Collect services that are neither webhook nor metrics services
	for _, service := range o.resources.Services {
		if !o.isWebhookService(service) && !o.isMetricsService(service) {
			extrasResources = append(extrasResources, service)
		}
	}

	// Collect all other uncategorized resources (ConfigMaps, Secrets, etc.)
	extrasResources = append(extrasResources, o.resources.Other...)

	return extrasResources
}
