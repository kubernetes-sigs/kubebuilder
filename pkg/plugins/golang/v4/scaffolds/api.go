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
	"path"
	"path/filepath"
	"regexp"
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
	boilerplatePath := hack.DefaultBoilerplatePath
	boilerplate, err := afero.ReadFile(s.fs.FS, boilerplatePath)
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			log.Warn("unable to find boilerplate file. "+
				"This file is used to generate the license header in the project.\n"+
				"Note that controller-gen will also use this. Ensure that you "+
				"add the license file or configure your project accordingly",
				"file_path", hack.DefaultBoilerplatePath, "error", err)
			boilerplate = []byte("")
			// A project created with '--license none' has no boilerplate file
			boilerplatePath = ""
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
	// True when --force re-runs an API that already uses SSA
	hadSSA := false
	if res, err := s.config.GetResource(s.resource.GVK); err == nil {
		hadSSA = res.API != nil && res.API.SSA
	}

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
			s.updateGroupVersionInfo()
			// The ac:generate package marker enables generation for every kind in the
			// group/version, so kinds scaffolded without SSA must opt out explicitly.
			// Do it only for the first SSA API in the group/version: after that, a kind
			// without the opt-out marker is one the user opted in.
			if resources, err := s.config.GetResources(); err == nil {
				if !hadSSA && !s.hasSSAInPackage() {
					s.optOutExistingKinds(resources)
				}
				s.warnUntrackedKinds(resources)
			}
			// Update Makefile if this is the first SSA API in the project
			if s.isFirstSSAAPI() {
				s.updateMakefile(boilerplatePath)
			}
		}
	}

	if doController {
		if err := scaffold.Execute(
			&controllers.SuiteTest{Force: s.force},
			&controllers.Controller{
				ControllerRuntimeVersion: ControllerRuntimeVersion,
				Force:                    s.force,
				ControllerName:           s.controllerName(),
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
			ControllerName: s.controllerName(),
		},
	); err != nil {
		return fmt.Errorf("error updating cmd/main.go: %w", err)
	}

	return nil
}

// controllerName returns the last explicitly named controller, preserving the
// selection used by library consumers that append controllers to a resource.
// CLI requests carry only the controller being scaffolded.
func (s *apiScaffolder) controllerName() string {
	names := s.resource.Controllers.GetControllerNames()
	if len(names) == 0 {
		return ""
	}

	return names[len(names)-1]
}

// apiPackageDir returns the directory of the resource group/version package.
func (s *apiScaffolder) apiPackageDir() string {
	if s.config.IsMultiGroup() && s.resource.Group != "" {
		return filepath.Join("api", s.resource.Group, s.resource.Version)
	}
	return filepath.Join("api", s.resource.Version)
}

// acGenerateMarkerHint tells users how to apply the marker manually when
// scaffolding could not do it for them.
const acGenerateMarkerHint = "Add '// +kubebuilder:ac:generate=true' to the package comment in " +
	"groupversion_info.go, alongside the other Kubebuilder markers (for example right after " +
	"'// +kubebuilder:object:generate=true'), to enable Server-Side Apply for this group/version"

// updateGroupVersionInfo adds the applyconfiguration generation marker
// when groupversion_info.go already exists (e.g., adding a second API to an existing version).
// On failure, logs a warning and does not stop scaffolding.
func (s *apiScaffolder) updateGroupVersionInfo() {
	groupVersionPath := filepath.Join(s.apiPackageDir(), "groupversion_info.go")

	// Check if marker already exists to avoid duplicates when using --force or multiple kinds
	hasMarker, err := util.HasFileContentWith(groupVersionPath, "+kubebuilder:ac:generate=true")
	if err != nil {
		log.Warn("unable to check the '+kubebuilder:ac:generate=true' marker. "+acGenerateMarkerHint,
			"path", groupVersionPath, "error", err)
		return
	}
	if hasMarker {
		return
	}

	// Add the marker after the object:generate marker. The anchor is checked
	// first because the InsertCode error would echo the whole file.
	marker := `// +kubebuilder:object:generate=true`
	hasAnchor, err := util.HasFileContentWith(groupVersionPath, marker)
	if err != nil {
		log.Warn("unable to check the '+kubebuilder:object:generate=true' marker. "+acGenerateMarkerHint,
			"path", groupVersionPath, "error", err)
		return
	}
	if !hasAnchor {
		log.Warn("the '+kubebuilder:object:generate=true' marker was not found. "+acGenerateMarkerHint,
			"path", groupVersionPath)
		return
	}

	insert := "\n// +kubebuilder:ac:generate=true"
	if err := util.InsertCode(groupVersionPath, marker, insert); err != nil {
		log.Warn("unable to add the '+kubebuilder:ac:generate=true' marker. "+acGenerateMarkerHint,
			"path", groupVersionPath, "error", err)
	}
}

// Makefile injection constants
const (
	makefileApplyConfigurationMarker = "applyconfiguration"

	// The generators of the manifests target. Only this fragment is matched,
	// so customized flags elsewhere in the line do not block the update.
	makefileManifestsGeneratorsWithoutSSA = "rbac:roleName=manager-role crd webhook"

	makefileManifestsHelpWithoutSSA = "## Generate WebhookConfiguration, ClusterRole and " +
		"CustomResourceDefinition objects."
	makefileManifestsHelpWithSSA = "## Generate WebhookConfiguration, ClusterRole and " +
		"CustomResourceDefinition objects and ApplyConfiguration types."
)

// makefileManifestsGeneratorsWithSSA returns the manifests generators with applyconfiguration.
// The headerFile is set only when boilerplatePath is not empty, as in the generate target.
func makefileManifestsGeneratorsWithSSA(boilerplatePath string) string {
	generator := makefileApplyConfigurationMarker
	if boilerplatePath != "" {
		generator += fmt.Sprintf(":headerFile=%q", boilerplatePath)
	}
	return makefileManifestsGeneratorsWithoutSSA + " " + generator
}

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
				"from Server-Side Apply",
				"path", typesPath, "error", err)
			continue
		}
		if hasMarker {
			continue
		}

		// Check the anchor first, because the InsertCode error prints the whole file.
		anchor := "// +kubebuilder:object:root=true"
		hasAnchor, err := util.HasFileContentWith(typesPath, anchor)
		if err != nil {
			log.Warn("unable to check the '+kubebuilder:object:root=true' marker. "+
				"Add '+kubebuilder:ac:generate=false' above the kind to exclude it "+
				"from Server-Side Apply",
				"path", typesPath, "error", err)
			continue
		}
		if !hasAnchor {
			log.Warn("the '+kubebuilder:object:root=true' marker was not found. "+
				"Add '+kubebuilder:ac:generate=false' above the kind to exclude it "+
				"from Server-Side Apply",
				"path", typesPath)
			continue
		}

		if err := util.InsertCode(typesPath, anchor,
			"\n// +kubebuilder:ac:generate=false"); err != nil {
			log.Warn("unable to add the '+kubebuilder:ac:generate=false' marker. "+
				"Add it above the kind to exclude it from Server-Side Apply",
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
			"Review this API. If it should not use Server-Side Apply, add the marker above the kind "+
			"and remove its '+genclient' markers, if any",
			"path", filepath.Join(pkgDir, name))
	}
}

// updateMakefile adds applyconfiguration generation to the manifests target.
// Only runs when the first SSA API is created.
// On failure, logs a warning and does not stop scaffolding.
func (s *apiScaffolder) updateMakefile(boilerplatePath string) {
	updated, err := addApplyConfigGenToMakefile("Makefile", boilerplatePath)
	if err != nil {
		log.Warn("unable to update the Makefile 'manifests' target for Server-Side Apply. "+
			"Add 'applyconfiguration' to its controller-gen command, for example change "+
			"'$(CONTROLLER_GEN) "+makefileManifestsGeneratorsWithoutSSA+" paths=\"./...\"' to "+
			"'$(CONTROLLER_GEN) "+makefileManifestsGeneratorsWithSSA(boilerplatePath)+" paths=\"./...\"'",
			"error", err)
		return
	}
	if updated {
		log.Info("updated the Makefile 'manifests' target for Server-Side Apply")
	}
}

// manifestsRecipePattern matches the manifests target and the recipe lines that follow it.
var manifestsRecipePattern = regexp.MustCompile(`(?m)^manifests:.*\n((?:\t.*(?:\n|$))*)`)

// shellCommandSeparator splits a recipe line into the commands it runs.
var shellCommandSeparator = regexp.MustCompile(`&&|\|\||;|\|`)

// controllerGenArgs returns the arguments of the controller-gen commands in a recipe line.
// Leading environment assignments, such as GOFLAGS=-mod=mod, are skipped.
func controllerGenArgs(line string) []string {
	var args []string
	for _, command := range shellCommandSeparator.Split(line, -1) {
		fields := strings.Fields(strings.TrimLeft(strings.TrimSpace(command), "@-+"))
		for len(fields) > 0 && strings.Contains(fields[0], "=") {
			fields = fields[1:]
		}
		if len(fields) == 0 {
			continue
		}
		name := strings.Trim(fields[0], `"'`)
		if name == "$(CONTROLLER_GEN)" || name == "${CONTROLLER_GEN}" || path.Base(name) == "controller-gen" {
			args = append(args, fields[1:]...)
		}
	}
	return args
}

// manifestsRecipeRunsApplyConfig reports whether a controller-gen command in the manifests recipe
// already runs applyconfiguration. Lines that end with a backslash are joined into one command.
// Comments, other commands and other targets are ignored.
func manifestsRecipeRunsApplyConfig(makefilePath string) (bool, error) {
	content, err := os.ReadFile(makefilePath)
	if err != nil {
		return false, fmt.Errorf("failed to read %q: %w", makefilePath, err)
	}
	joined := strings.NewReplacer("\\\r\n", " ", "\\\n", " ").Replace(string(content))
	recipe := manifestsRecipePattern.FindStringSubmatch(joined)
	if recipe == nil {
		return false, nil
	}
	for line := range strings.SplitSeq(recipe[1], "\n") {
		line, _, _ = strings.Cut(line, "#")
		for _, arg := range controllerGenArgs(line) {
			if arg == makefileApplyConfigurationMarker || strings.HasPrefix(arg, makefileApplyConfigurationMarker+":") {
				return true, nil
			}
		}
	}
	return false, nil
}

// addApplyConfigGenToMakefile adds applyconfiguration generation to the manifests target.
// An empty boilerplatePath adds it without headerFile.
// Returns false when the Makefile already runs applyconfiguration generation.
func addApplyConfigGenToMakefile(makefilePath, boilerplatePath string) (bool, error) {
	hasApplyConfig, err := manifestsRecipeRunsApplyConfig(makefilePath)
	if err != nil {
		return false, fmt.Errorf("failed to check the Makefile for Server-Side Apply: %w", err)
	}
	if hasApplyConfig {
		return false, nil
	}

	// One replacement updates the help comment and the recipe together, so a
	// customized Makefile fails with no partial edits. The pattern is anchored
	// on the manifests target, so another target running the same generators
	// is not the one updated.
	pattern := "(?m)^(manifests: controller-gen )" +
		regexp.QuoteMeta(makefileManifestsHelpWithoutSSA) +
		"\n(\t.*?)" + regexp.QuoteMeta(makefileManifestsGeneratorsWithoutSSA)
	replacement := "${1}" + makefileManifestsHelpWithSSA + "\n${2}" +
		makefileManifestsGeneratorsWithSSA(boilerplatePath)
	if err := util.ReplaceRegexInFile(makefilePath, pattern, replacement); err != nil {
		return false, fmt.Errorf("failed to update the manifests target in %q: %w", makefilePath, err)
	}
	return true, nil
}
