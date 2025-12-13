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
	"strconv"
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
func (c *ChartConverter) ExtractDeploymentConfig() map[string]any {
	if c.resources.Deployment == nil {
		return make(map[string]any)
	}

	config := make(map[string]any)
	specMap := extractDeploymentSpec(c.resources.Deployment)
	if specMap == nil {
		return config
	}

	extractPodSecurityContext(specMap, config)
	extractImagePullSecrets(specMap, config)
	extractPodNodeSelector(specMap, config)
	extractPodTolerations(specMap, config)
	extractPodAffinity(specMap, config)

	container := firstManagerContainer(specMap)
	if container == nil {
		return config
	}

	extractContainerEnv(container, config)
	extractContainerImage(container, config)
	extractContainerArgs(container, config)
	extractContainerPorts(container, config)
	extractContainerResources(container, config)
	extractContainerSecurityContext(container, config)

	return config
}

func extractDeploymentSpec(deployment *unstructured.Unstructured) map[string]any {
	spec, found, err := unstructured.NestedFieldNoCopy(deployment.Object, "spec", "template", "spec")
	if !found || err != nil {
		return nil
	}

	specMap, ok := spec.(map[string]any)
	if !ok {
		return nil
	}

	return specMap
}

func extractImagePullSecrets(specMap map[string]any, config map[string]any) {
	imagePullSecrets, found, err := unstructured.NestedFieldNoCopy(specMap, "imagePullSecrets")
	if !found || err != nil {
		return
	}

	imagePullSecretsList, ok := imagePullSecrets.([]any)
	if !ok || len(imagePullSecretsList) == 0 {
		return
	}

	config["imagePullSecrets"] = imagePullSecretsList
}

func extractPodSecurityContext(specMap map[string]any, config map[string]any) {
	podSecurityContext, found, err := unstructured.NestedFieldNoCopy(specMap, "securityContext")
	if !found || err != nil {
		return
	}

	podSecMap, ok := podSecurityContext.(map[string]any)
	if !ok || len(podSecMap) == 0 {
		return
	}

	config["podSecurityContext"] = podSecurityContext
}

func extractPodNodeSelector(specMap map[string]any, config map[string]any) {
	raw, found, err := unstructured.NestedFieldNoCopy(specMap, "nodeSelector")
	if !found || err != nil {
		return
	}

	result, ok := raw.(map[string]any)
	if !ok || len(result) == 0 {
		return
	}

	config["podNodeSelector"] = result
}

func extractPodTolerations(specMap map[string]any, config map[string]any) {
	raw, found, err := unstructured.NestedFieldNoCopy(specMap, "tolerations")
	if !found || err != nil {
		return
	}

	result, ok := raw.([]any)
	if !ok || len(result) == 0 {
		return
	}

	config["podTolerations"] = result
}

func extractPodAffinity(specMap map[string]any, config map[string]any) {
	raw, found, err := unstructured.NestedFieldNoCopy(specMap, "affinity")
	if !found || err != nil {
		return
	}

	result, ok := raw.(map[string]any)
	if !ok || len(result) == 0 {
		return
	}

	config["podAffinity"] = result
}

func firstManagerContainer(specMap map[string]any) map[string]any {
	containers, found, err := unstructured.NestedFieldNoCopy(specMap, "containers")
	if !found || err != nil {
		return nil
	}

	containersList, ok := containers.([]any)
	if !ok || len(containersList) == 0 {
		return nil
	}

	firstContainer, ok := containersList[0].(map[string]any)
	if !ok {
		return nil
	}

	return firstContainer
}

func extractContainerEnv(container map[string]any, config map[string]any) {
	env, found, err := unstructured.NestedFieldNoCopy(container, "env")
	if !found || err != nil {
		return
	}

	envList, ok := env.([]any)
	if !ok || len(envList) == 0 {
		return
	}

	config["env"] = envList
}

func extractContainerImage(container map[string]any, config map[string]any) {
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

	config["image"] = map[string]any{
		"repository": repository,
		"tag":        tag,
		"pullPolicy": pullPolicy,
	}
}

func extractContainerArgs(container map[string]any, config map[string]any) {
	args, found, err := unstructured.NestedFieldNoCopy(container, "args")
	if !found || err != nil {
		return
	}

	argsList, ok := args.([]any)
	if !ok || len(argsList) == 0 {
		return
	}

	filteredArgs := make([]any, 0, len(argsList))
	for _, rawArg := range argsList {
		strArg, ok := rawArg.(string)
		if !ok {
			filteredArgs = append(filteredArgs, rawArg)
			continue
		}

		// Extract port values from bind-address arguments and store them
		// These arguments should not be exposed under args because they will be
		// reconstructed from the port values in values.yaml
		if strings.Contains(strArg, "--metrics-bind-address") {
			if port := extractPortFromArg(strArg); port > 0 {
				if _, exists := config["metricsPort"]; !exists {
					config["metricsPort"] = port
				}
			}
			continue
		}
		if strings.Contains(strArg, "--health-probe-bind-address") {
			continue
		}
		if strings.Contains(strArg, "--webhook-cert-path") ||
			strings.Contains(strArg, "--metrics-cert-path") {
			continue
		}
		filteredArgs = append(filteredArgs, strArg)
	}

	if len(filteredArgs) > 0 {
		config["args"] = filteredArgs
	}
}

// extractPortFromArg extracts port number from arguments like "--metrics-bind-address=:8443"
func extractPortFromArg(arg string) int {
	// Handle formats: --flag=:8443, --flag=0.0.0.0:8443, etc.
	parts := strings.Split(arg, "=")
	if len(parts) != 2 {
		return 0
	}

	portPart := parts[1]
	// Remove leading : or host part
	if idx := strings.LastIndex(portPart, ":"); idx != -1 {
		portPart = portPart[idx+1:]
	}

	port, err := strconv.Atoi(portPart)
	if err != nil || port <= 0 || port > 65535 {
		return 0
	}
	return port
}

// extractContainerPorts extracts port configurations from container ports
func extractContainerPorts(container map[string]any, config map[string]any) {
	// Use NestedFieldNoCopy to avoid deep copy issues with int values
	portsField, found, err := unstructured.NestedFieldNoCopy(container, "ports")
	if !found || err != nil {
		return
	}

	ports, ok := portsField.([]any)
	if !ok {
		return
	}

	for _, p := range ports {
		portMap, ok := p.(map[string]any)
		if !ok {
			continue
		}

		name, _ := portMap["name"].(string)
		var containerPort int

		// Try int64 first (from YAML unmarshaling)
		if cp, ok := portMap["containerPort"].(int64); ok {
			containerPort = int(cp)
		} else if cp, ok := portMap["containerPort"].(int); ok {
			containerPort = cp
		} else {
			continue
		}

		// Look for webhook-server port
		if name == "webhook-server" || strings.Contains(name, "webhook") {
			if _, exists := config["webhookPort"]; !exists {
				config["webhookPort"] = containerPort
			}
		}
	}
}

func extractContainerResources(container map[string]any, config map[string]any) {
	resources, found, err := unstructured.NestedFieldNoCopy(container, "resources")
	if !found || err != nil {
		return
	}

	resourcesMap, ok := resources.(map[string]any)
	if !ok || len(resourcesMap) == 0 {
		return
	}

	config["resources"] = resources
}

func extractContainerSecurityContext(container map[string]any, config map[string]any) {
	securityContext, found, err := unstructured.NestedFieldNoCopy(container, "securityContext")
	if !found || err != nil {
		return
	}

	secMap, ok := securityContext.(map[string]any)
	if !ok || len(secMap) == 0 {
		return
	}

	config["securityContext"] = securityContext
}
