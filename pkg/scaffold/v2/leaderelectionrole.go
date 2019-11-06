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

var _ input.File = &LeaderElectionRole{}

// LeaderElectionRole scaffolds the config/rbac/leader_election_role.yaml file
type LeaderElectionRole struct {
	input.Input
}

// GetInput implements input.File
func (f *LeaderElectionRole) GetInput() (input.Input, error) {
	if f.Path == "" {
		f.Path = filepath.Join("config", "rbac", "leader_election_role.yaml")
	}
	f.TemplateBody = leaderElectionRoleTemplate
	return f.Input, nil
}

const leaderElectionRoleTemplate = `# permissions to do leader election.
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: leader-election-role
rules:
- apiGroups:
  - ""
  resources:
  - configmaps
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
- apiGroups:
  - ""
  resources:
  - configmaps/status
  verbs:
  - get
  - update
  - patch
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create
`
