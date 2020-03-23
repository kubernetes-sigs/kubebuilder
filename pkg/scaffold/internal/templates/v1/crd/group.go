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

var _ file.Template = &Group{}

// Group scaffolds the pkg/apis/group/group.go
type Group struct {
	file.TemplateMixin
	file.BoilerplateMixin
	file.ResourceMixin
}

// SetTemplateDefaults implements input.Template
func (f *Group) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("pkg", "apis", "%[group-package-name]", "group.go")
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.TemplateBody = groupTemplate

	return nil
}

const groupTemplate = `{{ .Boilerplate }}

// Package {{ .Resource.GroupPackageName }} contains {{ .Resource.Group }} API versions
package {{ .Resource.GroupPackageName }}
`
