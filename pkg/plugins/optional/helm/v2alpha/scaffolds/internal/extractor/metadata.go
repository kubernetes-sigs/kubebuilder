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

// defaultImageTag is used when no version tag can be extracted from the deployment image.
const defaultImageTag = "latest"

// MetadataExtractor extracts chart metadata from resources.
type MetadataExtractor struct{}

// ChartMetadata contains chart metadata extracted from resources.
// It includes the Helm chart name, kustomize namePrefix, manager namespace, and manager version.
type ChartMetadata struct {
	ChartName        string
	DetectedPrefix   string
	ManagerNamespace string
	ManagerVersion   string
}

// ExtractMetadata extracts chart metadata from parsed resources.
// projectName is the configured project name which becomes the chart name.
func (m *MetadataExtractor) ExtractMetadata(resources *ResourceSet, projectName string) ChartMetadata {
	detectedPrefix := estimatePrefix(resources)
	if detectedPrefix == "" && projectName != "" {
		detectedPrefix = projectName
	}
	managerNamespace := getManagerNamespace(resources, detectedPrefix)
	managerVersion := extractDeployManagerVersion(resources.Deployment)

	chartName := projectName
	if chartName == "" {
		chartName = detectedPrefix
		if chartName == "" {
			chartName = "chart"
		}
	}

	return ChartMetadata{
		ChartName:        chartName,
		DetectedPrefix:   detectedPrefix,
		ManagerNamespace: managerNamespace,
		ManagerVersion:   managerVersion,
	}
}

// estimatePrefix estimates the kustomize namePrefix from resource names.
func estimatePrefix(resources *ResourceSet) string {
	// Try to estimate from deployment name
	if resources.Deployment != nil {
		name := resources.Deployment.GetName()
		if suffix := "-controller-manager"; strings.HasSuffix(name, suffix) {
			prefix := strings.TrimSuffix(name, suffix)
			if validatePrefix(prefix, resources.Services) {
				return prefix
			}
		}
	}

	// Try to estimate from service account name
	if resources.ServiceAccount != nil {
		name := resources.ServiceAccount.GetName()
		if suffix := "-controller-manager"; strings.HasSuffix(name, suffix) {
			prefix := strings.TrimSuffix(name, suffix)
			if validatePrefix(prefix, resources.Services) {
				return prefix
			}
		}
	}

	// Try to estimate from namespace name
	if resources.Namespace != nil {
		name := resources.Namespace.GetName()
		if suffix := "-system"; strings.HasSuffix(name, suffix) {
			prefix := strings.TrimSuffix(name, suffix)
			if validatePrefix(prefix, resources.Services) {
				return prefix
			}
		}
	}

	return ""
}

// validatePrefix checks if the prefix is consistent with all Services.
func validatePrefix(prefix string, services []*unstructured.Unstructured) bool {
	if len(services) == 0 {
		return true
	}

	for _, svc := range services {
		if !strings.HasPrefix(svc.GetName(), prefix+"-") {
			return false
		}
	}

	return true
}

// getManagerNamespace returns the namespace where the manager deployment runs
func getManagerNamespace(resources *ResourceSet, detectedPrefix string) string {
	// First try to get from deployment
	if resources.Deployment != nil {
		ns := resources.Deployment.GetNamespace()
		if ns != "" {
			return ns
		}
	}

	// Fallback to namespace resource
	if resources.Namespace != nil {
		return resources.Namespace.GetName()
	}

	// Fallback to estimated namespace from prefix
	if detectedPrefix != "" {
		return detectedPrefix + "-system"
	}

	return "system"
}

// extractDeployManagerVersion extracts the manager version from deployment image tag.
// Returns empty string when no meaningful version can be extracted (nil deployment, missing fields, or "latest" tag).
func extractDeployManagerVersion(deployment *unstructured.Unstructured) string {
	if deployment == nil {
		return ""
	}

	// Extract containers using NestedFieldNoCopy to avoid deep copy issues with integer fields
	containers, found, err := unstructured.NestedFieldNoCopy(deployment.Object, "spec", "template", "spec", "containers")
	if !found || err != nil {
		return ""
	}

	containersList, ok := containers.([]any)
	if !ok || len(containersList) == 0 {
		return ""
	}

	firstContainer, ok := containersList[0].(map[string]any)
	if !ok {
		return ""
	}

	image, found, err := unstructured.NestedString(firstContainer, "image")
	if !found || err != nil || image == "" {
		return ""
	}

	// Strip digest suffix if present (e.g., repository:tag@sha256:... or repository@sha256:...)
	// appVersion must be a semantic version, not a digest
	imageWithoutDigest := image
	if before, _, ok0 := strings.Cut(image, "@"); ok0 {
		imageWithoutDigest = before
	}

	// Extract tag from image (format: repository:tag)
	lastColon := strings.LastIndex(imageWithoutDigest, ":")
	lastSlash := strings.LastIndex(imageWithoutDigest, "/")
	if lastColon != -1 && lastColon > lastSlash {
		if lastColon+1 < len(imageWithoutDigest) {
			tag := imageWithoutDigest[lastColon+1:]
			// Ignore "latest" tag as it's not a real version
			if tag != "" && tag != defaultImageTag {
				return tag
			}
		}
	}

	// Return empty string to let the chart template use its default
	return ""
}
