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

package metricsauth

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &ClientClusterRole{}

// ClientClusterRole scaffolds the config/rbac/client_clusterrole.yaml file
type ClientClusterRole struct {
	file.TemplateMixin
}

// SetTemplateDefaults implements input.Template
func (f *ClientClusterRole) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "rbac", "auth_proxy_client_clusterrole.yaml")
	}

	f.TemplateBody = clientClusterRoleTemplate

	return nil
}

const clientClusterRoleTemplate = `apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: metrics-reader
rules:
- nonResourceURLs: ["/metrics"]
  verbs: ["get"]
`
