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

package appliers

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

// SubstituteCertManagerReferences applies cert-manager specific template substitutions.
func SubstituteCertManagerReferences(
	detectedPrefix, chartName string,
	yamlContent string,
	resource *unstructured.Unstructured,
) string {
	kind := resource.GetKind()

	if kind == common.KindIssuer || kind == common.KindCertificate {
		hardcodedIssuerRef := detectedPrefix + "-selfsigned-issuer"
		yamlContent = strings.ReplaceAll(
			yamlContent, hardcodedIssuerRef, ResourceNameTemplate(chartName, "selfsigned-issuer"))
	}

	if kind == common.KindValidatingWebhook || kind == common.KindMutatingWebhook || kind == common.KindCRD {
		hardcodedService := "name: " + detectedPrefix + "-webhook-service"
		templatedService := "name: " + ResourceNameTemplate(chartName, "webhook-service")
		yamlContent = strings.ReplaceAll(yamlContent, hardcodedService, templatedService)
	}

	yamlContent = SubstituteCertManagerAnnotations(detectedPrefix, chartName, yamlContent)
	return yamlContent
}

// SubstituteCertManagerAnnotations replaces hardcoded cert-manager cert names with Helm templates.
func SubstituteCertManagerAnnotations(detectedPrefix, chartName, yamlContent string) string {
	hardcodedServingCert := detectedPrefix + "-serving-cert"
	yamlContent = strings.ReplaceAll(yamlContent, hardcodedServingCert, ResourceNameTemplate(chartName, "serving-cert"))
	hardcodedMetricsCert := detectedPrefix + "-metrics-certs"
	yamlContent = strings.ReplaceAll(yamlContent, hardcodedMetricsCert, ResourceNameTemplate(chartName, "metrics-certs"))
	return yamlContent
}
