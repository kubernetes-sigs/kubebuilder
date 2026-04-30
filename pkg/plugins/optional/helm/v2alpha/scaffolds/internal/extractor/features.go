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
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// FeaturesExtractor detects features from resources.
type FeaturesExtractor struct{}

// FeatureSet represents detected features in the resources.
// It includes flags for CRDs, webhooks, metrics, Prometheus, cert-manager, and cluster-scoped RBAC.
// It also includes port configurations and multi-namespace RBAC mappings.
type FeatureSet struct {
	HasCRDs              bool
	HasWebhooks          bool
	HasMetrics           bool
	HasPrometheus        bool
	HasCertManager       bool
	HasClusterScopedRBAC bool
	WebhookPort          int
	MetricsPort          int
	RoleNamespaces       map[string]string
}

// DetectFeatures detects features from parsed resources.
// The namePrefix is the project prefix used in resource names.
// The managerNamespace is the namespace where the manager deployment runs.
func (f *FeaturesExtractor) DetectFeatures(resources *ResourceSet, namePrefix, managerNamespace string) FeatureSet {
	features := FeatureSet{
		WebhookPort:    9443,
		MetricsPort:    8443,
		RoleNamespaces: make(map[string]string),
	}

	features.HasCRDs = len(resources.CustomResourceDefinitions) > 0
	features.HasWebhooks = len(resources.WebhookConfigurations) > 0

	if resources.Issuer != nil {
		features.HasCertManager = true
	}
	if len(resources.Certificates) > 0 {
		features.HasCertManager = true
	}

	features.HasPrometheus = len(resources.ServiceMonitors) > 0

	for _, svc := range resources.Services {
		name := svc.GetName()
		if strings.HasSuffix(name, "-metrics-service") || strings.HasSuffix(name, "-controller-manager-metrics-service") {
			features.HasMetrics = true
			if port := extractPortFromService(svc); port > 0 {
				features.MetricsPort = port
			}
			break
		}
	}

	if features.HasWebhooks {
		webhookPortFromDeployment := false
		if resources.Deployment != nil {
			if port := extractWebhookPortFromDeployment(resources.Deployment); port > 0 {
				features.WebhookPort = port
				webhookPortFromDeployment = true
			}
		}

		// Only extract from service if we didn't find it in deployment
		if !webhookPortFromDeployment {
			for _, svc := range resources.Services {
				name := svc.GetName()
				if strings.HasSuffix(name, "-webhook-service") {
					if port := extractPortFromService(svc); port > 0 {
						features.WebhookPort = port
					}
					break
				}
			}
		}
	}

	// Detect cluster-scoped RBAC for business logic.
	// Kubebuilder scaffolds metrics-auth-role and metrics-reader which must remain cluster-scoped.
	// This checks if there are additional ClusterRoles for business logic that can be converted to
	// namespace-scoped Roles via the rbac.namespaced toggle.
	for _, cr := range resources.ClusterRoles {
		name := cr.GetName()
		if strings.HasSuffix(name, "-metrics-auth-role") || strings.HasSuffix(name, "-metrics-reader") {
			continue
		}
		features.HasClusterScopedRBAC = true
		break
	}

	// Collect Roles and RoleBindings that deploy to namespaces other than the manager namespace.
	// This enables multi-namespace RBAC scenarios.
	for _, role := range resources.Roles {
		ns := role.GetNamespace()
		if ns != "" && ns != managerNamespace {
			roleName := role.GetName()
			suffix := strings.TrimPrefix(roleName, namePrefix+"-")
			features.RoleNamespaces[suffix] = ns
		}
	}
	for _, binding := range resources.RoleBindings {
		ns := binding.GetNamespace()
		if ns != "" && ns != managerNamespace {
			bindingName := binding.GetName()
			suffix := strings.TrimPrefix(bindingName, namePrefix+"-")
			features.RoleNamespaces[suffix] = ns
		}
	}

	return features
}

// extractPortFromService extracts the port number from a service.
func extractPortFromService(svc *unstructured.Unstructured) int {
	ports, found, err := unstructured.NestedFieldNoCopy(svc.Object, "spec", "ports")
	if !found || err != nil {
		return 0
	}

	portsList, ok := ports.([]any)
	if !ok || len(portsList) == 0 {
		return 0
	}

	firstPort, ok := portsList[0].(map[string]any)
	if !ok {
		return 0
	}

	if port, ok := firstPort["port"].(int64); ok {
		return int(port)
	}
	if port, ok := firstPort["port"].(int); ok {
		return port
	}

	return 0
}

// extractWebhookPortFromDeployment extracts the webhook port from deployment container ports.
func extractWebhookPortFromDeployment(deployment *unstructured.Unstructured) int {
	containers, found, err := unstructured.NestedFieldNoCopy(deployment.Object, "spec", "template", "spec", "containers")
	if !found || err != nil {
		return 0
	}

	containersList, ok := containers.([]any)
	if !ok || len(containersList) == 0 {
		return 0
	}

	firstContainer, ok := containersList[0].(map[string]any)
	if !ok {
		return 0
	}

	portsField, found, err := unstructured.NestedFieldNoCopy(firstContainer, "ports")
	if !found || err != nil {
		return 0
	}

	portsList, ok := portsField.([]any)
	if !ok {
		return 0
	}

	for _, p := range portsList {
		portMap, ok := p.(map[string]any)
		if !ok {
			continue
		}

		name, _ := portMap["name"].(string)
		name = strings.ToLower(name)
		if name == "webhook-server" || name == "webhook" {
			if cp, ok := portMap["containerPort"].(int64); ok {
				return int(cp)
			}
			if cp, ok := portMap["containerPort"].(int); ok {
				return cp
			}
		}
	}

	return 0
}
