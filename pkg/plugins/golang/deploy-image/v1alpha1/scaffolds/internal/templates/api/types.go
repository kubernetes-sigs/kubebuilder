/*
Copyright 2022 The Kubernetes Authors.

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

package api

import (
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Types{}

// Types scaffolds the file that defines the schema for a CRD
// nolint:maligned
type Types struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	// Port if informed we will create the scaffold with this spec
	Port string
}

// SetTemplateDefaults implements file.Template
func (f *Types) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup && f.Resource.Group != "" {
			f.Path = filepath.Join("api", "%[group]", "%[version]", "%[kind]_types.go")
		} else {
			f.Path = filepath.Join("api", "%[version]", "%[kind]_types.go")
		}
	}

	f.Path = f.Resource.Replacer().Replace(f.Path)
	log.Println(f.Path)

	f.TemplateBody = typesTemplate

	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

//nolint:lll
const typesTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// {{ .Resource.Kind }}Spec defines the desired state of {{ .Resource.Kind }}
type {{ .Resource.Kind }}Spec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Size defines the number of {{ .Resource.Kind }} instances
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3
	// +kubebuilder:validation:ExclusiveMaximum=false
	Size int32 ` + "`" + `json:"size,omitempty"` + "`" + `

	{{ if not (isEmptyStr .Port) -}}
	// Port defines the port that will be used to init the container with the image
	ContainerPort int32 ` + "`" + `json:"containerPort,omitempty"` + "`" + `
	{{- end }}
}

// {{ .Resource.Kind }}Status defines the observed state of {{ .Resource.Kind }}
type {{ .Resource.Kind }}Status struct {
	// Represents the observations of a {{ .Resource.Kind }}'s current state.
	// {{ .Resource.Kind }}.status.conditions.type are: "Available", "Progressing", and "Degraded"
	// {{ .Resource.Kind }}.status.conditions.status are one of True, False, Unknown.
	// {{ .Resource.Kind }}.status.conditions.reason the value should be a CamelCase string and producers of specific
	// condition types may define expected values and meanings for this field, and whether the values
	// are considered a guaranteed API.
	// {{ .Resource.Kind }}.status.conditions.Message is a human readable message indicating details about the transition.
	// For further information see: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	Conditions []metav1.Condition ` + "`" + `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"` + "`" + `
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
{{- if and (not .Resource.API.Namespaced) (not .Resource.IsRegularPlural) }}
// +kubebuilder:resource:path={{ .Resource.Plural }},scope=Cluster
{{- else if not .Resource.API.Namespaced }}
// +kubebuilder:resource:scope=Cluster
{{- else if not .Resource.IsRegularPlural }}
// +kubebuilder:resource:path={{ .Resource.Plural }}
{{- end }}

// {{ .Resource.Kind }} is the Schema for the {{ .Resource.Plural }} API
type {{ .Resource.Kind }} struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`" + `
	metav1.ObjectMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `

	Spec   {{ .Resource.Kind }}Spec   ` + "`" + `json:"spec,omitempty"` + "`" + `
	Status {{ .Resource.Kind }}Status ` + "`" + `json:"status,omitempty"` + "`" + `
}

// +kubebuilder:object:root=true

// {{ .Resource.Kind }}List contains a list of {{ .Resource.Kind }}
type {{ .Resource.Kind }}List struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`" + `
	metav1.ListMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `
	Items           []{{ .Resource.Kind }} ` + "`" + `json:"items"` + "`" + `
}

func init() {
	SchemeBuilder.Register(&{{ .Resource.Kind }}{}, &{{ .Resource.Kind }}List{})
}
`
