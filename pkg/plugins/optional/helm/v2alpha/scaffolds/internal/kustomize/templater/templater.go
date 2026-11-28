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

package templater

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/kustomize/templater/appliers"
)

// TemplatedResource represents a single templated resource.
type TemplatedResource struct {
	Kind          string
	Name          string
	TemplatedYAML string
}

// Templater applies Helm template syntax to kustomize-rendered Kubernetes resources.
// It converts kustomize manifests into Helm chart templates by adding:
//   - Template variables for resource names, namespaces, and labels
//   - Conditional rendering based on values.yaml configuration
//   - Release-specific metadata and annotations
//
// The templater preserves the structure of the original resources while making them
// configurable through Helm values.
type Templater struct {
	detectedPrefix   string
	chartName        string
	managerNamespace string
	roleNamespaces   map[string]string
}

func NewTemplater(
	detectedPrefix, chartName, managerNamespace string, roleNamespaces map[string]string,
) *Templater {
	return &Templater{
		detectedPrefix:   detectedPrefix,
		chartName:        chartName,
		managerNamespace: managerNamespace,
		roleNamespaces:   roleNamespaces,
	}
}

// GetManagerNamespace returns the manager namespace.
func (t *Templater) GetManagerNamespace() string {
	return t.managerNamespace
}

// applyCommonSubstitutions applies the substitutions every resource gets, whatever its kind.
func (t *Templater) applyCommonSubstitutions(yamlContent string, resource *unstructured.Unstructured) string {
	yamlContent = appliers.EscapeExistingTemplateSyntax(yamlContent)
	yamlContent = appliers.AddConditionalWrappers(yamlContent, resource, t.detectedPrefix)
	yamlContent = appliers.SubstituteProjectNames(yamlContent, resource)
	yamlContent = appliers.SubstituteNamespace(
		t.detectedPrefix, t.chartName, t.managerNamespace, t.roleNamespaces, yamlContent, resource)
	yamlContent = appliers.SubstituteCertManagerReferences(t.detectedPrefix, t.chartName, yamlContent, resource)
	yamlContent = appliers.SubstituteResourceNamesWithPrefix(t.detectedPrefix, t.chartName, yamlContent, resource)
	yamlContent = appliers.AddHelmLabelsAndAnnotations(t.detectedPrefix, t.chartName, yamlContent)
	return appliers.SubstituteRBACValues(t.detectedPrefix, t.chartName, yamlContent)
}

// RenderLabels returns labels as the chart renders them on a resource, one "key: value" line each.
// The resources the chart adds select the pods and the Services from config/ with these lines, so
// they get the same substitutions.
func (t *Templater) RenderLabels(labels map[string]string) []string {
	if len(labels) == 0 {
		return nil
	}
	// The substitutions match indented lines, as the labels are in a resource.
	content, err := yaml.Marshal(map[string]any{"selector": labels})
	if err != nil {
		return nil
	}
	const indent = "  "
	var lines []string
	rendered := t.applyCommonSubstitutions(string(content), &unstructured.Unstructured{})
	for line := range strings.SplitSeq(rendered, "\n") {
		if label, indented := strings.CutPrefix(line, indent); indented {
			lines = append(lines, label)
		}
	}
	return lines
}

// ApplyHelmSubstitutions applies Helm template syntax to a single resource.
// This is the main transformation orchestrator that coordinates all template substitutions.
func (t *Templater) ApplyHelmSubstitutions(yamlContent string, resource *unstructured.Unstructured) string {
	yamlContent = t.applyCommonSubstitutions(yamlContent, resource)
	if resource.GetKind() == common.KindCRD {
		yamlContent = appliers.AddCRDLabelsAndAnnotations(t.chartName, yamlContent)
	}
	if resource.GetKind() == common.KindServiceAccount {
		yamlContent = appliers.TemplateServiceAccount(t.detectedPrefix, t.chartName, yamlContent)
	}
	if resource.GetKind() == common.KindDeployment && appliers.IsManagerDeployment(resource) {
		yamlContent = appliers.AddCustomLabelsAndAnnotations(t.chartName, yamlContent, appliers.SelectorKeys(resource))
		yamlContent = appliers.TemplateDeploymentFields(t.detectedPrefix, t.chartName, yamlContent)
		yamlContent = appliers.MakeContainerArgsConditional(yamlContent)
		yamlContent = appliers.MakeWebhookVolumeMountsConditional(yamlContent)
		yamlContent = appliers.MakeWebhookVolumesConditional(yamlContent)
		yamlContent = appliers.MakeMetricsVolumeMountsConditional(yamlContent)
		yamlContent = appliers.MakeMetricsVolumesConditional(yamlContent)
	}
	if resource.GetKind() == common.KindService ||
		resource.GetKind() == common.KindDeployment ||
		resource.GetKind() == common.KindNetworkPolicy {
		yamlContent = appliers.TemplatePorts(yamlContent, resource, t.detectedPrefix)
	}
	if resource.GetKind() == common.KindServiceMonitor {
		yamlContent = appliers.AddServiceMonitorLabelsAndAnnotations(t.chartName, yamlContent)
		yamlContent = appliers.TemplateServiceMonitor(yamlContent)
	}
	yamlContent = appliers.CollapseBlankLineAfterIf(yamlContent)

	return yamlContent
}

// templatePorts is a wrapper for testing purposes, exposing the appliers.TemplatePorts function
func (t *Templater) templatePorts(yamlContent string, resource *unstructured.Unstructured) string {
	return appliers.TemplatePorts(yamlContent, resource, t.detectedPrefix)
}
