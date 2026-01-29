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

package scaffolds

import (
	"fmt"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	kustomizecommonv2 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds"
)

var _ plugins.Scaffolder = &editScaffolder{}

type editScaffolder struct {
	config     config.Config
	multigroup bool
	namespaced bool

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem
}

// NewEditScaffolder returns a new Scaffolder for configuration edit operations
func NewEditScaffolder(cfg config.Config, multigroup bool, namespaced bool) plugins.Scaffolder {
	return &editScaffolder{
		config:     cfg,
		multigroup: multigroup,
		namespaced: namespaced,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *editScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements cmdutil.Scaffolder
func (s *editScaffolder) Scaffold() error {
	filename := "Dockerfile"
	bs, err := afero.ReadFile(s.fs.FS, filename)
	if err != nil {
		return fmt.Errorf("error reading %q: %w", filename, err)
	}
	str := string(bs)

	// Track if we're toggling namespaced mode
	wasNamespaced := s.config.IsNamespaced()

	// Update config flags
	if s.multigroup {
		_ = s.config.SetMultiGroup()
	} else {
		_ = s.config.ClearMultiGroup()
	}

	if s.namespaced {
		_ = s.config.SetNamespaced()
	} else {
		_ = s.config.ClearNamespaced()
	}

	// Scaffold appropriate RBAC based on namespaced flag
	if s.namespaced && !wasNamespaced {
		// Switching to namespace-scoped: scaffold Role/RoleBinding
		if rbacErr := s.scaffoldNamespacedRBAC(); rbacErr != nil {
			return fmt.Errorf("failed to scaffold namespace-scoped RBAC: %w", rbacErr)
		}
		// Print warning about manual steps required
		fmt.Println("\nManual steps required:")
		fmt.Println("- Add namespace= parameter to RBAC markers in existing controller files")
		fmt.Println("- Run 'make manifests' to regenerate RBAC")
		fmt.Printf("See: https://book.kubebuilder.io/migration/namespace-scoped\n")
	} else if !s.namespaced && wasNamespaced {
		// Switching to cluster-scoped: scaffold ClusterRole/ClusterRoleBinding
		if rbacErr := s.scaffoldClusterRBAC(); rbacErr != nil {
			return fmt.Errorf("failed to scaffold cluster-scoped RBAC: %w", rbacErr)
		}
		// Print warning about manual steps required
		fmt.Println("\nManual steps required:")
		fmt.Println("- Remove namespace= parameter from RBAC markers in controller files")
		fmt.Println("- Remove namespace watching code from cmd/main.go")
		fmt.Println("- Run 'make manifests' to regenerate RBAC")
	}

	// Check if the str is not empty, because when the file is already in desired format it will return empty string
	// because there is nothing to replace.
	if str != "" {
		// TODO: instead of writing it directly, we should use the scaffolding machinery for consistency
		if err = afero.WriteFile(s.fs.FS, filename, []byte(str), 0o644); err != nil {
			return fmt.Errorf("error writing %q: %w", filename, err)
		}
	}

	return nil
}

func (s *editScaffolder) scaffoldNamespacedRBAC() error {
	// Use the kustomize/v2 scaffolder to scaffold namespace-scoped RBAC
	rbacScaffolder := kustomizecommonv2.NewEditScaffolder(s.config, true)
	rbacScaffolder.InjectFS(s.fs)
	if err := rbacScaffolder.Scaffold(); err != nil {
		return fmt.Errorf("failed to scaffold RBAC: %w", err)
	}
	return nil
}

func (s *editScaffolder) scaffoldClusterRBAC() error {
	// Use the kustomize/v2 scaffolder to scaffold cluster-scoped RBAC
	rbacScaffolder := kustomizecommonv2.NewEditScaffolder(s.config, false)
	rbacScaffolder.InjectFS(s.fs)
	if err := rbacScaffolder.Scaffold(); err != nil {
		return fmt.Errorf("failed to scaffold RBAC: %w", err)
	}
	return nil
}
