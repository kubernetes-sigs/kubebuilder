/*
Copyright 2025 The Kubernetes Authors.

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
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &CustomGcl{}

// CustomGcl scaffolds a file ..custom-gcl.yaml which define KAL configuration to install
type CustomGcl struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// GolangciLintVersion is the golangci-lint version used to build the custom binary
	GolangciLintVersion string
}

// SetTemplateDefaults implements machinery.Template
func (f *CustomGcl) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = ".custom-gcl.yml"
	}

	f.TemplateBody = customGCLTemplate

	f.IfExistsAction = machinery.SkipFile

	return nil
}

const customGCLTemplate = `version: {{ .GolangciLintVersion }}
name: golangci-lint-kube-api
destination: ./bin

plugins:
  - module: sigs.k8s.io/kube-api-linter
    version: latest
`
