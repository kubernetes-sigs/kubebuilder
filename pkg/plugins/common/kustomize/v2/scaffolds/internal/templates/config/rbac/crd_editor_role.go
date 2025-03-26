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

//nolint:dupl
package rbac

import (
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &CRDEditorRole{}

// CRDEditorRole scaffolds a file that defines the role that allows to edit plurals
type CRDEditorRole struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.ResourceMixin
	machinery.ProjectNameMixin

	RoleName string
}

// SetTemplateDefaults implements machinery.Template
func (f *CRDEditorRole) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup && f.Resource.Group != "" {
			f.Path = filepath.Join("config", "rbac", "%[group]_%[kind]_editor_role.yaml")
		} else {
			f.Path = filepath.Join("config", "rbac", "%[kind]_editor_role.yaml")
		}
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	if f.RoleName == "" {
		if f.MultiGroup && f.Resource.Group != "" {
			f.RoleName = fmt.Sprintf("%s-%s-editor-role",
				strings.ToLower(f.Resource.Group),
				strings.ToLower(f.Resource.Kind))
		} else {
			f.RoleName = fmt.Sprintf("%s-editor-role",
				strings.ToLower(f.Resource.Kind))
		}
	}

	f.TemplateBody = crdRoleEditorTemplate

	return nil
}

const crdRoleEditorTemplate = `# This rule is not used by the project {{ .ProjectName }} itself.
# It is provided to allow the cluster admin to help manage permissions for users.
#
# Grants permissions to create, update, and delete resources within the {{ .Resource.QualifiedGroup }}.
# This role is intended for users who need to manage these resources
# but should not control RBAC or manage permissions for others.

apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    app.kubernetes.io/name: {{ .ProjectName }}
    app.kubernetes.io/managed-by: kustomize
  name: {{ .RoleName }}
rules:
- apiGroups:
  - {{ .Resource.QualifiedGroup }}
  resources:
  - {{ .Resource.Plural }}
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - {{ .Resource.QualifiedGroup }}
  resources:
  - {{ .Resource.Plural }}/status
  verbs:
  - get
`
