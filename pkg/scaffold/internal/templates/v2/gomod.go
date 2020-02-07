/*
Copyright 2019 The Kubernetes Authors.

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

package v2

import (
	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &GoMod{}

// GoMod writes a templatefile for go.mod
type GoMod struct {
	file.Input
	file.RepositoryMixin

	ControllerRuntimeVersion string
}

// GetInput implements input.Template
func (f *GoMod) GetInput() (file.Input, error) {
	if f.Path == "" {
		f.Path = "go.mod"
	}
	f.Input.IfExistsAction = file.Overwrite
	f.TemplateBody = goModTemplate
	return f.Input, nil
}

const goModTemplate = `
module {{ .Repo }}

go 1.13

require (
	sigs.k8s.io/controller-runtime {{ .ControllerRuntimeVersion }}
)
`
