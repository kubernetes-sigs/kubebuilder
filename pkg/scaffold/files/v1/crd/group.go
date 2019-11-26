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

var _ input.File = &Group{}

// Group scaffolds the pkg/apis/group/group.go
type Group struct {
	input.Input

	// Resource is a resource in the API group
	Resource *resource.Resource
}

// GetInput implements input.File
func (g *Group) GetInput() (input.Input, error) {
	if g.Path == "" {
		g.Path = filepath.Join("pkg", "apis", g.Resource.Group, "group.go")
	}
	g.TemplateBody = groupTemplate
	return g.Input, nil
}

// Validate validates the values
func (g *Group) Validate() error {
	return g.Resource.Validate()
}

const groupTemplate = `{{ .Boilerplate }}

// Package {{ .Resource.GroupImportSafe }} contains {{ .Resource.Group }} API versions
package {{ .Resource.GroupImportSafe }}
`
