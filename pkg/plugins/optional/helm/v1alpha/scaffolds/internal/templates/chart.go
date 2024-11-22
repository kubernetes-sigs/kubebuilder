/*
Copyright 2024 The Kubernetes Authors.

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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &HelmChart{}

// HelmChart scaffolds a file that defines the Helm chart structure
type HelmChart struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *HelmChart) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("dist", "chart", "Chart.yaml")
	}

	f.TemplateBody = helmChartTemplate

	f.IfExistsAction = machinery.SkipFile

	return nil
}

const helmChartTemplate = `apiVersion: v2
name: {{ .ProjectName }}
description: A Helm chart to distribute the project {{ .ProjectName }}
type: application
version: 0.1.0
appVersion: "0.1.0"
icon: "https://example.com/icon.png"
`
