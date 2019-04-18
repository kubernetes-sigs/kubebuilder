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

package certmanager

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

// KustomizeConfig scaffolds the kustomizeconfig in the certmanager folder
type KustomizeConfig struct {
	input.Input
}

// GetInput implements input.File
func (p *KustomizeConfig) GetInput() (input.Input, error) {
	if p.Path == "" {
		p.Path = filepath.Join("config", "certmanager", "kustomizeconfig.yaml")
	}
	p.TemplateBody = kustomizeConfigTemplate
	return p.Input, nil
}

var kustomizeConfigTemplate = `# This configuration is for teaching kustomize how to update name ref and var substitution 
nameReference:
- kind: Issuer
  group: certmanager.k8s.io
  fieldSpecs:
  - kind: Certificate
    group: certmanager.k8s.io
    path: spec/issuerRef/name

varReference:
- kind: Certificate
  group: certmanager.k8s.io
  path: spec/commonName
- kind: Certificate
  group: certmanager.k8s.io
  path: spec/dnsNames
`
