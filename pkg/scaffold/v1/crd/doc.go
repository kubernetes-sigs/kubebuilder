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

package crd

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
)

var _ input.File = &Doc{}

// Doc scaffolds the pkg/apis/group/version/doc.go directory
type Doc struct {
	input.Input

	// Resource is a resource for the API version
	Resource *resource.Resource
}

// GetInput implements input.File
func (f *Doc) GetInput() (input.Input, error) {
	if f.Path == "" {
		f.Path = filepath.Join("pkg", "apis", f.Resource.Group, f.Resource.Version, "doc.go")
	}
	f.TemplateBody = docGoTemplate
	return f.Input, nil
}

// Validate validates the values
func (f *Doc) Validate() error {
	return f.Resource.Validate()
}

// nolint:lll
const docGoTemplate = `{{ .Boilerplate }}

// Package {{.Resource.Version}} contains API Schema definitions for the {{ .Resource.GroupImportSafe }} {{.Resource.Version}} API group
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package,register
// +k8s:conversion-gen={{ .Repo }}/pkg/apis/{{ .Resource.Group }}
// +k8s:defaulter-gen=TypeMeta
// +groupName={{ .Resource.Group }}.{{ .Domain }}
package {{.Resource.Version}}
`
