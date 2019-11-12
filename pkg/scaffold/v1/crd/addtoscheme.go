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
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
)

var _ input.File = &AddToScheme{}

// AddToScheme scaffolds the code to add the resource to a SchemeBuilder.
type AddToScheme struct {
	input.Input

	// Resource is a resource in the API group
	Resource *resource.Resource
}

// GetInput implements input.File
func (a *AddToScheme) GetInput() (input.Input, error) {
	if a.Path == "" {
		a.Path = filepath.Join("pkg", "apis", fmt.Sprintf(
			"addtoscheme_%s_%s.go", a.Resource.Group, a.Resource.Version))
	}
	a.TemplateBody = addResourceTemplate
	return a.Input, nil
}

// Validate validates the values
func (a *AddToScheme) Validate() error {
	return a.Resource.Validate()
}

// NB(directxman12): we need that package alias on the API import otherwise imports.Process
// gets wicked (or hella, if you're feeling west-coasty) confused.

var addResourceTemplate = `{{ .Boilerplate }}

package apis

import (
	api "{{ .Repo }}/pkg/apis/{{ .Resource.Group }}/{{ .Resource.Version }}"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, api.SchemeBuilder.AddToScheme)
}
`
