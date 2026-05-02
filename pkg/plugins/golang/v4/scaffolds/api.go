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
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/api"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/cmd"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/controllers"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/hack"
)

var _ plugins.Scaffolder = &apiScaffolder{}

// apiScaffolder contains configuration for generating scaffolding for Go type
// representing the API and controller that implements the behavior for the API.
type apiScaffolder struct {
	config   config.Config
	resource resource.Resource

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem

	// force indicates whether to scaffold controller files even if it exists or not
	force bool
}

// NewAPIScaffolder returns a new Scaffolder for API/controller creation operations
func NewAPIScaffolder(cfg config.Config, res resource.Resource, force bool) plugins.Scaffolder {
	return &apiScaffolder{
		config:   cfg,
		resource: res,
		force:    force,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *apiScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements cmdutil.Scaffolder
func (s *apiScaffolder) Scaffold() error {
	log.Info("Writing scaffold for you to edit...")

	// Load the boilerplate
	boilerplate, err := afero.ReadFile(s.fs.FS, hack.DefaultBoilerplatePath)
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			log.Warn("unable to find boilerplate file. "+
				"This file is used to generate the license header in the project.\n"+
				"Note that controller-gen will also use this. Ensure that you "+
				"add the license file or configure your project accordingly",
				"file_path", hack.DefaultBoilerplatePath, "error", err)
			boilerplate = []byte("")
		} else {
			return fmt.Errorf("error scaffolding API/controller: failed to load boilerplate: %w", err)
		}
	}

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
		machinery.WithBoilerplate(string(boilerplate)),
		machinery.WithResource(&s.resource),
	)

	// Keep track of these values before the update
	doAPI := s.resource.HasAPI()
	doController := s.resource.HasController()

	if err := s.config.UpdateResource(s.resource); err != nil {
		return fmt.Errorf("error updating resource: %w", err)
	}

	if doAPI {
		if err := scaffold.Execute(
			&api.Types{Force: s.force},
			&api.Group{},
		); err != nil {
			return fmt.Errorf("error scaffolding APIs: %w", err)
		}

		// If SSA is enabled and groupversion_info.go already exists, we need to inject the marker
		// (the template only runs when creating a new version package)
		if s.resource.API != nil && s.resource.API.SSA {
			if err := s.updateGroupVersionInfo(); err != nil {
				return fmt.Errorf("error adding ac:generate marker: %w", err)
			}
			// Update Makefile if this is the first SSA API in the project
			if s.isFirstSSAAPI() {
				s.updateMakefile()
			}
		}
	}

	if doController {
		// Get the controller name to scaffold
		// If using the new Controllers field, get the last added controller name
		// Otherwise, use empty string to generate default name
		controllerName := ""
		if s.resource.Controllers != nil && !s.resource.Controllers.IsEmpty() {
			names := s.resource.Controllers.GetControllerNames()
			if len(names) > 0 {
				// Use the last controller name (the one just added)
				controllerName = names[len(names)-1]
			}
		}

		if err := scaffold.Execute(
			&controllers.SuiteTest{Force: s.force},
			&controllers.Controller{
				ControllerRuntimeVersion: ControllerRuntimeVersion,
				Force:                    s.force,
				ControllerName:           controllerName,
			},
			&controllers.ControllerTest{Force: s.force, DoAPI: doAPI},
		); err != nil {
			return fmt.Errorf("error scaffolding controller: %w", err)
		}
	}

	if err := scaffold.Execute(
		&cmd.MainUpdater{
			WireResource:   doAPI,
			WireController: doController,
			ControllerName: func() string {
				if s.resource.Controllers != nil && !s.resource.Controllers.IsEmpty() {
					names := s.resource.Controllers.GetControllerNames()
					if len(names) > 0 {
						return names[len(names)-1]
					}
				}
				return ""
			}(),
		},
	); err != nil {
		return fmt.Errorf("error updating cmd/main.go: %w", err)
	}

	return nil
}

// updateGroupVersionInfo adds the applyconfiguration generation marker
// when groupversion_info.go already exists (e.g., adding a second API to an existing version)
func (s *apiScaffolder) updateGroupVersionInfo() error {
	var groupVersionPath string
	if s.config.IsMultiGroup() && s.resource.Group != "" {
		groupVersionPath = filepath.Join("api", s.resource.Group, s.resource.Version, "groupversion_info.go")
	} else {
		groupVersionPath = filepath.Join("api", s.resource.Version, "groupversion_info.go")
	}

	// Check if marker already exists to avoid duplicates when using --force or multiple kinds
	hasMarker, err := util.HasFileContentWith(groupVersionPath, "+kubebuilder:ac:generate=true")
	if err != nil {
		return fmt.Errorf("error checking for existing ac:generate marker: %w", err)
	}
	if hasMarker {
		return nil
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

// Makefile injection constants
const (
	makefileApplyConfigurationMarker = "applyconfiguration"

	// Patterns to match and replace in Makefile
	//nolint:lll
	makefileOldObjectGenWithBoilerplateAndYear = "\"$(CONTROLLER_GEN)\" object:headerFile=\"hack/boilerplate.go.txt\",year=$(YEAR) " +
		"paths=\"./...\""
	//nolint:lll
	makefileNewObjectGenWithBoilerplateAndYear = "\"$(CONTROLLER_GEN)\" object:headerFile=\"hack/boilerplate.go.txt\",year=$(YEAR) " +
		"applyconfiguration:headerFile=\"hack/boilerplate.go.txt\" paths=\"./...\""

	makefileOldObjectGenWithBoilerplate = "\"$(CONTROLLER_GEN)\" object:headerFile=\"hack/boilerplate.go.txt\" " +
		"paths=\"./...\""
	makefileNewObjectGenWithBoilerplate = "\"$(CONTROLLER_GEN)\" object:headerFile=\"hack/boilerplate.go.txt\" " +
		"applyconfiguration:headerFile=\"hack/boilerplate.go.txt\" paths=\"./...\""

	makefileOldObjectGenNoBoilerplate = "\"$(CONTROLLER_GEN)\" object paths=\"./...\""
	makefileNewObjectGenNoBoilerplate = "\"$(CONTROLLER_GEN)\" object applyconfiguration paths=\"./...\""
)

// isFirstSSAAPI checks if this is the first API with SSA enabled in the project.
// Returns true if there are no other resources with SSA enabled.
func (s *apiScaffolder) isFirstSSAAPI() bool {
	// Get all resources in the project
	resources, err := s.config.GetResources()
	if err != nil {
		// If we can't get resources, assume this is the first
		return true
	}

	// Count resources with SSA enabled (excluding the current one)
	for _, res := range resources {
		// Skip the current resource
		if res.GVK == s.resource.GVK {
			continue
		}
		// Check if this resource has SSA enabled
		if res.API != nil && res.API.SSA {
			return false
		}
	}
	return true
}

// updateMakefile modifies the existing controller-gen object generator line to also run
// applyconfiguration generation. Only runs when the first SSA API is created.
// On failure, logs a warning and does not stop scaffolding.
func (s *apiScaffolder) updateMakefile() {
	makefilePath := "Makefile"

	// Skip if already updated
	hasMarker, err := util.HasFileContentWith(makefilePath, makefileApplyConfigurationMarker)
	if err != nil {
		log.Warn("unable to read Makefile. Add applyconfiguration generation manually for SSA",
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
