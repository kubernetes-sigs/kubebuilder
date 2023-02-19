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

var _ machinery.Template = &AuthProxyClientRole{}

// AuthProxyClientRole scaffolds a file that defines the role for the metrics reader
type AuthProxyClientRole struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults implements file.Template
func (f *AuthProxyClientRole) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "rbac", "auth_proxy_client_clusterrole.yaml")
	}

	f.TemplateBody = clientClusterRoleTemplate

	return nil
}

const clientClusterRoleTemplate = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    app.kubernetes.io/name: clusterrole
    app.kubernetes.io/instance: metrics-reader
    app.kubernetes.io/component: kube-rbac-proxy
    app.kubernetes.io/created-by: {{ .ProjectName }}
    app.kubernetes.io/part-of: {{ .ProjectName }}
    app.kubernetes.io/managed-by: kustomize
  name: metrics-reader
rules:
- nonResourceURLs:
  - "/metrics"
  verbs:
  - get
`
