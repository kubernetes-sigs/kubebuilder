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

package webhook

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

var _ input.File = &Kustomization{}

// Kustomization scaffolds the Kustomization file in manager folder.
type Kustomization struct {
	input.Input
}

// GetInput implements input.File
func (c *Kustomization) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = filepath.Join("config", "webhook", "kustomization.yaml")
	}
	c.TemplateBody = KustomizeWebhookTemplate
	c.Input.IfExistsAction = input.Error
	return c.Input, nil
}

var KustomizeWebhookTemplate = `resources:
- manifests.yaml
- service.yaml

configurations:
- kustomizeconfig.yaml

# the following config is for teaching kustomize how to do var substitution
vars:
- name: NAMESPACE
  objref:
    kind: Service
    version: v1
    name: webhook-service
  fieldref:
    fieldpath: metadata.namespace
- name: SERVICENAME
  objref:
    kind: Service
    version: v1
    name: webhook-service
`
