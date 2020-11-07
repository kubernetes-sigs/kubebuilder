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

package templates

import (
	"sigs.k8s.io/kubebuilder/v2/pkg/model/file"
)

var _ file.Template = &GoMod{}

// GoMod scaffolds a file that defines the project dependencies
type GoMod struct {
	file.TemplateMixin
	file.RepositoryMixin

	ControllerRuntimeVersion string
}

// SetTemplateDefaults implements file.Template
func (f *GoMod) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = "go.mod"
	}

	f.TemplateBody = goModTemplate

	f.IfExistsAction = file.Overwrite

	return nil
}

const goModTemplate = `
module {{ .Repo }}

go 1.15

require (
	sigs.k8s.io/controller-runtime {{ .ControllerRuntimeVersion }}
)
`
