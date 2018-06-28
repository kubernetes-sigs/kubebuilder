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

package resource

import (
	"path/filepath"

	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
)

var _ input.File = &Register{}

// Register scaffolds the pkg/apis/group/version/register.go file
type Register struct {
	input.Input

	// Resource is the resource to scaffold the types_test.go file for
	Resource *Resource
}

// GetInput implements input.File
func (r *Register) GetInput() (input.Input, error) {
	if r.Path == "" {
		r.Path = filepath.Join("pkg", "apis", r.Resource.Group, r.Resource.Version, "register.go")
	}
	r.TemplateBody = registerTemplate
	return r.Input, nil
}

var registerTemplate = `{{ .Boilerplate }}

// NOTE: Boilerplate only.  Ignore this file.

// Package {{.Resource.Version}} contains API Schema definitions for the {{ .Resource.Group }} {{.Resource.Version}} API group
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package,register
// +k8s:conversion-gen={{ .Repo }}/pkg/apis/{{ .Resource.Group }}
// +k8s:defaulter-gen=TypeMeta
// +groupName={{ .Resource.Group }}.{{ .Domain }}
package {{.Resource.Version}}

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "{{ .Resource.Group }}.{{ .Domain }}", Version: "{{ .Resource.Version }}"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)
`
