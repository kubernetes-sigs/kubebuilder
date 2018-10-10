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

package resource

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
)

var _ input.File = &RoleBinding{}

// Role scaffolds the config/manager/group_role_rbac.yaml file
type Role struct {
	input.Input

	// Resource is a resource in the API group
	Resource *Resource
}

// GetInput implements input.File
func (r *Role) GetInput() (input.Input, error) {
	if r.Path == "" {
		r.Path = filepath.Join("config", "manager", fmt.Sprintf(
			"%s_role_rbac.yaml", r.Resource.Group))
	}
	r.TemplateBody = roleTemplate
	return r.Input, nil
}

var roleTemplate = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: {{.Resource.Group}}-role
rules:
- apiGroups:
  - {{ .Resource.Group }}.{{ .Domain }}
  resources:
  - '*'
  verbs:
  - '*'

`
