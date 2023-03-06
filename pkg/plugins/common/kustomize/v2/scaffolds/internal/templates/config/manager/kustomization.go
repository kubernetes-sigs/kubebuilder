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

package manager

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &Kustomization{}

// Kustomization scaffolds a file that defines the kustomization scheme for the manager folder
type Kustomization struct {
	machinery.TemplateMixin
	machinery.ComponentConfigMixin
}

// SetTemplateDefaults implements file.Template
func (f *Kustomization) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "manager", "kustomization.yaml")
	}

	f.TemplateBody = kustomizeManagerTemplate

	f.IfExistsAction = machinery.Error

	return nil
}

const kustomizeManagerTemplate = `resources:
- manager.yaml

{{- if .ComponentConfig }}

generatorOptions:
  disableNameSuffixHash: true

configMapGenerator:
- name: manager-config
  files:
  - controller_manager_config.yaml
{{- end }}
`
