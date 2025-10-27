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
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// ChartConverter orchestrates the conversion of kustomize output to Helm chart templates
type ChartConverter struct {
	resources   *ParsedResources
	projectName string
	outputDir   string

	// Components for conversion
	organizer *ResourceOrganizer
	templater *HelmTemplater
	writer    *ChartWriter
}

// NewChartConverter creates a new chart converter with all necessary components
func NewChartConverter(resources *ParsedResources, projectName, outputDir string) *ChartConverter {
	organizer := NewResourceOrganizer(resources)
	templater := NewHelmTemplater(projectName)
	writer := NewChartWriter(templater, outputDir)

	return &ChartConverter{
		resources:   resources,
		projectName: projectName,
		outputDir:   outputDir,
		organizer:   organizer,
		templater:   templater,
		writer:      writer,
	}
}

// WriteChartFiles converts all resources to Helm chart templates and writes them to the filesystem
func (c *ChartConverter) WriteChartFiles(fs machinery.Filesystem) error {
	// Organize resources by their logical function
	resourceGroups := c.organizer.OrganizeByFunction()

	// Write each group to appropriate template files
	for groupName, resources := range resourceGroups {
		if len(resources) > 0 {
			// De-duplicate exact resources by (apiVersion, kind, namespace, name)
			deduped := dedupeResources(resources)
			if err := c.writer.WriteResourceGroup(fs, groupName, deduped); err != nil {
				return fmt.Errorf("failed to write %s resources: %w", groupName, err)
			}
		}
	}

	return nil
}

// dedupeResources removes exact duplicate resources by keying on
// apiVersion, kind, namespace (optional), and name.
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

// ExtractDeploymentConfig extracts configuration values from the deployment for values.yaml
func (c *ChartConverter) ExtractDeploymentConfig() map[string]interface{} {
	if c.resources.Deployment == nil {
		return make(map[string]interface{})
	}

	config := make(map[string]interface{})

	specMap := extractDeploymentSpec(c.resources.Deployment)
	if specMap == nil {
		return config
	}

	extractPodSecurityContext(specMap, config)

	container := firstManagerContainer(specMap)
	if container == nil {
		return config
	}

	extractContainerImage(container, config)
	extractContainerEnv(container, config)
	extractContainerResources(container, config)
	extractContainerSecurityContext(container, config)

	return config
}

func extractDeploymentSpec(deployment *unstructured.Unstructured) map[string]interface{} {
	spec, found, err := unstructured.NestedFieldNoCopy(deployment.Object, "spec", "template", "spec")
	if !found || err != nil {
		return nil
	}

	specMap, ok := spec.(map[string]interface{})
	if !ok {
		return nil
	}

	return specMap
}

func extractPodSecurityContext(specMap map[string]interface{}, config map[string]interface{}) {
	if podSecurityContext, podSecFound, podSecErr := unstructured.NestedFieldNoCopy(
		specMap,
		"securityContext",
	); podSecFound && podSecErr == nil {
		if podSecMap, podSecOk := podSecurityContext.(map[string]interface{}); podSecOk && len(podSecMap) > 0 {
			config["podSecurityContext"] = podSecurityContext
		}
	}
}

func firstManagerContainer(specMap map[string]interface{}) map[string]interface{} {
	containers, found, err := unstructured.NestedFieldNoCopy(specMap, "containers")
	if !found || err != nil {
		return nil
	}

	containersList, ok := containers.([]interface{})
	if !ok || len(containersList) == 0 {
		return nil
	}

	firstContainer, ok := containersList[0].(map[string]interface{})
	if !ok {
		return nil
	}

	return firstContainer
}

func extractContainerEnv(container map[string]interface{}, config map[string]interface{}) {
	if env, envFound, envErr := unstructured.NestedFieldNoCopy(container, "env"); envFound && envErr == nil {
		if envList, envOk := env.([]interface{}); envOk && len(envList) > 0 {
			config["env"] = envList
		}
	}
}

func extractContainerImage(container map[string]interface{}, config map[string]interface{}) {
	imageRef, imageFound, imageErr := unstructured.NestedString(container, "image")
	if !imageFound || imageErr != nil {
		return
	}

	repository, tag, digest := splitImageReference(imageRef)
	if repository == "" {
		return
	}

	imageConfig := map[string]interface{}{
		"repository": repository,
	}

	if tag != "" {
		imageConfig["tag"] = tag
	}

	if digest != "" {
		imageConfig["digest"] = digest
	}

	if pullPolicy, pullFound, pullErr := unstructured.NestedString(
		container,
		"imagePullPolicy",
	); pullFound && pullErr == nil {
		if trimmed := strings.TrimSpace(pullPolicy); trimmed != "" {
			imageConfig["pullPolicy"] = trimmed
		}
	}

	config["image"] = imageConfig
}

func extractContainerResources(container map[string]interface{}, config map[string]interface{}) {
	if resources, resFound, resErr := unstructured.NestedFieldNoCopy(container, "resources"); resFound && resErr == nil {
		if resourcesMap, resOk := resources.(map[string]interface{}); resOk && len(resourcesMap) > 0 {
			config["resources"] = resources
		}
	}
}

func extractContainerSecurityContext(container map[string]interface{}, config map[string]interface{}) {
	if securityContext, secFound, secErr := unstructured.NestedFieldNoCopy(
		container,
		"securityContext",
	); secFound && secErr == nil {
		if secMap, secOk := securityContext.(map[string]interface{}); secOk && len(secMap) > 0 {
			config["securityContext"] = securityContext
		}
	}
}

func splitImageReference(imageRef string) (repository, tag, digest string) {
	reference := strings.TrimSpace(imageRef)
	if reference == "" {
		return "", "", ""
	}

	if atIndex := strings.Index(reference, "@"); atIndex >= 0 {
		digest = strings.TrimSpace(reference[atIndex+1:])
		reference = reference[:atIndex]
	}

	lastSlash := strings.LastIndex(reference, "/")
	lastColon := strings.LastIndex(reference, ":")
	if lastColon > -1 && lastColon > lastSlash {
		tag = strings.TrimSpace(reference[lastColon+1:])
		reference = reference[:lastColon]
	}

	return strings.TrimSpace(reference), strings.TrimSpace(tag), strings.TrimSpace(digest)
}
