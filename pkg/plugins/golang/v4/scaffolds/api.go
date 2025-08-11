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
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
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

	// withFeatureGates indicates whether to include feature gate support
	withFeatureGates bool
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

// SetWithFeatureGates sets whether to include feature gate support
func (s *apiScaffolder) SetWithFeatureGates(withFeatureGates bool) {
	s.withFeatureGates = withFeatureGates
}

// Scaffold implements cmdutil.Scaffolder
func (s *apiScaffolder) Scaffold() error {
	slog.Info("Writing scaffold for you to edit...")

	// Load the boilerplate
	boilerplate, err := afero.ReadFile(s.fs.FS, hack.DefaultBoilerplatePath)
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			slog.Warn("Unable to find boilerplate file."+
				"This file is used to generate the license header in the project.\n"+
				"Note that controller-gen will also use this. Therefore, ensure that you "+
				"add the license file or configure your project accordingly.",
				"file_path", hack.DefaultBoilerplatePath, "error", err)
			boilerplate = []byte("")
		} else {
			return fmt.Errorf("error scaffolding API/controller: unable to load boilerplate: %w", err)
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

	// Check if feature gates infrastructure already exists
	existingFeatureGatesFile := filepath.Join("internal", "featuregates", "featuregates.go")
	if _, err := os.Stat(existingFeatureGatesFile); err == nil {
		// Feature gates infrastructure already exists, enable support
		s.withFeatureGates = true
		slog.Debug("Detected existing feature gates infrastructure, enabling support")
	}

	// If using --force, discover existing feature gates before overwriting files
	var existingGates []string
	if s.force && doAPI {
		existingGates = s.discoverFeatureGates()
	}

	if doAPI {
		if err := scaffold.Execute(
			&api.Types{
				Force:                     s.force,
				IncludeFeatureGateExample: s.withFeatureGates,
			},
			&api.Group{},
		); err != nil {
			return fmt.Errorf("error scaffolding APIs: %w", err)
		}
	}

	if doController {
		if err := scaffold.Execute(
			&controllers.SuiteTest{Force: s.force},
			&controllers.Controller{ControllerRuntimeVersion: ControllerRuntimeVersion, Force: s.force},
			&controllers.ControllerTest{Force: s.force, DoAPI: doAPI},
		); err != nil {
			return fmt.Errorf("error scaffolding controller: %w", err)
		}
	}

	// Discover feature gates from newly scaffolded API types
	newGates := s.discoverFeatureGates()
	var availableGates []string

	// Merge existing gates with newly discovered ones if we used --force
	if len(existingGates) > 0 {
		gateMap := make(map[string]bool)

		// Add existing gates
		for _, gate := range existingGates {
			gateMap[gate] = true
		}

		// Add newly discovered gates (from template)
		for _, gate := range newGates {
			gateMap[gate] = true
		}

		// Create merged list
		var mergedGates []string
		for gate := range gateMap {
			mergedGates = append(mergedGates, gate)
		}

		availableGates = mergedGates
	} else {
		availableGates = newGates
	}

	// Only generate feature gates infrastructure if requested or if gates were discovered
	if s.withFeatureGates || len(availableGates) > 0 {
		// Generate feature gates file
		featureGatesTemplate := &cmd.FeatureGates{}
		featureGatesTemplate.AvailableGates = availableGates
		featureGatesTemplate.IfExistsAction = machinery.OverwriteFile
		if err := scaffold.Execute(featureGatesTemplate); err != nil {
			return fmt.Errorf("error scaffolding feature gates: %w", err)
		}
	}

	if err := scaffold.Execute(
		&cmd.MainUpdater{WireResource: doAPI, WireController: doController},
	); err != nil {
		return fmt.Errorf("error updating cmd/main.go: %w", err)
	}

	return nil
}

// discoverFeatureGates scans the API directory for feature gate markers
func (s *apiScaffolder) discoverFeatureGates() []string {
	mgParser := machinery.NewFeatureGateMarkerParser()

	// Try to parse the API directory
	apiDir := "api"
	if s.config.IsMultiGroup() && s.resource.Group != "" {
		apiDir = filepath.Join("api", s.resource.Group)
	}

	// Debug: Get current working directory
	if wd, err := os.Getwd(); err == nil {
		slog.Debug("Feature gate discovery", "workingDir", wd, "apiDir", apiDir)
	}

	// Check if the directory exists before trying to parse it
	if _, err := os.Stat(apiDir); err != nil {
		slog.Debug("API directory does not exist yet, skipping feature gate discovery", "apiDir", apiDir, "error", err)
		return []string{}
	}

	var allMarkers []machinery.FeatureGateMarker

	// Walk through each version directory and parse for feature gates
	entries, err := os.ReadDir(apiDir)
	if err != nil {
		slog.Debug("Error reading API directory", "error", err, "apiDir", apiDir)
		return []string{}
	}

	slog.Debug("API directory contents", "apiDir", apiDir, "fileCount", len(entries))

	for _, entry := range entries {
		slog.Debug("API directory file", "name", entry.Name(), "isDir", entry.IsDir())
		if entry.IsDir() {
			versionDir := filepath.Join(apiDir, entry.Name())

			// Use the existing parser to parse the version directory
			markers, err := mgParser.ParseDirectory(versionDir)
			if err != nil {
				slog.Debug("Error parsing version directory for feature gates", "error", err, "versionDir", versionDir)
				continue
			}

			slog.Debug("Parsed markers from directory", "versionDir", versionDir, "markerCount", len(markers))

			// Debug: Print all discovered markers
			for _, marker := range markers {
				slog.Debug("Found feature gate marker", "gateName", marker.GateName, "line", marker.Line, "file", marker.File)
			}

			allMarkers = append(allMarkers, markers...)
		}
	}

	featureGates := machinery.ExtractFeatureGates(allMarkers)
	if len(featureGates) > 0 {
		slog.Debug("Discovered feature gates", "featureGates", featureGates)
	} else {
		slog.Debug("No feature gates found in directory", "apiDir", apiDir)
	}

	return featureGates
}
