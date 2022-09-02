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

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &GoMod{}

// GoMod scaffolds a file that defines the project dependencies
type GoMod struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.RepositoryMixin

	ModuleName               string
	ControllerRuntimeVersion string
}

// SetTemplateDefaults implements file.Template
func (f *GoMod) SetTemplateDefaults() error {
	f.TemplateBody = goModTemplate
	f.IfExistsAction = machinery.SkipFile
	fmt.Printf("using module name %s\n", f.ModuleName)
	return nil
}

const goModTemplate = `
module {{ .ModuleName }}

go 1.18

require (
	sigs.k8s.io/controller-runtime {{ .ControllerRuntimeVersion }}
)
`
