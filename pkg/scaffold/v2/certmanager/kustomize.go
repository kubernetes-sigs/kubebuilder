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

// Kustomization scaffolds the kustomizaiton in the certmanager folder
type Kustomization struct {
	input.Input
}

// GetInput implements input.File
func (p *Kustomization) GetInput() (input.Input, error) {
	if p.Path == "" {
		p.Path = filepath.Join("config", "certmanager", "kustomization.yaml")
	}
	p.TemplateBody = kustomizationTemplate
	return p.Input, nil
}

var kustomizationTemplate = `resources:
- certificate.yaml

# the following config is for teaching kustomize how to do var substitution
vars:
- name: NAMESPACE # namespace of the service and the certificate CR
  objref:
    kind: Service
    version: v1
    name: webhook-service
  fieldref:
    fieldpath: metadata.namespace
- name: CERTIFICATENAME
  objref:
    kind: Certificate
    group: certmanager.k8s.io
    version: v1alpha1
    name: serving-cert # this name should match the one in certificate.yaml
- name: SERVICENAME
  objref:
    kind: Service
    version: v1
    name: webhook-service

configurations:
- kustomizeconfig.yaml
`
