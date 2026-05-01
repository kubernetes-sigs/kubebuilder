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
	"os"
	"path/filepath"
	"strings"

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
		ssaEnabled := s.resource.API != nil && s.resource.API.SSA

		if err := scaffold.Execute(
			&api.Types{Force: s.force, SkipApplyConfig: !ssaEnabled && s.hasSSAInPackage()},
			&api.Group{},
		); err != nil {
			return fmt.Errorf("error scaffolding APIs: %w", err)
		}

		// If SSA is enabled and groupversion_info.go already exists, we need to inject the marker
		// (the template only runs when creating a new version package)
		if ssaEnabled {
			if err := s.updateGroupVersionInfo(); err != nil {
				return fmt.Errorf("error adding ac:generate marker: %w", err)
			}
			// The ac:generate package marker enables generation for every kind in the
			// group/version, so kinds scaffolded without SSA must opt out explicitly
			if resources, err := s.config.GetResources(); err == nil {
				s.optOutExistingKinds(resources)
				s.warnUntrackedKinds(resources)
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

// apiPackageDir returns the directory of the resource group/version package.
func (s *apiScaffolder) apiPackageDir() string {
	if s.config.IsMultiGroup() && s.resource.Group != "" {
		return filepath.Join("api", s.resource.Group, s.resource.Version)
	}
	return filepath.Join("api", s.resource.Version)
}

// updateGroupVersionInfo adds the applyconfiguration generation marker
// when groupversion_info.go already exists (e.g., adding a second API to an existing version)
func (s *apiScaffolder) updateGroupVersionInfo() error {
	groupVersionPath := filepath.Join(s.apiPackageDir(), "groupversion_info.go")

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
	insert := "\n// +kubebuilder:ac:generate=true"
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

	//nolint:lll
	makefileOldGenerateHelp = "generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations."
	//nolint:lll
	makefileNewGenerateHelp = "generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations and ApplyConfiguration types."
)

// isFirstSSAAPI checks if this is the first API with SSA enabled in the project.
// Returns true if there are no other resources with SSA enabled.
func (s *apiScaffolder) isFirstSSAAPI() bool {
	resources, err := s.config.GetResources()
	if err != nil {
		// If we can't get resources, assume this is the first
		return true
	}

	for _, res := range resources {
		if res.GVK == s.resource.GVK {
			continue
		}
		if res.API != nil && res.API.SSA {
			return false
		}
	}
	return true
}

// hasSSAInPackage checks if another kind in the same group/version has SSA enabled.
func (s *apiScaffolder) hasSSAInPackage() bool {
	resources, err := s.config.GetResources()
	if err != nil {
		return false
	}

	for _, res := range resources {
		if res.GVK == s.resource.GVK {
			continue
		}
		if res.Group == s.resource.Group && res.Version == s.resource.Version &&
			res.API != nil && res.API.SSA {
			return true
		}
	}
	return false
}

// optOutExistingKinds adds the +kubebuilder:ac:generate=false marker to kinds in the
// same group/version that were scaffolded without SSA, so the package-level marker
// does not generate ApplyConfigurations for them.
// On failure, logs a warning and does not stop scaffolding.
func (s *apiScaffolder) optOutExistingKinds(resources []resource.Resource) {
	for _, res := range resources {
		if res.GVK == s.resource.GVK || res.Group != s.resource.Group || res.Version != s.resource.Version {
			continue
		}
		if !res.HasAPI() || res.API.SSA {
			continue
		}

		typesPath := filepath.Join(s.apiPackageDir(), fmt.Sprintf("%s_types.go", strings.ToLower(res.Kind)))

		hasMarker, err := util.HasFileContentWith(typesPath, "+kubebuilder:ac:generate")
		if err != nil {
			log.Warn("unable to check the '+kubebuilder:ac:generate' marker. "+
				"Add '+kubebuilder:ac:generate=false' above the kind to exclude it "+
				"from ApplyConfiguration generation",
				"path", typesPath, "error", err)
			continue
		}
		if hasMarker {
			continue
		}

		if err := util.InsertCode(typesPath, "// +kubebuilder:object:root=true",
			"\n// +kubebuilder:ac:generate=false"); err != nil {
			log.Warn("unable to add the '+kubebuilder:ac:generate=false' marker. "+
				"Add it above the kind to exclude it from ApplyConfiguration generation",
				"path", typesPath, "error", err)
		}
	}
}

// warnUntrackedKinds warns when the group/version package has *_types.go files for
// kinds not tracked in the PROJECT file (e.g. APIs added manually), since the
// +kubebuilder:ac:generate=false marker cannot be added to them automatically.
func (s *apiScaffolder) warnUntrackedKinds(resources []resource.Resource) {
	known := map[string]bool{
		fmt.Sprintf("%s_types.go", strings.ToLower(s.resource.Kind)): true,
	}
	for _, res := range resources {
		if res.Group == s.resource.Group && res.Version == s.resource.Version {
			known[fmt.Sprintf("%s_types.go", strings.ToLower(res.Kind))] = true
		}
	}

	pkgDir := s.apiPackageDir()
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, "_types.go") || known[name] {
			continue
		}
		log.Warn("found an API not tracked in the PROJECT file, likely added manually. "+
			"The '+kubebuilder:ac:generate=false' marker cannot be added automatically. "+
			"Review this API and add the marker above the kind if it should not use Server-Side Apply",
			"path", filepath.Join(pkgDir, name))
	}
}

// updateMakefile modifies the existing controller-gen object generator line to also run
// applyconfiguration generation. Only runs when the first SSA API is created.
// On failure, logs a warning and does not stop scaffolding.
func (s *apiScaffolder) updateMakefile() {
	updated, err := addApplyConfigGenToMakefile("Makefile")
	if err != nil {
		log.Warn("unable to update Makefile 'generate' target to add ApplyConfiguration generation for Server-Side Apply. "+
			"Ensure your Makefile is updated to include 'applyconfiguration' in the controller-gen command. "+
			"For example, change '$(CONTROLLER_GEN) object paths=\"./...\"' to "+
			"'$(CONTROLLER_GEN) object applyconfiguration paths=\"./...\"'",
			"error", err)
		return
	}
	if updated {
		log.Info("applyconfiguration generation added to Makefile generate target")
	}
}

// addApplyConfigGenToMakefile adds applyconfiguration generation to the controller-gen
// object line, trying known patterns in order with a single read and write.
// Returns false when the Makefile already runs applyconfiguration generation.
func addApplyConfigGenToMakefile(makefilePath string) (bool, error) {
	replacements := []struct {
		old string
		new string
	}{
		{
			old: makefileOldObjectGenWithBoilerplateAndYear,
			new: makefileNewObjectGenWithBoilerplateAndYear,
		},
		{
			old: makefileOldObjectGenWithBoilerplate,
			new: makefileNewObjectGenWithBoilerplate,
		},
		{
			old: makefileOldObjectGenNoBoilerplate,
			new: makefileNewObjectGenNoBoilerplate,
		},
	}

	info, err := os.Stat(makefilePath)
	if err != nil {
		return false, fmt.Errorf("failed to stat file %q: %w", makefilePath, err)
	}
	//nolint:gosec // false positive
	content, err := os.ReadFile(makefilePath)
	if err != nil {
		return false, fmt.Errorf("failed to read file %q: %w", makefilePath, err)
	}

	text := string(content)
	if strings.Contains(text, makefileApplyConfigurationMarker) {
		return false, nil
	}

	for _, replacement := range replacements {
		if strings.Contains(text, replacement.old) {
			text = strings.Replace(text, replacement.old, replacement.new, 1)
			// Best effort: keep the generate target help accurate when it was not customized
			text = strings.Replace(text, makefileOldGenerateHelp, makefileNewGenerateHelp, 1)
			if err := os.WriteFile(makefilePath, []byte(text), info.Mode()); err != nil {
				return false, fmt.Errorf("failed to write file %q: %w", makefilePath, err)
			}
			return true, nil
		}
	}

	return false, fmt.Errorf("none of the known controller-gen object generator patterns matched")
}
