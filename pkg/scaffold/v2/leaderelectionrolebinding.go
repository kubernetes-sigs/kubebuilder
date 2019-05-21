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

package v2

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

var _ input.File = &LeaderElectionRoleBinding{}

// LeaderElectionRoleBinding scaffolds the config/rbac/leader_election_role_binding.yaml file
type LeaderElectionRoleBinding struct {
	input.Input
}

// GetInput implements input.File
func (r *LeaderElectionRoleBinding) GetInput() (input.Input, error) {
	if r.Path == "" {
		r.Path = filepath.Join("config", "rbac", "leader_election_role_binding.yaml")
	}
	r.TemplateBody = leaderElectionRoleBindingTemplate
	return r.Input, nil
}

var leaderElectionRoleBindingTemplate = `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: leader-election-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: leader-election-role
subjects:
- kind: ServiceAccount
  name: default
  namespace: system
`
