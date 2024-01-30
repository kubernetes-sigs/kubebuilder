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

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v4/scaffolds/internal/templates"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v4/scaffolds/internal/templates/api"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v4/scaffolds/internal/templates/controllers"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v4/scaffolds/internal/templates/hack"
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
func NewAPIScaffolder(config config.Config, res resource.Resource, force bool) plugins.Scaffolder {
	return &apiScaffolder{
		config:   config,
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
	log.Println("Writing scaffold for you to edit...")

	// Load the boilerplate
	boilerplate, err := afero.ReadFile(s.fs.FS, hack.DefaultBoilerplatePath)
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
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

	if doAPI {
		if err := scaffold.Execute(
			&api.Types{Force: s.force},
			&api.Group{},
		); err != nil {
			return fmt.Errorf("error scaffolding APIs: %v", err)
		}
	}

	if doController {
		if err := scaffold.Execute(
			&controllers.SuiteTest{Force: s.force, K8SVersion: EnvtestK8SVersion},
			&controllers.Controller{ControllerRuntimeVersion: ControllerRuntimeVersion, Force: s.force},
			&controllers.ControllerTest{Force: s.force, DoAPI: doAPI},
		); err != nil {
			return fmt.Errorf("error scaffolding controller: %v", err)
		}
	}

	if err := scaffold.Execute(
		&templates.MainUpdater{WireResource: doAPI, WireController: doController},
	); err != nil {
		return fmt.Errorf("error updating cmd/main.go: %v", err)
	}

	return nil
}
