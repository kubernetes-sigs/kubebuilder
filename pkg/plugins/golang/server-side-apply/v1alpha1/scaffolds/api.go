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
	"os"
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

// Makefile injection: only when the server-side-apply plugin is used for the first time.
const (
	makefileGenerateTarget          = "generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations."
	makefileApplyConfigurationGen  = "\n\t\"$(CONTROLLER_GEN)\" applyconfiguration:headerFile=\"hack/boilerplate.go.txt\" paths=\"./api/...\""
	makefileApplyConfigurationMarker = "applyconfiguration:headerFile"
)

// apiScaffolder contains configuration for generating scaffolding with Server-Side Apply
type apiScaffolder struct {
	config   config.Config
	resource resource.Resource

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem

	// isFirstAPI is true when this is the first API using the server-side-apply plugin in the project.
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

	// 2. Add genclient markers to types for applyconfiguration generation (server-side-apply only)
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
	if err := s.updateMakefile(); err != nil {
		return fmt.Errorf("error updating Makefile: %w", err)
	}

	return nil
}

// injectGenclientMarkers adds +genclient and optionally +genclient:nonNamespaced to the API
// types file so controller-gen can generate applyconfiguration. Only used by server-side-apply.
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

// updateMakefile adds applyconfiguration generation to the generate target only when the
// first API using the server-side-apply plugin is created (PROJECT does not yet contain
// the plugin). Subsequent APIs do not modify the Makefile. Uses paths="./api/..." so
// controller-gen covers all API packages; it only generates for packages that have
// +kubebuilder:ac:generate=true in groupversion_info.go. On failure logs a warning and
// does not stop scaffolding.
func (s *apiScaffolder) updateMakefile() error {
	if !s.isFirstAPI {
		return nil
	}

	makefilePath := "Makefile"
	contents, err := os.ReadFile(makefilePath)
	if err != nil {
		log.Warn("unable to read Makefile. Add the applyconfiguration target manually for server-side-apply",
			"path", makefilePath, "error", err)
		return nil
	}
	if strings.Contains(string(contents), makefileApplyConfigurationMarker) {
		return nil
	}

	if err := util.InsertCode(makefilePath, makefileGenerateTarget, makefileApplyConfigurationGen); err != nil {
		log.Warn("unable to update Makefile to add server-side-apply applyconfiguration target. Add it manually if using server-side-apply",
			"error", err)
		return nil
	}
	log.Info("applyconfiguration generation added to Makefile (paths=./api/...)")

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
