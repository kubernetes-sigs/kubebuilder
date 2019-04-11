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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/resource"
)

var _ input.File = &ResourceDoc{}

// Doc scaffolds the api/version/doc.go directory
type ResourceDoc struct {
	input.Input

	// Resource is a resource for the API version
	Resource *resource.Resource

	// Comments are additional lines to write to the doc.go file
	Comments []string
}

// GetInput implements input.File
func (a *ResourceDoc) GetInput() (input.Input, error) {
	if a.Path == "" {
		a.Path = filepath.Join("api", a.Resource.Version, "doc.go")
	}
	a.TemplateBody = resourceDocGoTemplate
	return a.Input, nil
}

// Validate validates the values
func (a *ResourceDoc) Validate() error {
	return a.Resource.Validate()
}

var resourceDocGoTemplate = `{{ .Boilerplate }}

// Package {{.Resource.Version}} contains API Schema definitions for the {{ .Resource.Group }} {{.Resource.Version}} API group
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package,register
// +k8s:conversion-gen={{ .Repo }}/api/{{ .Resource.Version }}
// +k8s:defaulter-gen=TypeMeta
// +groupName={{ .Resource.Group }}.{{ .Domain }}
package {{.Resource.Version}}
`
