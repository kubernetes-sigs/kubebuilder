/*
Copyright 2026 The Kubernetes Authors.

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

package scaffolds

import (
	"fmt"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/manager"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/rbac"
)

var _ plugins.Scaffolder = &editScaffolder{}

type editScaffolder struct {
	config     config.Config
	namespaced bool
	force      bool

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem
}

// NewEditScaffolder returns a new Scaffolder for configuration edit operations
func NewEditScaffolder(cfg config.Config, namespaced bool, force bool) plugins.Scaffolder {
	return &editScaffolder{
		config:     cfg,
		namespaced: namespaced,
		force:      force,
	}
}

// InjectFS implements plugins.Scaffolder
func (s *editScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements plugins.Scaffolder
func (s *editScaffolder) Scaffold() error {
	// Initialize the machinery.Scaffold
	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
	)

	var templates []machinery.Builder

	if s.namespaced {
		// Scaffold namespace-scoped RBAC and manager config
		templates = []machinery.Builder{
			&rbac.NamespacedRole{},
			&rbac.NamespacedRoleBinding{},
			&manager.Config{Image: imageName, Force: s.force},
		}
	} else {
		// Scaffold cluster-scoped RBAC and manager config
		templates = []machinery.Builder{
			&rbac.ClusterRole{},
			&rbac.ClusterRoleBinding{},
			&manager.Config{Image: imageName, Force: s.force},
		}
	}

	if err := scaffold.Execute(templates...); err != nil {
		return fmt.Errorf("failed to scaffold: %w", err)
	}

	// Regenerate CRD admin/editor/viewer roles for all existing resources
	// to match the new namespaced/cluster-scoped configuration
	resources, err := s.config.GetResources()
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}

	for _, res := range resources {
		if res.HasAPI() {
			// Create a scaffold with the resource injected for each resource
			resourceScaffold := machinery.NewScaffold(s.fs,
				machinery.WithConfig(s.config),
				machinery.WithResource(&res),
			)

			if err := resourceScaffold.Execute(
				&rbac.CRDAdminRole{},
				&rbac.CRDEditorRole{},
				&rbac.CRDViewerRole{},
			); err != nil {
				return fmt.Errorf("failed to scaffold CRD roles for %s: %w", res.Kind, err)
			}
		}
	}

	return nil
}
