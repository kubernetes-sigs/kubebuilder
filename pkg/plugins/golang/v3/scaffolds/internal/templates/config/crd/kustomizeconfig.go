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

package crd

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v2/pkg/model/file"
)

const v1 = "v1"

var _ file.Template = &KustomizeConfig{}

// KustomizeConfig  scaffolds a file that configures the kustomization for the crd folder
type KustomizeConfig struct {
	file.TemplateMixin

	// Version of CRD patch generated.
	CRDVersion string
}

// SetTemplateDefaults implements file.Template
func (f *KustomizeConfig) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "crd", "kustomizeconfig.yaml")
	}

	f.TemplateBody = kustomizeConfigTemplate

	if f.CRDVersion == "" {
		f.CRDVersion = v1
	}

	return nil
}

//nolint:lll
const kustomizeConfigTemplate = `# This file is for teaching kustomize how to substitute name and namespace reference in CRD
nameReference:
- kind: Service
  version: v1
  fieldSpecs:
  - kind: CustomResourceDefinition
    version: {{ .CRDVersion }}
    group: apiextensions.k8s.io
    {{- if ne .CRDVersion "v1" }}
    path: spec/conversion/webhookClientConfig/service/name
    {{- else }}
    path: spec/conversion/webhook/clientConfig/service/name
    {{- end }}

namespace:
- kind: CustomResourceDefinition
  version: {{ .CRDVersion }}
  group: apiextensions.k8s.io
  {{- if ne .CRDVersion "v1" }}
  path: spec/conversion/webhookClientConfig/service/namespace
  {{- else }}
  path: spec/conversion/webhook/clientConfig/service/namespace
  {{- end }}
  create: false

varReference:
- path: metadata/annotations
`
