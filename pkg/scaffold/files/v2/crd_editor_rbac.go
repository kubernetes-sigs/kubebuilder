/*
Copyright 2018 The Kubernetes Authors.

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

package v2

import (
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
)

var _ input.File = &CRDEditorRole{}

// CRD Editor role scaffolds the config/rbca/<kind>_editor_role.yaml
type CRDEditorRole struct {
	input.Input

	// Resource is a resource in the API group
	Resource *resource.Resource
}

// GetInput implements input.File
func (g *CRDEditorRole) GetInput() (input.Input, error) {
	if g.Path == "" {
		g.Path = filepath.Join("config", "rbac", fmt.Sprintf("%s_editor_role.yaml", strings.ToLower(g.Resource.Kind)))
	}

	g.TemplateBody = crdRoleEditorTemplate
	return g.Input, nil
}

// Validate validates the values
func (g *CRDEditorRole) Validate() error {
	return g.Resource.Validate()
}

const crdRoleEditorTemplate = `# permissions for end users to edit {{ .Resource.Resource }}.
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{ lower .Resource.Kind }}-editor-role
rules:
- apiGroups:
  - {{ .Resource.Group }}.{{ .Domain }}
  resources:
  - {{ .Resource.Resource }}
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - {{ .Resource.Group }}.{{ .Domain }}
  resources:
  - {{ .Resource.Resource }}/status
  verbs:
  - get
`
