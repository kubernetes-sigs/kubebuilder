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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &SpokeConversion{}

// SpokeConversion scaffolds the methods of Convertible interface for conversion webhook
type SpokeConversion struct { // nolint:maligned
	file.TemplateMixin
	file.MultiGroupMixin
	file.BoilerplateMixin
	file.ResourceMixin

	SpokeVersion string
}

// SetTemplateDefaults implements input.Template
func (c *SpokeConversion) SetTemplateDefaults() error {
	c.Resource.Version = c.SpokeVersion
	if c.Path == "" {
		if c.MultiGroup {
			c.Path = filepath.Join("apis", "%[group]", "%[version]", "%[kind]_conversion.go")
		} else {
			c.Path = filepath.Join("api", "%[version]", "%[kind]_conversion.go")
		}
	}
	c.Path = c.Resource.Replacer().Replace(c.Path)
	c.TemplateBody = spokeConversionTemplate
	c.IfExistsAction = file.Overwrite

	return nil
}

const spokeConversionTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

func (src  *{{ .Resource.Kind }}) ConvertTo(dstRaw conversion.Hub) error {
	// Implement your logic here to convert from hub to spoke version.
	return nil
}

func (dst *{{ .Resource.Kind }}) ConvertFrom(srcRaw conversion.Hub) error {
	// Implement your logic here to convert from spoke to hub version.
	return nil
}
`
