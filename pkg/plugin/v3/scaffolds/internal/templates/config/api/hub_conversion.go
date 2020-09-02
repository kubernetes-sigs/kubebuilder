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

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &HubConversion{}

// HubConversion scaffolds the Hub interface for conversion webhook
type HubConversion struct { // nolint:maligned
	file.TemplateMixin
	file.MultiGroupMixin
	file.BoilerplateMixin
	file.ResourceMixin

	HubVersion string
}

// SetTemplateDefaults implements input.Template
func (c *HubConversion) SetTemplateDefaults() error {
	c.Resource.Version = c.HubVersion
	if c.Path == "" {
		if c.MultiGroup {
			c.Path = filepath.Join("apis", "%[group]", "%[version]", "%[kind]_conversion.go")
		} else {
			c.Path = filepath.Join("api", "%[version]", "%[kind]_conversion.go")
		}
	}
	c.Path = c.Resource.Replacer().Replace(c.Path)
	fmt.Printf("Scaffolding conversion logic for hub version %s in path %s", c.Resource.Version, c.Path)
	c.TemplateBody = conversionTemplate
	c.IfExistsAction = file.Overwrite

	return nil
}

const conversionTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

// Implementing the Hub interface is straightforward -- only the no-op method 'Hub()' is required.
// See https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion?tab=doc#Hub.
func ({{ .Resource.Kind }}) Hub() {}

`
