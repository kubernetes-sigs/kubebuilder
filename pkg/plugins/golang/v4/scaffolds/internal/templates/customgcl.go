/*
Copyright 2026 The Kubernetes Authors.

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

// CustomGcl scaffolds the .custom-gcl.yml file for golangci-lint module plugins
type CustomGcl struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// GolangciLintVersion is the version of golangci-lint to use
	GolangciLintVersion string
}

// SetTemplateDefaults implements machinery.Template
func (f *CustomGcl) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = ".custom-gcl.yml"
	}

	f.TemplateBody = customGclTemplate

	f.IfExistsAction = machinery.SkipFile

	return nil
}

const customGclTemplate = `# This file configures golangci-lint with module plugins.
# When you run 'make lint', it will automatically build a custom golangci-lint binary
# with all the plugins listed below.
#
# See: https://golangci-lint.run/plugins/module-plugins/
version: {{ .GolangciLintVersion }}
name: golangci-lint-custom
destination: ./bin

plugins:
  # logcheck validates structured logging calls and parameters (e.g., balanced key-value pairs)
  - module: "sigs.k8s.io/logtools"
    import: "sigs.k8s.io/logtools/logcheck/gclplugin"
    version: latest 
  # kube-api-linter checks Kubernetes API conventions
  - module: "sigs.k8s.io/kube-api-linter"
    version: latest
`
