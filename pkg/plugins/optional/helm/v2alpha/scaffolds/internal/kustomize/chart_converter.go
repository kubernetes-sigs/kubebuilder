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
	"slices"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/extractor"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/kustomize/templater"
)

// ChartConverter converts kustomize output to Helm chart templates.
// It categorizes resources, applies templating, and generates machinery.Builders.
type ChartConverter struct {
	resources      *ParsedResources
	detectedPrefix string
	chartName      string
	outputDir      string

	categorizer *ResourceCategorizer
	templater   *templater.Templater
	generator   *ChartGenerator
}

// NewChartConverter creates a new chart converter.
func NewChartConverter(
	resources *ParsedResources, detectedPrefix, chartName, managerNamespace, outputDir string,
	roleNamespaces map[string]string,
) *ChartConverter {
	categorizer := NewResourceCategorizer(resources)
	t := templater.NewTemplater(detectedPrefix, chartName, managerNamespace, roleNamespaces)
	chartGenerator := NewChartGenerator(t, detectedPrefix)

	return &ChartConverter{
		resources:      resources,
		detectedPrefix: detectedPrefix,
		chartName:      chartName,
		outputDir:      outputDir,
		categorizer:    categorizer,
		templater:      t,
		generator:      chartGenerator,
	}
}

// GetChartBuilders converts resources to machinery.Builders for chart template files.
func (c *ChartConverter) GetChartBuilders() []machinery.Builder {
	resourceGroups := c.categorizer.CategorizeByFunction()

	for groupName, resources := range resourceGroups {
		resourceGroups[groupName] = dedupeResources(resources)
	}

	chartFiles := c.generator.GenerateChart(resourceGroups)

	// Sort filenames for deterministic order
	filenames := make([]string, 0, len(chartFiles.TemplateFiles))
	for filename := range chartFiles.TemplateFiles {
		filenames = append(filenames, filename)
	}
	slices.Sort(filenames)

	builders := make([]machinery.Builder, 0, len(filenames))
	for _, filename := range filenames {
		builders = append(builders, &DynamicTemplate{
			RelativePath: filename,
			Content:      chartFiles.TemplateFiles[filename],
			OutputDir:    c.outputDir,
		})
	}

	return builders
}

// dedupeResources removes duplicate resources to prevent rendering the same resource multiple times.
func dedupeResources(resources []*unstructured.Unstructured) []*unstructured.Unstructured {
	seen := make(map[string]struct{})
	out := make([]*unstructured.Unstructured, 0, len(resources))
	for _, r := range resources {
		if r == nil {
			continue
		}
		key := r.GetAPIVersion() + "|" + r.GetKind() + "|" + r.GetNamespace() + "|" + r.GetName()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, r)
	}
	return out
}

// ExtractDeploymentConfig extracts configuration values from the deployment for values.yaml.
func (c *ChartConverter) ExtractDeploymentConfig() map[string]any {
	deploymentExtractor := extractor.DeploymentExtractor{}
	valuesConfig := deploymentExtractor.ExtractDeploymentConfig(c.resources.Deployment)

	return convertValuesConfigToMap(valuesConfig)
}

// convertValuesConfigToMap converts ValuesConfig struct to map[string]any.
func convertValuesConfigToMap(vc extractor.ValuesConfig) map[string]any {
	config := make(map[string]any)
	m := vc.Manager

	if m.Replicas != nil {
		config["replicas"] = *m.Replicas
	}
	if m.Image.Repository != "" || m.Image.Tag != "" || m.Image.PullPolicy != "" {
		config["image"] = map[string]any{
			"repository": m.Image.Repository,
			"tag":        m.Image.Tag,
			"pullPolicy": m.Image.PullPolicy,
		}
	}
	if m.Resources != nil {
		config["resources"] = m.Resources
	}
	if m.NodeSelector != nil {
		config["podNodeSelector"] = m.NodeSelector
	}
	if m.Tolerations != nil {
		config["podTolerations"] = m.Tolerations
	}
	if m.Affinity != nil {
		config["podAffinity"] = m.Affinity
	}
	if m.Args != nil {
		config["args"] = m.Args
	}
	if m.Env != nil {
		config["env"] = m.Env
	}
	if m.SecurityContext != nil {
		config["securityContext"] = m.SecurityContext
	}
	if m.PodSecurityContext != nil {
		config["podSecurityContext"] = m.PodSecurityContext
	}
	if m.ImagePullSecrets != nil {
		config["imagePullSecrets"] = m.ImagePullSecrets
	}
	if m.PriorityClassName != "" {
		config["priorityClassName"] = m.PriorityClassName
	}
	if m.TopologySpreadConstraints != nil {
		config["topologySpreadConstraints"] = m.TopologySpreadConstraints
	}
	if m.TerminationGracePeriodSeconds != nil {
		config["terminationGracePeriodSeconds"] = *m.TerminationGracePeriodSeconds
	}
	if m.Strategy != nil {
		config["strategy"] = m.Strategy
	}
	if m.ExtraVolumes != nil {
		config["extraVolumes"] = m.ExtraVolumes
	}
	if m.ExtraVolumeMounts != nil {
		config["extraVolumeMounts"] = m.ExtraVolumeMounts
	}

	if vc.WebhookPort > 0 {
		config["webhookPort"] = vc.WebhookPort
	}
	if vc.MetricsPort > 0 {
		config["metricsPort"] = vc.MetricsPort
	}

	return config
}
