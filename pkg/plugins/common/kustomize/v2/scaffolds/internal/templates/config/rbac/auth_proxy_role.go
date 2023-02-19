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

var _ machinery.Template = &AuthProxyRole{}

// AuthProxyRole scaffolds a file that defines the role for the auth proxy
type AuthProxyRole struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults implements file.Template
func (f *AuthProxyRole) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "rbac", "auth_proxy_role.yaml")
	}

	f.TemplateBody = proxyRoleTemplate

	return nil
}

const proxyRoleTemplate = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    app.kubernetes.io/name: clusterrole
    app.kubernetes.io/instance: proxy-role
    app.kubernetes.io/component: kube-rbac-proxy
    app.kubernetes.io/created-by: {{ .ProjectName }}
    app.kubernetes.io/part-of: {{ .ProjectName }}
    app.kubernetes.io/managed-by: kustomize
  name: proxy-role
rules:
- apiGroups:
  - authentication.k8s.io
  resources:
  - tokenreviews
  verbs:
  - create
- apiGroups:
  - authorization.k8s.io
  resources:
  - subjectaccessreviews
  verbs:
  - create
`
