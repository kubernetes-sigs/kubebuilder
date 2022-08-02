/*
Copyright 2022 The Kubernetes Authors.
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

package prometheus_rbac

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &Role{}

// Role scaffolds a file that defines the Prometheus Role
type Role struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements file.Template
func (f *Role) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "prometheus_rbac", "role.yaml")
	}

	f.TemplateBody = roleTemplate

	return nil
}

const roleTemplate = `
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: controller-manager-cluster-role
  namespace: system
rules:
  - apiGroups: [""]
    resources:
      - services
      - endpoints
      - pods
    verbs: ["get", "list"]
`
