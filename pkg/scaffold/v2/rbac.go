/*
Copyright 2019 The Kubernetes Authors.

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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

var _ input.File = &KustomizeRBAC{}

// KustomizeRBAC scaffolds the Kustomization file in rbac folder.
type KustomizeRBAC struct {
	input.Input
}

// GetInput implements input.File
func (c *KustomizeRBAC) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = filepath.Join("deploy", "rbac", "kustomization.yaml")
	}
	c.TemplateBody = kustomizeRBACTemplate
	c.Input.IfExistsAction = input.Error
	return c.Input, nil
}

var kustomizeRBACTemplate = `resources:
- role.yaml
- role_binding.yaml
- leader_election_role.yaml
- leader_election_role_binding.yaml
# Comment the following 3 lines if you want to disable
# the auth proxy (https://github.com/brancz/kube-rbac-proxy)
# which protects your /metrics endpoint.
- auth_proxy_service.yaml
- auth_proxy_role.yaml
- auth_proxy_role_binding.yaml
`
