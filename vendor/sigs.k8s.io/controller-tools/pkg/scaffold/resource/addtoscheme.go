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
	"fmt"
	"path/filepath"

	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
)

var _ input.File = &AddToScheme{}

// AddToScheme scaffolds the code to add the resource to a SchemeBuilder.
type AddToScheme struct {
	input.Input

	// Resource is a resource in the API group
	Resource *Resource
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

var addResourceTemplate = `{{ .Boilerplate }}

package apis

import (
	"{{ .Repo }}/pkg/apis/{{ .Resource.Group }}/{{ .Resource.Version }}"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, {{ .Resource.Version }}.SchemeBuilder.AddToScheme)
}
`
