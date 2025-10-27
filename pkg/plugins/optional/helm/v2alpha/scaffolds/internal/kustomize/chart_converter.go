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

	// Extract from deployment spec
	spec, found, err := unstructured.NestedFieldNoCopy(c.resources.Deployment.Object, "spec", "template", "spec")
	if !found || err != nil {
		return config
	}

	specMap, ok := spec.(map[string]interface{})
	if !ok {
		return config
	}

	// Extract pod security context
	if podSecurityContext, podSecFound, podSecErr := unstructured.NestedFieldNoCopy(specMap,
		"securityContext"); podSecFound && podSecErr == nil {
		if podSecMap, podSecOk := podSecurityContext.(map[string]interface{}); podSecOk && len(podSecMap) > 0 {
			config["podSecurityContext"] = podSecurityContext
		}
	}

	// Extract container configuration
	containers, found, err := unstructured.NestedFieldNoCopy(specMap, "containers")
	if !found || err != nil {
		return config
	}

	containersList, ok := containers.([]interface{})
	if !ok || len(containersList) == 0 {
		return config
	}

	// Find manager container by name, fallback to first container
	var targetContainer map[string]interface{}
	for _, c := range containersList {
		container, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if name, nameOk := container["name"].(string); nameOk && name == "manager" {
			targetContainer = container
			break
		}
	}

	// Fallback to first container if manager not found
	if targetContainer == nil {
		if firstContainer, ok := containersList[0].(map[string]interface{}); ok {
			targetContainer = firstContainer
		} else {
			return config
		}
	}

	firstContainer := targetContainer

	// Extract environment variables
	if env, envFound, envErr := unstructured.NestedFieldNoCopy(firstContainer, "env"); envFound && envErr == nil {
		if envList, envOk := env.([]interface{}); envOk && len(envList) > 0 {
			config["env"] = envList
		}
	}

	// Extract resources
	if resources, resFound, resErr := unstructured.NestedFieldNoCopy(firstContainer,
		"resources"); resFound && resErr == nil {
		if resourcesMap, resOk := resources.(map[string]interface{}); resOk && len(resourcesMap) > 0 {
			config["resources"] = resources
		}
	}

	// Extract container security context
	if securityContext, secFound, secErr := unstructured.NestedFieldNoCopy(firstContainer,
		"securityContext"); secFound && secErr == nil {
		if secMap, secOk := securityContext.(map[string]interface{}); secOk && len(secMap) > 0 {
			config["securityContext"] = securityContext
		}
	}

	// Extract image configuration
	if image, found, err := unstructured.NestedString(firstContainer, "image"); found && err == nil && image != "" {
		config["image"] = parseImageString(image)
	}

	// Extract imagePullPolicy
	if pullPolicy, found, err := unstructured.NestedString(firstContainer, "imagePullPolicy"); found && err == nil && pullPolicy != "" {
		config["imagePullPolicy"] = pullPolicy
	}

	return config
}

// parseImageString parses "<repo>[@<digest>]" or "<repo>[:<tag>]".
// It distinguishes registry ports from tags by requiring the tag colon
// to come AFTER the last '/'.
func parseImageString(image string) map[string]interface{} {
	out := make(map[string]interface{})

	// Digest form takes precedence
	if at := strings.IndexByte(image, '@'); at != -1 {
		out["repository"] = image[:at]
		if at+1 < len(image) {
			out["digest"] = image[at+1:]
		}
		return out
	}

	lastSlash := strings.LastIndexByte(image, '/')
	lastColon := strings.LastIndexByte(image, ':')

	// Tag only if the colon comes after the last slash
	if lastColon != -1 && lastColon > lastSlash {
		out["repository"] = image[:lastColon]
		if lastColon+1 < len(image) {
			out["tag"] = image[lastColon+1:]
		}
		return out
	}

	// Untagged/undigested; kube will pull :latest, but we surface it explicitly
	out["repository"] = image
	out["tag"] = "latest"
	return out
}
