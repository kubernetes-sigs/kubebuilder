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

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &KustomizeConfig{}

// KustomizeConfig  scaffolds a file that configures the kustomization for the crd folder
type KustomizeConfig struct {
	machinery.TemplateMixin
	machinery.ResourceMixin
}

// SetTemplateDefaults implements machinery.Template
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
    version: v1
    group: apiextensions.k8s.io
    path: spec/conversion/webhook/clientConfig/service/name

namespace:
- kind: CustomResourceDefinition
  version: v1
  group: apiextensions.k8s.io
  path: spec/conversion/webhook/clientConfig/service/namespace
  create: false

varReference:
- path: metadata/annotations
`
