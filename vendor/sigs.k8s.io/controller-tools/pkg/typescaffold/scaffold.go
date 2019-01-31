/*
Copyright 2018 The Kubernetes Authors.

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

package typescaffold

import (
	"io"
	"strings"
	"text/template"
)

var (
	typesTemplateRaw = `// {{.Resource.Kind}}Spec defines the desired state of {{.Resource.Kind}}
type {{.Resource.Kind}}Spec struct {
	// INSERT ADDITIONAL SPEC FIELDS -- desired state of cluster
{{- if .AdditionalHelp }}
{{- range .AdditionalHelp | SplitLines }}
	// {{.}}
{{- end }}
{{- end }}
}

// {{.Resource.Kind}}Status defines the observed state of {{.Resource.Kind}}.
// It should always be reconstructable from the state of the cluster and/or outside world.
type {{.Resource.Kind}}Status struct {
	// INSERT ADDITIONAL STATUS FIELDS -- observed state of cluster
{{- if .AdditionalHelp }}
{{- range .AdditionalHelp | SplitLines }}
	// {{.}}
{{- end }}
{{- end }}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
{{- if .GenerateClients }}
// +genclient
{{- if not .Resource.Namespaced }}
// +genclient:nonNamespaced
{{- end }}
{{- end }}

// {{.Resource.Kind}} is the Schema for the {{ .Resource.Resource }} API
// +k8s:openapi-gen=true
type {{.Resource.Kind}} struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`" + `
	metav1.ObjectMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `

	Spec   {{.Resource.Kind}}Spec   ` + "`" + `json:"spec,omitempty"` + "`" + `
	Status {{.Resource.Kind}}Status ` + "`" + `json:"status,omitempty"` + "`" + `
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
{{- if and (.GenerateClients) (not .Resource.Namespaced) }}
// +genclient:nonNamespaced
{{- end }}

// {{.Resource.Kind}}List contains a list of {{.Resource.Kind}}
type {{.Resource.Kind}}List struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`" + `
	metav1.ListMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `
	Items           []{{ .Resource.Kind }} ` + "`" + `json:"items"` + "`" + `
}
`
	typesTemplateHelpers = template.FuncMap{
		"SplitLines": func(raw string) []string { return strings.Split(raw, "\n") },
	}

	typesTemplate = template.Must(template.New("object-scaffolding").Funcs(typesTemplateHelpers).Parse(typesTemplateRaw))
)

// ScaffoldOptions describes how to scaffold out a Kubernetes object
// with the basic metadata and comment annotations required to generate code
// for and conform to runtime.Object and metav1.Object.
type ScaffoldOptions struct {
	Resource        Resource
	AdditionalHelp  string
	GenerateClients bool
}

// Validate validates the options, returning an error if anything is invalid.
func (o *ScaffoldOptions) Validate() error {
	if err := o.Resource.Validate(); err != nil {
		return err
	}

	return nil
}

// Scaffold prints the Kubernetes object scaffolding to the given output.
func (o *ScaffoldOptions) Scaffold(out io.Writer) error {
	return typesTemplate.Execute(out, o)
}
