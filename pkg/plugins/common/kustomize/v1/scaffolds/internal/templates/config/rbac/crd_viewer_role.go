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

package rbac

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &CRDViewerRole{}

// CRDViewerRole scaffolds a file that defines the role that allows to view plurals
type CRDViewerRole struct {
	machinery.TemplateMixin
	machinery.ResourceMixin
}

// SetTemplateDefaults implements file.Template
func (f *CRDViewerRole) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "rbac", "%[kind]_viewer_role.yaml")
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.TemplateBody = crdRoleViewerTemplate

	return nil
}

const crdRoleViewerTemplate = `# permissions for end users to view {{ .Resource.Plural }}.
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{ lower .Resource.Kind }}-viewer-role
rules:
- apiGroups:
  - {{ .Resource.QualifiedGroup }}
  resources:
  - {{ .Resource.Plural }}
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - {{ .Resource.QualifiedGroup }}
  resources:
  - {{ .Resource.Plural }}/status
  verbs:
  - get
`
