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

package kdefault

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &ManagerMetricsPatch{}

// ManagerMetricsPatch scaffolds a file that defines the patch that enables prometheus metrics for the manager
type ManagerMetricsPatch struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *ManagerMetricsPatch) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "default", "manager_metrics_patch.yaml")
	}

	f.TemplateBody = kustomizeMetricsPatchTemplate

	f.IfExistsAction = machinery.Error

	return nil
}

const kustomizeMetricsPatchTemplate = `# This patch adds the args to allow exposing the metrics endpoint using HTTPS
- op: add
  path: /spec/template/spec/containers/0/args/0
  value: --metrics-bind-address=:8443
`
