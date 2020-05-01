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

package crd

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &KustomizeConfig{}

// KustomizeConfig scaffolds the kustomizeconfig file in crd folder.
type KustomizeConfig struct {
	file.TemplateMixin
}

// SetTemplateDefaults implements input.Template
func (f *KustomizeConfig) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "crd", "kustomizeconfig.yaml")
	}

	f.TemplateBody = kustomizeConfigTemplate

	return nil
}

//nolint:lll
const kustomizeConfigTemplate = `# This file is for teaching kustomize how to substitute name and namespace reference in CRD
nameReference:
- kind: Service
  version: v1
  fieldSpecs:
  - kind: CustomResourceDefinition
    group: apiextensions.k8s.io
    path: spec/conversion/webhookClientConfig/service/name

namespace:
- kind: CustomResourceDefinition
  group: apiextensions.k8s.io
  path: spec/conversion/webhookClientConfig/service/namespace
  create: false

varReference:
- path: metadata/annotations
`
