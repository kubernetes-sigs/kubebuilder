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

package v2

import (
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/resource"
)

var _ input.File = &Types{}

// Types scaffolds the api/<version>/<kind>_types.go file to define the schema for an API
type Types struct {
	input.Input

	// Resource is the resource to scaffold the types_test.go file for
	Resource *resource.Resource

	// Pattern specifies an alternative style of resource to generate
	Pattern string
}

// GetInput implements input.File
func (t *Types) GetInput() (input.Input, error) {
	if t.Path == "" {
		t.Path = filepath.Join("pkg", "apis", t.Resource.Group, t.Resource.Version,
			fmt.Sprintf("%s_types.go", strings.ToLower(t.Resource.Kind)))
	}
	t.TemplateBody = typesTemplate
	t.IfExistsAction = input.Error
	return t.Input, nil
}

// Validate validates the values
func (t *Types) Validate() error {
	return t.Resource.Validate()
}

// JSONTag is a helper to build the json tag for a struct
// It works around escaping problems for the json tag syntax
func (t *Types) JSONTag(tag string) string {
	return fmt.Sprintf("`json:\"%s\"`", tag)
}

var typesTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
{{- if eq .Pattern "addon" }}
	addonv1alpha1 "sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/addon/pkg/apis/v1alpha1"
{{ end }}
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// {{.Resource.Kind}}Spec defines the desired state of {{.Resource.Kind}}
type {{.Resource.Kind}}Spec struct {
{{ if eq .Pattern "addon" }}
	addonv1alpha1.CommonSpec {{ .JSONTag ",inline" }}
	addonv1alpha1.PatchSpec  {{ .JSONTag ",inline" }}

{{ end -}}
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// {{.Resource.Kind}}Status defines the observed state of {{.Resource.Kind}}
type {{.Resource.Kind}}Status struct {
{{ if eq .Pattern "addon" }}
	addonv1alpha1.CommonStatus {{ .JSONTag ",inline" }}

{{ end -}}
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// {{.Resource.Kind}} is the Schema for the {{ .Resource.Resource }} API
type {{.Resource.Kind}} struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`" + `
	metav1.ObjectMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `

	Spec   {{.Resource.Kind}}Spec   ` + "`" + `json:"spec,omitempty"` + "`" + `
	Status {{.Resource.Kind}}Status ` + "`" + `json:"status,omitempty"` + "`" + `
}

{{ if eq .Pattern "addon" }}
var _ addonv1alpha1.CommonObject = &{{.Resource.Kind}}{}

func (o *{{.Resource.Kind}}) ComponentName() string {
	return "{{ .Resource.Kind | lower }}"
}

func (o *{{.Resource.Kind}}) CommonSpec() addonv1alpha1.CommonSpec {
	return o.Spec.CommonSpec
}

func (o *{{.Resource.Kind}}) PatchSpec() addonv1alpha1.PatchSpec {
	return o.Spec.PatchSpec
}

func (o *{{.Resource.Kind}}) GetCommonStatus() addonv1alpha1.CommonStatus {
	return o.Status.CommonStatus
}

func (o *{{.Resource.Kind}}) SetCommonStatus(s addonv1alpha1.CommonStatus) {
	o.Status.CommonStatus = s
}
{{ end }}

// +kubebuilder:object:root=true

// {{.Resource.Kind}}List contains a list of {{.Resource.Kind}}
type {{.Resource.Kind}}List struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`" + `
	metav1.ListMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `
	Items           []{{ .Resource.Kind }} ` + "`" + `json:"items"` + "`" + `
}

func init() {
	SchemeBuilder.Register(&{{.Resource.Kind}}{}, &{{.Resource.Kind}}List{})
}
`
