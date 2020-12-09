/*
Copyright 2020 The Kubernetes Authors.

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

	"sigs.k8s.io/kubebuilder/v2/pkg/model"
	"sigs.k8s.io/kubebuilder/v2/pkg/model/config"
	"sigs.k8s.io/kubebuilder/v2/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates/api"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates/config/crd"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates/config/crd/patches"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates/config/rbac"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates/config/samples"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates/controllers"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/internal/cmdutil"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/internal/machinery"
)

var _ cmdutil.Scaffolder = &apiScaffolder{}

// apiScaffolder contains configuration for generating scaffolding for Go type
// representing the API and controller that implements the behavior for the API.
type apiScaffolder struct {
	config      *config.Config
	boilerplate string
	resource    *resource.Resource
	// plugins is the list of plugins we should allow to transform our generated scaffolding
	plugins []model.Plugin
}

// NewAPIScaffolder returns a new Scaffolder for API/controller creation operations
func NewAPIScaffolder(
	config *config.Config,
	boilerplate string,
	res *resource.Resource,
	plugins []model.Plugin,
) cmdutil.Scaffolder {
	return &apiScaffolder{
		config:      config,
		boilerplate: boilerplate,
		resource:    res,
		plugins:     plugins,
	}
}

// Scaffold implements Scaffolder
func (s *apiScaffolder) Scaffold() error {
	fmt.Println("Writing scaffold for you to edit...")
	return s.scaffold()
}

func (s *apiScaffolder) newUniverse() *model.Universe {
	return model.NewUniverse(
		model.WithConfig(s.config),
		model.WithBoilerplate(s.boilerplate),
		model.WithResource(s.resource),
	)
}

// TODO: re-use universe created by s.newUniverse() if possible.
func (s *apiScaffolder) scaffold() error {
	// Check if we need to do the API and controller before updating with previously existing info
	doAPI := s.resource.API != nil && s.resource.API.Version != ""
	doController := s.resource.Controller

	// Update the known data about resource
	var err error
	s.resource, err = s.config.UpdateResources(s.resource)
	if err != nil {
		return fmt.Errorf("error updating resources in config: %w", err)
	}

	if doAPI {
		if err := machinery.NewScaffold(s.plugins...).Execute(
			s.newUniverse(),
			&api.Types{},
			&api.Group{},
			&samples.CRDSample{},
			&rbac.CRDEditorRole{},
			&rbac.CRDViewerRole{},
			&patches.EnableWebhookPatch{},
			&patches.EnableCAInjectionPatch{},
		); err != nil {
			return fmt.Errorf("error scaffolding APIs: %w", err)
		}

		if err := machinery.NewScaffold().Execute(
			s.newUniverse(),
			&crd.Kustomization{},
			&crd.KustomizeConfig{},
		); err != nil {
			return fmt.Errorf("error scaffolding kustomization: %w", err)
		}
	}

	if doController {
		if err := machinery.NewScaffold(s.plugins...).Execute(
			s.newUniverse(),
			&controllers.SuiteTest{WireResource: doAPI},
			&controllers.Controller{ControllerRuntimeVersion: ControllerRuntimeVersion, WireResource: doAPI},
		); err != nil {
			return fmt.Errorf("error scaffolding controller: %w", err)
		}
	}

	if err := machinery.NewScaffold(s.plugins...).Execute(
		s.newUniverse(),
		&templates.MainUpdater{WireResource: doAPI, WireController: doController},
	); err != nil {
		return fmt.Errorf("error updating main.go: %w", err)
	}

	return nil
}
