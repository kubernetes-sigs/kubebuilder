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
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// ChartWriter handles writing Helm chart template files
type ChartWriter struct {
	templater *HelmTemplater
	outputDir string
}

// NewChartWriter creates a new chart writer
func NewChartWriter(templater *HelmTemplater, outputDir string) *ChartWriter {
	return &ChartWriter{
		templater: templater,
		outputDir: outputDir,
	}
}

// WriteResourceGroup writes a group of resources to a Helm template file
func (w *ChartWriter) WriteResourceGroup(fs machinery.Filesystem, groupName string,
	resources []*unstructured.Unstructured,
) error {
	// Special handling for namespace - write as single file
	if groupName == "namespace" {
		return w.writeNamespaceFile(fs, resources[0])
	}

	// For CRDs, certificates, and other resources that should be split, write individual files
	if w.shouldSplitFiles(groupName) {
		return w.writeSplitFiles(fs, groupName, resources)
	}

	// For other groups, write as directory-based template files
	return w.writeGroupDirectory(fs, groupName, resources)
}

// writeNamespaceFile writes the namespace as a single file in templates/
func (w *ChartWriter) writeNamespaceFile(fs machinery.Filesystem, namespace *unstructured.Unstructured) error {
	// Apply Helm templating
	yamlContent := w.convertToYAML(namespace)
	yamlContent = w.templater.ApplyHelmSubstitutions(yamlContent, namespace)

	// Write to templates/namespace.yaml
	filePath := filepath.Join(w.outputDir, "chart", "templates", "namespace.yaml")
	return w.writeFileWithNewline(fs, filePath, yamlContent)
}

// writeGroupDirectory writes resources as files in a group-specific directory
func (w *ChartWriter) writeGroupDirectory(fs machinery.Filesystem, groupName string,
	resources []*unstructured.Unstructured,
) error {
	var finalContent bytes.Buffer

	// Convert each resource to YAML and apply templating
	for i, resource := range resources {
		if i > 0 {
			finalContent.WriteString("---\n")
		}

		yamlContent := w.convertToYAML(resource)
		yamlContent = w.templater.ApplyHelmSubstitutions(yamlContent, resource)
		finalContent.WriteString(yamlContent)
	}

	// Write to templates/{groupName}/{groupName}.yaml
	dirPath := filepath.Join(w.outputDir, "chart", "templates", groupName)
	filePath := filepath.Join(dirPath, groupName+".yaml")

	return w.writeFileWithNewline(fs, filePath, finalContent.String())
}

// convertToYAML converts an unstructured object to YAML string
func (w *ChartWriter) convertToYAML(resource *unstructured.Unstructured) string {
	yamlBytes, err := yaml.Marshal(resource.Object)
	if err != nil {
		return fmt.Sprintf("# Error converting to YAML: %v\n", err)
	}
	return string(yamlBytes)
}

// shouldSplitFiles determines if resources in a group should be written as individual files
func (w *ChartWriter) shouldSplitFiles(groupName string) bool {
	return groupName == "crd" || groupName == "cert-manager" || groupName == "webhook" ||
		groupName == "prometheus" || groupName == "rbac" || groupName == "metrics"
}

// writeSplitFiles writes each resource in the group to its own file
func (w *ChartWriter) writeSplitFiles(fs machinery.Filesystem, groupName string,
	resources []*unstructured.Unstructured,
) error {
	// Create the group directory
	groupDir := filepath.Join(w.outputDir, "chart", "templates", groupName)
	if err := fs.FS.MkdirAll(groupDir, 0o755); err != nil {
		return fmt.Errorf("creating group directory %s: %w", groupDir, err)
	}

	// Write each resource to its own file
	for i, resource := range resources {
		fileName := w.generateFileName(resource, i)
		filePath := filepath.Join(groupDir, fileName)

		yamlContent := w.convertToYAML(resource)
		yamlContent = w.templater.ApplyHelmSubstitutions(yamlContent, resource)

		if err := w.writeFileWithNewline(fs, filePath, yamlContent); err != nil {
			return fmt.Errorf("writing resource file %s: %w", filePath, err)
		}
	}

	return nil
}

// generateFileName creates a unique filename for a resource based on its metadata
func (w *ChartWriter) generateFileName(resource *unstructured.Unstructured, index int) string {
	// Try to use the resource name if available
	if name := resource.GetName(); name != "" {
		// Remove project prefix from the filename for cleaner file names
		projectPrefix := w.templater.projectName + "-"
		fileName := name
		if strings.HasPrefix(name, projectPrefix) {
			fileName = strings.TrimPrefix(name, projectPrefix)
		}

		// Handle special cases where filename might be empty after prefix removal
		if fileName == "" {
			fileName = resource.GetKind()
			if fileName == "" {
				fileName = "resource"
			}
		}

		// Replace dots and other special characters with underscores for filename safety
		fileName = filepath.Base(fileName) // Remove any path separators
		return fmt.Sprintf("%s.yaml", fileName)
	}

	// Fall back to kind + index if no name
	kind := resource.GetKind()
	if kind == "" {
		kind = "resource"
	}
	return fmt.Sprintf("%s-%d.yaml", kind, index)
}

// writeFileWithNewline ensures the file ends with a newline
func (w *ChartWriter) writeFileWithNewline(fs machinery.Filesystem, filePath, content string) error {
	// Ensure content ends with newline
	if content != "" && content[len(content)-1] != '\n' {
		content += "\n"
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := fs.FS.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	// Use afero to write directly through the filesystem
	if err := afero.WriteFile(fs.FS, filePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing file %s: %w", filePath, err)
	}
	return nil
}
