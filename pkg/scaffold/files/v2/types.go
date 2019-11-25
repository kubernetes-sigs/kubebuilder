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
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
)

var _ input.File = &Types{}

// Types scaffolds the api/<version>/<kind>_types.go file to define the schema for an API
type Types struct {
	input.Input

	// Resource is the resource to scaffold the types_test.go file for
	Resource *resource.Resource
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

const typesTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// {{.Resource.Kind}}Spec defines the desired state of {{.Resource.Kind}}
type {{.Resource.Kind}}Spec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of {{.Resource.Kind}}. Edit {{.Resource.Kind}}_types.go to remove/update
	Foo string ` + "`" + `json:"foo,omitempty"` + "`" + `
}

// {{.Resource.Kind}}Status defines the observed state of {{.Resource.Kind}}
type {{.Resource.Kind}}Status struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
{{ if not .Resource.Namespaced }} // +kubebuilder:resource:scope=Cluster {{ end }}

// {{.Resource.Kind}} is the Schema for the {{ .Resource.Resource }} API
type {{.Resource.Kind}} struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`" + `
	metav1.ObjectMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `

	Spec   {{.Resource.Kind}}Spec   ` + "`" + `json:"spec,omitempty"` + "`" + `
	Status {{.Resource.Kind}}Status ` + "`" + `json:"status,omitempty"` + "`" + `
}

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
