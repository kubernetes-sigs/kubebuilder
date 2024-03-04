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

package rbac

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &ServiceAccount{}

// ServiceAccount scaffolds a file that defines the service account the manager is deployed in.
type ServiceAccount struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults implements file.Template
func (f *ServiceAccount) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "rbac", "service_account.yaml")
	}

	f.TemplateBody = serviceAccountTemplate

	return nil
}

const serviceAccountTemplate = `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/name: {{ .ProjectName }}
    app.kubernetes.io/managed-by: kustomize
  name: controller-manager
  namespace: system
`
