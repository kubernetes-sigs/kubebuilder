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

package kustomize

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// SamplesWriter writes CustomResource samples to either a sub-chart or main chart templates
type SamplesWriter struct {
	outputDir  string
	isSubchart bool
}

// NewSamplesWriter creates a new samples writer
// If isSubchart is true, samples go to samples/templates/
// If false, samples go to templates/samples/ (main chart)
func NewSamplesWriter(outputDir string, isSubchart bool) *SamplesWriter {
	return &SamplesWriter{
		outputDir:  outputDir,
		isSubchart: isSubchart,
	}
}

// WriteSamples writes all CustomResource samples to either:
// - samples/templates/ (if isSubchart=true): separate sub-chart, installed manually after main chart
// - templates/samples/ (if isSubchart=false): conditional templates in main chart ({{ if .Values.samples.install }})
func (w *SamplesWriter) WriteSamples(fs machinery.Filesystem, samples []*unstructured.Unstructured) error {
	for i, sample := range samples {
		if sample == nil {
			continue
		}

		// Generate filename from Kind and name
		kind := sample.GetKind()
		name := sample.GetName()
		if kind == "" || name == "" {
			return fmt.Errorf("CustomResource missing kind or name at index %d", i)
		}

		// Create filename: kind_name.yaml (lowercase)
		filename := fmt.Sprintf("%s_%s.yaml", strings.ToLower(kind), name)

		// Convert to YAML
		yamlBytes, err := yaml.Marshal(sample.Object)
		if err != nil {
			return fmt.Errorf("failed to marshal CustomResource %s/%s to YAML: %w", kind, name, err)
		}

		// Determine file path and content based on whether it's a sub-chart or in templates
		var filePath string
		var content []byte

		if w.isSubchart {
			// Sub-chart: samples/templates/
			filePath = filepath.Join(w.outputDir, "chart", "samples", "templates", filename)
			content = yamlBytes
		} else {
			// Main chart templates: templates/samples/ with conditional
			filePath = filepath.Join(w.outputDir, "chart", "templates", "samples", filename)
			// Wrap in Helm conditional so samples are only installed if explicitly enabled
			wrappedContent := fmt.Sprintf("{{- if .Values.samples.install }}\n%s{{- end }}\n", string(yamlBytes))
			content = []byte(wrappedContent)
		}

		// Create directory if it doesn't exist
		dir := filepath.Dir(filePath)
		if err := fs.FS.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		if err := afero.WriteFile(fs.FS, filePath, content, 0o644); err != nil {
			return fmt.Errorf("failed to write CustomResource %s/%s to %s: %w", kind, name, filePath, err)
		}
	}

	return nil
}
