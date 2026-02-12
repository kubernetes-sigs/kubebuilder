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
	log "log/slog"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	kustomizecommonv2 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/hack"
)

var _ plugins.Scaffolder = &editScaffolder{}

type editScaffolder struct {
	config      config.Config
	multigroup  bool
	namespaced  bool
	force       bool
	license     string
	owner       string
	licenseFile string

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem
}

// NewEditScaffolder returns a new Scaffolder for configuration edit operations
func NewEditScaffolder(cfg config.Config, multigroup bool, namespaced bool, force bool,
	license, owner, licenseFile string,
) plugins.Scaffolder {
	return &editScaffolder{
		config:      cfg,
		multigroup:  multigroup,
		namespaced:  namespaced,
		force:       force,
		license:     license,
		owner:       owner,
		licenseFile: licenseFile,
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

	// Update boilerplate if license flags are provided.
	// This allows users to change the license header after project initialization.
	if s.license != "" || s.licenseFile != "" {
		if updateErr := s.updateBoilerplate(); updateErr != nil {
			return fmt.Errorf("failed to update boilerplate: %w", updateErr)
		}
	}

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

	// Scaffold appropriate RBAC and manager config based on namespaced flag
	if s.namespaced && !wasNamespaced {
		// Switching to namespaced layout: scaffold Role/RoleBinding and WATCH_NAMESPACE
		if rbacErr := s.scaffoldNamespacedRBAC(s.force); rbacErr != nil {
			return fmt.Errorf("failed to scaffold namespaced RBAC: %w", rbacErr)
		}

		if !s.force {
			fmt.Println()
			fmt.Println("Run with --force to update config/manager/manager.yaml with WATCH_NAMESPACE")
		}

		// Print next steps
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("1. Update cmd/main.go to configure namespace-scoped cache")
		fmt.Println("2. Add namespace= to RBAC markers in existing controllers:")
		fmt.Printf("   // +kubebuilder:rbac:groups=mygroup,resources=myresources,verbs=get;list,"+
			"namespace=%s-system\n", s.config.GetProjectName())
		fmt.Println("3. Run: make manifests")
		fmt.Println()
		fmt.Println("See: https://book.kubebuilder.io/migration/namespace-scoped.html")
	} else if !s.namespaced && wasNamespaced {
		// Switching to cluster-scoped layout: scaffold ClusterRole/ClusterRoleBinding
		if rbacErr := s.scaffoldClusterRBAC(s.force); rbacErr != nil {
			return fmt.Errorf("failed to scaffold cluster-scoped RBAC: %w", rbacErr)
		}

		if !s.force {
			fmt.Println()
			fmt.Println("Run with --force to update config/manager/manager.yaml (remove WATCH_NAMESPACE)")
		}

		// Print next steps
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("1. Update cmd/main.go:")
		fmt.Println("   - Remove getWatchNamespace() and setupCacheNamespaces() functions")
		fmt.Println("   - Remove watchNamespace retrieval and cache configuration")
		fmt.Println("2. Remove namespace= from RBAC markers in existing controllers")
		fmt.Println("3. Run: make manifests")
		fmt.Println()
		fmt.Println("See: https://book.kubebuilder.io/migration/namespace-scoped.html")
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

func (s *editScaffolder) scaffoldNamespacedRBAC(force bool) error {
	// Use the kustomize/v2 scaffolder to scaffold namespace-scoped RBAC and manager config
	rbacScaffolder := kustomizecommonv2.NewEditScaffolder(s.config, true, force)
	rbacScaffolder.InjectFS(s.fs)
	if err := rbacScaffolder.Scaffold(); err != nil {
		return fmt.Errorf("failed to scaffold RBAC: %w", err)
	}
	return nil
}

func (s *editScaffolder) scaffoldClusterRBAC(force bool) error {
	// Use the kustomize/v2 scaffolder to scaffold cluster-scoped RBAC and manager config
	rbacScaffolder := kustomizecommonv2.NewEditScaffolder(s.config, false, force)
	rbacScaffolder.InjectFS(s.fs)
	if err := rbacScaffolder.Scaffold(); err != nil {
		return fmt.Errorf("failed to scaffold RBAC: %w", err)
	}
	return nil
}

func (s *editScaffolder) updateBoilerplate() error {
	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
	)

	bpFile := &hack.Boilerplate{
		License: s.license,
		Owner:   s.owner,
	}

	// If a custom license file is provided, read its content
	if s.licenseFile != "" {
		content, err := afero.ReadFile(afero.NewOsFs(), s.licenseFile)
		if err != nil {
			return fmt.Errorf("failed to read license file %q: %w", s.licenseFile, err)
		}
		bpFile.CustomBoilerplateContent = string(content)
		bpFile.HasCustomBoilerplate = true
		log.Info("Updating boilerplate with custom license file", "file", s.licenseFile)
	} else if s.license != "" {
		log.Info("Updating boilerplate with license", "license", s.license)
	}

	bpFile.Path = hack.DefaultBoilerplatePath
	if err := scaffold.Execute(bpFile); err != nil {
		return fmt.Errorf("failed to update boilerplate: %w", err)
	}

	log.Info("License header updated successfully", "path", hack.DefaultBoilerplatePath)
	return nil
}
