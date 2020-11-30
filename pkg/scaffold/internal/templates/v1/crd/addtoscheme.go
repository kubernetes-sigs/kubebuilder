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

	"sigs.k8s.io/kubebuilder/v2/pkg/model/file"
)

var _ file.Template = &AddToScheme{}

// AddToScheme scaffolds the code to add the resource to a SchemeBuilder.
type AddToScheme struct {
	file.TemplateMixin
	file.BoilerplateMixin
	file.ResourceMixin
}

// SetTemplateDefaults implements input.Template
func (f *AddToScheme) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("pkg", "apis", "addtoscheme_%[group-package-name]_%[version].go")
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.TemplateBody = addResourceTemplate

	return nil
}

// NB(directxman12): we need that package alias on the API import otherwise imports.Process
// gets wicked (or hella, if you're feeling west-coasty) confused.

const addResourceTemplate = `{{ .Boilerplate }}

package apis

import (
	{{ .Resource.ImportAlias }} "{{ .Resource.Package }}"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, {{ .Resource.ImportAlias }}.SchemeBuilder.AddToScheme)
}
`
