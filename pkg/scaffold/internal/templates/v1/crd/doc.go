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

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &Doc{}

// Doc scaffolds the pkg/apis/group/version/doc.go directory
type Doc struct {
	file.TemplateMixin
	file.RepositoryMixin
	file.BoilerplateMixin
	file.ResourceMixin
}

// SetTemplateDefaults implements input.Template
func (f *Doc) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("pkg", "apis", f.Resource.GroupPackageName, f.Resource.Version, "doc.go")
	}

	f.TemplateBody = docGoTemplate

	return nil
}

// Validate validates the values
func (f *Doc) Validate() error {
	return f.Resource.Validate()
}

// nolint:lll
const docGoTemplate = `{{ .Boilerplate }}

// Package {{ .Resource.Version }} contains API Schema definitions for the {{ .Resource.Group }} {{ .Resource.Version }} API group
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package,register
// +k8s:conversion-gen={{ .Repo }}/pkg/apis/{{ .Resource.GroupPackageName }}
// +k8s:defaulter-gen=TypeMeta
// +groupName={{ .Resource.Domain }}
package {{ .Resource.Version }}
`
