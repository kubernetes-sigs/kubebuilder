/*
Copyright 2020 The Kubernetes Authors.

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
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &Conversion{}

// Conversion scaffolds the methods of Hub interface for conversion webhook
type Conversion struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	machinery.MultiGroupMixin
	Force bool
	// Hub and spoke indicate the template to be scaffolded
	Hub   bool
	Spoke bool
}

// SetTemplateDefaults implements file.Template
func (c *Conversion) SetTemplateDefaults() error {
	if c.Path == "" {
		c.Path = filepath.Join("api", "%[version]", "%[kind]_conversion.go")
		if c.MultiGroup && c.Resource.Group != "" {
			c.Path = filepath.Join("apis", "%[group]", "%[version]", "%[kind]_conversion.go")
		}
	}

	c.Path = c.Resource.Replacer().Replace(c.Path)
	fmt.Println(c.Path)

	c.TemplateBody = conversionTemplate

	return nil
}

// GetIfExistsAction implements machinery.Builder
func (c Conversion) GetIfExistsAction() machinery.IfExistsAction {
	if c.Force {
		return machinery.OverwriteFile
	}
	return machinery.SkipFile
}

const conversionTemplate = `{{ .Boilerplate }}

{{- if .Hub }}
package {{ .Resource.Version }}
// Hub marks that a given type is the hub type for conversion. -- only the no-op method 'Hub()' is required.
// See https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion#Hub 
// or https://book.kubebuilder.io/multiversion-tutorial/conversion.html.
func ({{ .Resource.Kind }}) Hub() {}
{{else}}{{- if .Spoke }}

package {{ .Resource.Version }}

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// See https://book.kubebuilder.io/multiversion-tutorial/conversion.html. 
func (src  *{{ .Resource.Kind }}) ConvertTo(dstRaw conversion.Hub) error {
	// Implement your logic here to convert from hub to spoke version.
	return nil
}
func (dst *{{ .Resource.Kind }}) ConvertFrom(srcRaw conversion.Hub) error {
	// Implement your logic here to convert from spoke to hub version.
	return nil
}
{{end}}
{{end}}
`
