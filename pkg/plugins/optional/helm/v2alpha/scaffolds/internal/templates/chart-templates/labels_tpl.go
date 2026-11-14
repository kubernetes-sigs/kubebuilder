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

package charttemplates

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

var _ machinery.Template = &HelmLabels{}

// HelmLabels scaffolds the _labels.tpl file, with the helper the generated templates call to render
// labels and annotations. It always regenerates, so the helper matches the templates that call it.
type HelmLabels struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// OutputDir specifies the output directory for the chart
	OutputDir string
}

// SetTemplateDefaults sets the default template configuration
func (f *HelmLabels) SetTemplateDefaults() error {
	if f.Path == "" {
		outputDir := f.OutputDir
		if outputDir == "" {
			outputDir = common.DefaultOutputDir
		}
		f.Path = filepath.Join(outputDir, "chart", "templates", "_labels.tpl")
	}

	// The helpers are written as one raw string, so their Helm actions reach the chart as they are.
	f.TemplateBody = "{{`" + fmt.Sprintf(helmLabelsTemplate, f.ProjectName) + "`}}\n"
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

const helmLabelsTemplate = `{{/*
Renders a label or annotation: the value set in values.yaml, as a string, or the default from config/.
Usage: include "%[1]s.valueOrDefault" (list <labels or annotations from the values> "<key>" "<default>")
*/}}
{{- define "%[1]s.valueOrDefault" -}}
{{- $value := index (index . 0 | default dict) (index . 1) }}
{{- if kindIs "invalid" $value }}
{{- index . 2 }}
{{- else }}
{{- kindIs "string" $value | ternary $value (toJson $value) | toJson }}
{{- end }}
{{- end }}

{{/*
Renders labels or annotations set in values.yaml. Each value is a string, and a null value is left out.
Usage: include "%[1]s.stringMap" <labels or annotations from the values>
*/}}
{{- define "%[1]s.stringMap" -}}
{{- $strings := dict }}
{{- range $key, $value := . }}
{{- if not (kindIs "invalid" $value) }}
{{- $_ := set $strings $key (kindIs "string" $value | ternary $value (toJson $value)) }}
{{- end }}
{{- end }}
{{- if $strings }}
{{- toYaml $strings }}
{{- end }}
{{- end }}`
