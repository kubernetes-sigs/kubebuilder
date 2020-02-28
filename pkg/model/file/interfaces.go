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

package file

import (
	"text/template"

	"sigs.k8s.io/kubebuilder/pkg/model/resource"
)

// Builder defines the basic methods that any file builder must implement
type Builder interface {
	// GetPath returns the path to the file location
	GetPath() string
	// GetIfExistsAction returns the behavior when creating a file that already exists
	GetIfExistsAction() IfExistsAction
}

// RequiresValidation is a file builder that requires validation
type RequiresValidation interface {
	Builder
	// Validate returns true if the template has valid values
	Validate() error
}

// Template is file builder based on a file template
type Template interface {
	Builder
	// GetBody returns the template body
	GetBody() string
	// SetTemplateDefaults sets the default values for templates
	SetTemplateDefaults() error
}

// Inserter is a file builder that inserts code fragments in marked positions
type Inserter interface {
	Builder
	// GetMarkers returns the different markers where code fragments will be inserted
	GetMarkers() []Marker
	// GetCodeFragments returns a map that binds markers to code fragments
	GetCodeFragments() CodeFragmentsMap
}

// HasDomain allows the domain to be used on a template
type HasDomain interface {
	// InjectDomain sets the template domain
	InjectDomain(string)
}

// HasRepository allows the repository to be used on a template
type HasRepository interface {
	// InjectRepository sets the template repository
	InjectRepository(string)
}

// HasMultiGroup allows the multi-group flag to be used on a template
type HasMultiGroup interface {
	// InjectMultiGroup sets the template multi-group flag
	InjectMultiGroup(bool)
}

// HasBoilerplate allows a boilerplate to be used on a template
type HasBoilerplate interface {
	// InjectBoilerplate sets the template boilerplate
	InjectBoilerplate(string)
}

// HasResource allows a resource to be used on a template
type HasResource interface {
	// InjectResource sets the template resource
	InjectResource(*resource.Resource)
}

// UseCustomFuncMap allows a template to use a custom template.FuncMap instead of the default FuncMap.
type UseCustomFuncMap interface {
	// GetFuncMap returns a custom FuncMap.
	GetFuncMap() template.FuncMap
}
