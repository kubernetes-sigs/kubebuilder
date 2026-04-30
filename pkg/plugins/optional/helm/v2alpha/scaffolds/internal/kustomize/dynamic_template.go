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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

var _ machinery.Template = &DynamicTemplate{}

// DynamicTemplate is a machinery template wrapper for pre-rendered chart template files.
// Unlike normal templates that use Go template rendering, this simply outputs the content as-is.
type DynamicTemplate struct {
	machinery.TemplateMixin
	machinery.RepositoryMixin

	// RelativePath is the path relative to the chart/templates directory
	RelativePath string
	// Content is the pre-rendered template content
	Content string
	// OutputDir is the base output directory (e.g., "dist")
	OutputDir string
}

// SetTemplateDefaults implements machinery.Template
func (f *DynamicTemplate) SetTemplateDefaults() error {
	outputDir := f.OutputDir
	if outputDir == "" {
		outputDir = common.DefaultOutputDir
	}

	if f.Path == "" {
		f.Path = filepath.Join(outputDir, "chart", "templates", f.RelativePath)
	}

	// Content is already rendered - just set it as the template body
	// Ensure content ends with a newline for POSIX compliance
	content := f.Content
	if len(content) > 0 && content[len(content)-1] != '\n' {
		content += "\n"
	}
	f.TemplateBody = content

	// Use delimiters that won't match Helm template syntax ({{ }})
	// This prevents machinery from trying to execute Helm templates as Go templates
	f.SetDelim("<%", "%>")

	// Always overwrite - these are generated files that should match current kustomize output
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}
