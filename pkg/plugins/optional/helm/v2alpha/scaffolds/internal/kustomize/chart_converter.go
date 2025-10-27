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

	extractContainerEnv(container, config)
	extractContainerImage(container, config)
	extractContainerArgs(container, config)
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
	podSecurityContext, found, err := unstructured.NestedFieldNoCopy(specMap, "securityContext")
	if !found || err != nil {
		return
	}

	podSecMap, ok := podSecurityContext.(map[string]interface{})
	if !ok || len(podSecMap) == 0 {
		return
	}

	config["podSecurityContext"] = podSecurityContext
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
	env, found, err := unstructured.NestedFieldNoCopy(container, "env")
	if !found || err != nil {
		return
	}

	envList, ok := env.([]interface{})
	if !ok || len(envList) == 0 {
		return
	}

	config["env"] = envList
}

func extractContainerImage(container map[string]interface{}, config map[string]interface{}) {
	imageValue, found, err := unstructured.NestedString(container, "image")
	if !found || err != nil || imageValue == "" {
		return
	}

	repository := imageValue
	tag := "latest"
	lastColon := strings.LastIndex(imageValue, ":")
	lastSlash := strings.LastIndex(imageValue, "/")
	if lastColon != -1 && lastColon > lastSlash {
		repository = imageValue[:lastColon]
		if lastColon+1 < len(imageValue) {
			tag = imageValue[lastColon+1:]
		}
	}

	pullPolicy, _, err := unstructured.NestedString(container, "imagePullPolicy")
	if err != nil || pullPolicy == "" {
		pullPolicy = "IfNotPresent"
	}

	config["image"] = map[string]interface{}{
		"repository": repository,
		"tag":        tag,
		"pullPolicy": pullPolicy,
	}
}

func extractContainerArgs(container map[string]interface{}, config map[string]interface{}) {
	args, found, err := unstructured.NestedFieldNoCopy(container, "args")
	if !found || err != nil {
		return
	}

	argsList, ok := args.([]interface{})
	if !ok || len(argsList) == 0 {
		return
	}

	filteredArgs := make([]interface{}, 0, len(argsList))
	for _, rawArg := range argsList {
		strArg, ok := rawArg.(string)
		if !ok {
			filteredArgs = append(filteredArgs, rawArg)
			continue
		}

		// The following arguments should not be exposed under args
		// manager because they are not independently customizable
		if strings.Contains(strArg, "--metrics-bind-address") ||
			strings.Contains(strArg, "--health-probe-bind-address") ||
			strings.Contains(strArg, "--webhook-cert-path") ||
			strings.Contains(strArg, "--metrics-cert-path") {
			continue
		}
		filteredArgs = append(filteredArgs, strArg)
	}

	if len(filteredArgs) > 0 {
		config["args"] = filteredArgs
	}
}

func extractContainerResources(container map[string]interface{}, config map[string]interface{}) {
	resources, found, err := unstructured.NestedFieldNoCopy(container, "resources")
	if !found || err != nil {
		return
	}

	resourcesMap, ok := resources.(map[string]interface{})
	if !ok || len(resourcesMap) == 0 {
		return
	}

	config["resources"] = resources
}

func extractContainerSecurityContext(container map[string]interface{}, config map[string]interface{}) {
	securityContext, found, err := unstructured.NestedFieldNoCopy(container, "securityContext")
	if !found || err != nil {
		return
	}

	secMap, ok := securityContext.(map[string]interface{})
	if !ok || len(secMap) == 0 {
		return
	}

	config["securityContext"] = securityContext
}
