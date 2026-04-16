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
	"strings"

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

// Makefile injection: only when the ssa plugin is used for the first time.
// Instead of adding a separate controller-gen call, we modify the existing object generator
// line to include applyconfiguration. This is minimal and works for both SSA and non-SSA
// projects (applyconfiguration only generates for APIs with +kubebuilder:ac:generate=true).
const (
	// Marker to detect if applyconfiguration has already been added
	makefileApplyConfigurationMarker = "applyconfiguration"

	// Patterns to match and replace in Makefile
	// With boilerplate and YEAR variable (current default)
	//nolint:lll
	makefileOldObjectGenWithBoilerplateAndYear = "\"$(CONTROLLER_GEN)\" object:headerFile=\"hack/boilerplate.go.txt\",year=$(YEAR) " +
		"paths=\"./...\""
	//nolint:lll
	makefileNewObjectGenWithBoilerplateAndYear = "\"$(CONTROLLER_GEN)\" object:headerFile=\"hack/boilerplate.go.txt\",year=$(YEAR) " +
		"applyconfiguration:headerFile=\"hack/boilerplate.go.txt\" paths=\"./...\""

	// With boilerplate but without YEAR variable (legacy)
	makefileOldObjectGenWithBoilerplate = "\"$(CONTROLLER_GEN)\" object:headerFile=\"hack/boilerplate.go.txt\" " +
		"paths=\"./...\""
	makefileNewObjectGenWithBoilerplate = "\"$(CONTROLLER_GEN)\" object:headerFile=\"hack/boilerplate.go.txt\" " +
		"applyconfiguration:headerFile=\"hack/boilerplate.go.txt\" paths=\"./...\""

	// Without boilerplate file
	makefileOldObjectGenNoBoilerplate = "\"$(CONTROLLER_GEN)\" object paths=\"./...\""
	makefileNewObjectGenNoBoilerplate = "\"$(CONTROLLER_GEN)\" object applyconfiguration paths=\"./...\""
)

// apiScaffolder contains configuration for generating scaffolding with Server-Side Apply
type apiScaffolder struct {
	config   config.Config
	resource resource.Resource

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem

	// isFirstAPI is true when this is the first API using the ssa plugin in the project.
	// Only then do we add the applyconfiguration target to the Makefile.
	isFirstAPI bool
}

// NewAPIScaffolder returns a new Scaffolder for Server-Side Apply APIs. isFirstAPI should be
// true when the PROJECT file does not yet contain this plugin (first API); only then the
// Makefile is updated to add the applyconfiguration target.
func NewAPIScaffolder(cfg config.Config, res resource.Resource, isFirstAPI bool) plugins.Scaffolder {
	return &apiScaffolder{
		config:     cfg,
		resource:   res,
		isFirstAPI: isFirstAPI,
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

	// 2. Add genclient markers to types for applyconfiguration generation (ssa only)
	s.injectGenclientMarkers()

	// 3. Load boilerplate for templates
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

	// 4. Add applyconfiguration generation marker to groupversion_info.go
	if err := s.updateGroupVersionInfo(); err != nil {
		return fmt.Errorf("error updating groupversion_info.go: %w", err)
	}

	// 5. Scaffold Server-Side Apply controller (custom)
	if err := scaffold.Execute(
		&controllers.Controller{
			ControllerRuntimeVersion: golangv4scaffolds.ControllerRuntimeVersion,
		},
	); err != nil {
		return fmt.Errorf("error scaffolding controller: %w", err)
	}

	// 6. Update Makefile to add applyconfiguration generation (first API only)
	s.updateMakefile()

	return nil
}

// injectGenclientMarkers adds +genclient and optionally +genclient:nonNamespaced to the API
// types file so controller-gen can generate applyconfiguration. Only used by ssa.
// On failure logs a warning and continues (does not fail scaffolding).
func (s *apiScaffolder) injectGenclientMarkers() {
	typesPath := s.typesFilePath()
	const objectRootBlock = "// +kubebuilder:object:root=true\n// +kubebuilder:subresource:status"
	replacement := "// +genclient\n"
	if s.resource.API != nil && !s.resource.API.Namespaced {
		replacement += "// +genclient:nonNamespaced\n"
	}
	replacement += objectRootBlock

	if err := util.ReplaceInFile(typesPath, objectRootBlock, replacement); err != nil {
		log.Warn("unable to add genclient markers to types file. Applyconfiguration generation may need them",
			"path", typesPath, "error", err)
	}
}

func (s *apiScaffolder) typesFilePath() string {
	kindLower := strings.ToLower(s.resource.Kind)
	typesName := kindLower + "_types.go"
	if s.config.IsMultiGroup() && s.resource.Group != "" {
		return filepath.Join("api", s.resource.Group, s.resource.Version, typesName)
	}
	return filepath.Join("api", s.resource.Version, typesName)
}

// scaffoldStandardAPI scaffolds the standard API using golang/v4 and kustomize/v2
func (s *apiScaffolder) scaffoldStandardAPI() error {
	// Reuse golang/v4 API scaffolder for types
	// Use force=false to preserve existing suite_test.go with scaffold markers
	golangScaffolder := golangv4scaffolds.NewAPIScaffolder(s.config, s.resource, false)
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

// updateMakefile modifies the existing controller-gen object generator line to also run
// applyconfiguration generation. This only happens when the first API using the ssa
// plugin is created. The change is minimal - we add "applyconfiguration" to the existing
// controller-gen invocation. Since controller-gen only generates applyconfigurations for packages
// with +kubebuilder:ac:generate=true, this works for both SSA and non-SSA projects.
// On failure, logs a warning and does not stop scaffolding.
func (s *apiScaffolder) updateMakefile() {
	if !s.isFirstAPI {
		return
	}

	makefilePath := "Makefile"

	// Skip if already updated
	hasMarker, err := util.HasFileContentWith(makefilePath, makefileApplyConfigurationMarker)
	if err != nil {
		log.Warn("unable to read Makefile. Add applyconfiguration generation manually for ssa",
			"path", makefilePath, "error", err)
		return
	}
	if hasMarker {
		return
	}

	// Try multiple patterns to handle different Makefile formats
	// 1. Try with boilerplate and YEAR variable (current default)
	//nolint:lll
	err = util.ReplaceInFile(makefilePath, makefileOldObjectGenWithBoilerplateAndYear, makefileNewObjectGenWithBoilerplateAndYear)
	if err != nil {
		// 2. Try with boilerplate but without YEAR (legacy)
		err = util.ReplaceInFile(makefilePath, makefileOldObjectGenWithBoilerplate, makefileNewObjectGenWithBoilerplate)
		if err != nil {
			// 3. Try without boilerplate
			err = util.ReplaceInFile(makefilePath, makefileOldObjectGenNoBoilerplate, makefileNewObjectGenNoBoilerplate)
			if err != nil {
				log.Warn("unable to find standard controller-gen object generator line in Makefile. "+
					"Add applyconfiguration generation manually",
					"error", err)
				return
			}
		}
	}

	log.Info("applyconfiguration generation added to Makefile generate target")
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
