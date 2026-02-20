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
	"errors"
	"fmt"
	log "log/slog"
	"path/filepath"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	kustomizev2scaffolds "sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/server-side-apply/v1alpha1/scaffolds/internal/templates/controllers"
	golangv4scaffolds "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds"
)

var _ plugins.Scaffolder = &apiScaffolder{}

// apiScaffolder contains configuration for generating scaffolding with Server-Side Apply
type apiScaffolder struct {
	config   config.Config
	resource resource.Resource

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem
}

// NewAPIScaffolder returns a new Scaffolder for Server-Side Apply APIs
func NewAPIScaffolder(cfg config.Config, res resource.Resource) plugins.Scaffolder {
	return &apiScaffolder{
		config:   cfg,
		resource: res,
	}
}

// InjectFS implements plugins.Scaffolder
func (s *apiScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements plugins.Scaffolder
func (s *apiScaffolder) Scaffold() error {
	log.Info("writing scaffold for you to edit...")

	// 1. Scaffold standard API using golang/v4 (reuse completely)
	if err := s.scaffoldStandardAPI(); err != nil {
		return err
	}

	// 2. Load boilerplate for templates
	boilerplatePath := filepath.Join("hack", "boilerplate.go.txt")
	boilerplate, err := afero.ReadFile(s.fs.FS, boilerplatePath)
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			log.Warn("unable to find boilerplate file. "+
				"This file is used to generate the license header in the project.\n"+
				"Note that controller-gen will also use this. Ensure that you "+
				"add the license file or configure your project accordingly",
				"file_path", boilerplatePath, "error", err)
			boilerplate = []byte("")
		} else {
			return fmt.Errorf("error scaffolding API/controller: failed to load boilerplate: %w", err)
		}
	}

	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
		machinery.WithBoilerplate(string(boilerplate)),
		machinery.WithResource(&s.resource),
	)

	// 3. Add applyconfiguration generation marker to groupversion_info.go
	if err := s.updateGroupVersionInfo(); err != nil {
		return fmt.Errorf("error updating groupversion_info.go: %w", err)
	}

	// 4. Scaffold Server-Side Apply controller (custom)
	if err := scaffold.Execute(
		&controllers.Controller{
			ControllerRuntimeVersion: golangv4scaffolds.ControllerRuntimeVersion,
		},
	); err != nil {
		return fmt.Errorf("error scaffolding controller: %w", err)
	}

	// 4. Update Makefile to add applyconfiguration generation
	if err := s.updateMakefile(); err != nil {
		return fmt.Errorf("error updating Makefile: %w", err)
	}

	// 5. Update .gitignore to exclude pkg/applyconfiguration/
	if err := s.updateGitignore(); err != nil {
		return fmt.Errorf("error updating .gitignore: %w", err)
	}

	return nil
}

// scaffoldStandardAPI scaffolds the standard API using golang/v4 and kustomize/v2
func (s *apiScaffolder) scaffoldStandardAPI() error {
	// Reuse golang/v4 API scaffolder for types
	golangScaffolder := golangv4scaffolds.NewAPIScaffolder(s.config, s.resource, true)
	golangScaffolder.InjectFS(s.fs)
	if err := golangScaffolder.Scaffold(); err != nil {
		return fmt.Errorf("error scaffolding golang files: %w", err)
	}

	// Reuse kustomize/v2 scaffolder for CRD/RBAC
	kustomizeScaffolder := kustomizev2scaffolds.NewAPIScaffolder(s.config, s.resource, true)
	kustomizeScaffolder.InjectFS(s.fs)
	if err := kustomizeScaffolder.Scaffold(); err != nil {
		return fmt.Errorf("error scaffolding kustomize files: %w", err)
	}

	return nil
}

// updateMakefile adds or updates the APPLYCONFIGURATION_PATHS variable and generation command
func (s *apiScaffolder) updateMakefile() error {
	makefilePath := "Makefile"

	// Build the API path for this resource
	apiPath := s.getAPIPath()

	// Check if the applyconfiguration scaffold marker exists
	err := util.ReplaceInFile(makefilePath,
		"# +kubebuilder:scaffold:applyconfiguration-paths",
		"# +kubebuilder:scaffold:applyconfiguration-paths")

	if err != nil {
		// First API using this plugin - need to add the variable and generation command
		log.Info("adding applyconfiguration generation to Makefile")

		// Add the variable before the generate target
		variableMarker := "\n# +kubebuilder:scaffold:applyconfiguration-paths\n" +
			fmt.Sprintf("APPLYCONFIGURATION_PATHS ?= %s\n", apiPath)

		if err := util.InsertCode(makefilePath,
			".PHONY: generate",
			variableMarker); err != nil {
			return fmt.Errorf("error adding APPLYCONFIGURATION_PATHS variable: %w", err)
		}

		// Add the applyconfiguration generation BEFORE object generation
		// This ensures the package exists before controllers import it
		// Note: controller-gen applyconfiguration creates files relative to the API package
		// For example, api/v1 will generate api/v1/applyconfiguration/
		genCommand := `# +kubebuilder:scaffold:applyconfiguration-gen
	"$(CONTROLLER_GEN)" applyconfiguration:headerFile="hack/boilerplate.go.txt" \
		paths="$(APPLYCONFIGURATION_PATHS)"
	`

		// Find the generate target and add at the beginning
		//nolint:lll
		generateTarget := `generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.`
		if err := util.InsertCode(makefilePath, generateTarget, genCommand); err != nil {
			return fmt.Errorf("error adding applyconfiguration generation: %w", err)
		}
	} else {
		// Variable already exists - append this API path
		log.Info("updating APPLYCONFIGURATION_PATHS in Makefile")

		// Find the current value and append
		if err := util.ReplaceInFile(makefilePath,
			"APPLYCONFIGURATION_PATHS ?= ",
			fmt.Sprintf("APPLYCONFIGURATION_PATHS ?= %s;", apiPath)); err != nil {
			return fmt.Errorf("error updating APPLYCONFIGURATION_PATHS: %w", err)
		}
	}

	return nil
}

// updateGitignore adds pkg/applyconfiguration/ to .gitignore
func (s *apiScaffolder) updateGitignore() error {
	gitignorePath := ".gitignore"

	// Check if already added
	err := util.ReplaceInFile(gitignorePath,
		"pkg/applyconfiguration/",
		"pkg/applyconfiguration/")
	if err != nil {
		// Not found - append to end of file
		log.Info("adding pkg/applyconfiguration/ to .gitignore")

		ignoreEntry := `
# Server-Side Apply configurations (regenerated by 'make generate')
api/**/applyconfiguration/`

		if err := util.InsertCode(gitignorePath, "", ignoreEntry); err != nil {
			return fmt.Errorf("error updating .gitignore: %w", err)
		}
	}

	return nil
}

// updateGroupVersionInfo adds the applyconfiguration generation marker
func (s *apiScaffolder) updateGroupVersionInfo() error {
	var groupVersionPath string
	if s.config.IsMultiGroup() && s.resource.Group != "" {
		groupVersionPath = filepath.Join("api", s.resource.Group, s.resource.Version, "groupversion_info.go")
	} else {
		groupVersionPath = filepath.Join("api", s.resource.Version, "groupversion_info.go")
	}

	// Add the marker after the object:generate marker
	marker := `// +kubebuilder:object:generate=true`
	insert := `
// +kubebuilder:ac:generate=true`
	if err := util.InsertCode(groupVersionPath, marker, insert); err != nil {
		return fmt.Errorf("error adding ac:generate marker: %w", err)
	}

	return nil
}

// getAPIPath returns the path to the API directory for this resource
func (s *apiScaffolder) getAPIPath() string {
	if s.config.IsMultiGroup() && s.resource.Group != "" {
		return "./api/" + filepath.Join(s.resource.Group, s.resource.Version)
	}
	return "./api/" + s.resource.Version
}
