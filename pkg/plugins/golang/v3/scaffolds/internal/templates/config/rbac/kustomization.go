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

var _ machinery.Template = &Kustomization{}

// Kustomization scaffolds a file that defines the kustomization scheme for the rbac folder
type Kustomization struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements file.Template
func (f *Kustomization) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "rbac", "kustomization.yaml")
	}

	f.TemplateBody = kustomizeRBACTemplate

	f.IfExistsAction = machinery.Error

	return nil
}

const kustomizeRBACTemplate = `resources:
# All RBAC will be applied under this service account in
# the deployment namespace. You may comment out this resource
# if your manager will use a service account that exists at
# runtime. Be sure to update RoleBinding and ClusterRoleBinding
# subjects if changing service account names.
- service_account.yaml
- role.yaml
- role_binding.yaml
- leader_election_role.yaml
- leader_election_role_binding.yaml
# Comment the following 4 lines if you want to disable
# the auth proxy (https://github.com/brancz/kube-rbac-proxy)
# which protects your /metrics endpoint.
- auth_proxy_service.yaml
- auth_proxy_role.yaml
- auth_proxy_role_binding.yaml
- auth_proxy_client_clusterrole.yaml
`
