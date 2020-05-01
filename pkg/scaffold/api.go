/*
Copyright 2019 The Kubernetes Authors.

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

package scaffold

import (
	"fmt"

	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/internal/machinery"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/controller"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/crd"
)

var _ Scaffolder = &apiScaffolder{}

// apiScaffolder contains configuration for generating scaffolding for Go type
// representing the API and controller that implements the behavior for the API.
type apiScaffolder struct {
	config      *config.Config
	boilerplate string
	resource    *resource.Resource
	// plugins is the list of plugins we should allow to transform our generated scaffolding
	plugins []model.Plugin
	// doResource indicates whether to scaffold API Resource or not
	doResource bool
	// doController indicates whether to scaffold controller files or not
	doController bool
}

// NewAPIScaffolder returns a new Scaffolder for API/controller creation operations
func NewAPIScaffolder(
	config *config.Config,
	boilerplate string,
	res *resource.Resource,
	doResource, doController bool,
	plugins []model.Plugin,
) Scaffolder {
	return &apiScaffolder{
		config:       config,
		boilerplate:  boilerplate,
		resource:     res,
		plugins:      plugins,
		doResource:   doResource,
		doController: doController,
	}
}

// Scaffold implements Scaffolder
func (s *apiScaffolder) Scaffold() error {
	fmt.Println("Writing scaffold for you to edit...")

	switch {
	case s.config.IsV2(), s.config.IsV3():
		return s.scaffold()
	default:
		return fmt.Errorf("unknown project version %v", s.config.Version)
	}
}

func (s *apiScaffolder) newUniverse() *model.Universe {
	return model.NewUniverse(
		model.WithConfig(s.config),
		model.WithBoilerplate(s.boilerplate),
		model.WithResource(s.resource),
	)
}

func (s *apiScaffolder) scaffold() error {
	if s.doResource {
		s.config.AddResource(s.resource.GVK())

		if err := machinery.NewScaffold(s.plugins...).Execute(
			s.newUniverse(),
			&templates.Types{},
			&templates.Group{},
			&templates.CRDSample{},
			&templates.CRDEditorRole{},
			&templates.CRDViewerRole{},
			&crd.EnableWebhookPatch{},
			&crd.EnableCAInjectionPatch{},
		); err != nil {
			return fmt.Errorf("error scaffolding APIs: %v", err)
		}

		if err := machinery.NewScaffold().Execute(
			s.newUniverse(),
			&crd.Kustomization{},
			&crd.KustomizeConfig{},
		); err != nil {
			return fmt.Errorf("error scaffolding kustomization: %v", err)
		}

	}

	if s.doController {
		if err := machinery.NewScaffold(s.plugins...).Execute(
			s.newUniverse(),
			&controller.SuiteTest{},
			&controller.Controller{},
		); err != nil {
			return fmt.Errorf("error scaffolding controller: %v", err)
		}
	}

	if err := machinery.NewScaffold(s.plugins...).Execute(
		s.newUniverse(),
		&templates.MainUpdater{WireResource: s.doResource, WireController: s.doController},
	); err != nil {
		return fmt.Errorf("error updating main.go: %v", err)
	}

	return nil
}
