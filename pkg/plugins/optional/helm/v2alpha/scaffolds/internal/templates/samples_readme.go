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

package templates

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &SamplesReadme{}

// SamplesReadme scaffolds a README.md for the samples sub-chart
type SamplesReadme struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// OutputDir specifies the output directory for the chart
	OutputDir string
	// Force if true allows overwriting the scaffolded file
	Force bool
}

// SetTemplateDefaults implements machinery.Template
func (f *SamplesReadme) SetTemplateDefaults() error {
	if f.Path == "" {
		outputDir := f.OutputDir
		if outputDir == "" {
			outputDir = defaultOutputDir
		}
		f.Path = filepath.Join(outputDir, "chart", "samples", "README.md")
	}

	f.TemplateBody = samplesReadmeTemplate

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.SkipFile
	}

	return nil
}

const samplesReadmeTemplate = `# {{ .ProjectName }}-samples

Sample Custom Resources for {{ .ProjectName }}.

This chart must be installed AFTER the main {{ .ProjectName }} chart is running.

## Installation

` + "```sh" + `
# Install the main chart first
helm install {{ .ProjectName }} ../

# Wait for the controller to be ready
kubectl rollout status deployment/{{ .ProjectName }}-controller-manager -n {{ .ProjectName }}-system

# Install the samples
helm install {{ .ProjectName }}-samples ./
` + "```" + `

## Uninstall

` + "```sh" + `
helm uninstall {{ .ProjectName }}-samples
` + "```" + `
`
