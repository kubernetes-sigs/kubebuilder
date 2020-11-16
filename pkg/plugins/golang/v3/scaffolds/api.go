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
) cmdutil.Scaffolder {
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
	if s.doResource {

		s.config.UpdateResources(s.resource.GVK())

		if err := machinery.NewScaffold(s.plugins...).Execute(
			s.newUniverse(),
			&api.Types{},
			&api.Group{},
			&samples.CRDSample{},
			&rbac.CRDEditorRole{},
			&rbac.CRDViewerRole{},
			&patches.EnableWebhookPatch{CRDVersion: s.resource.CRDVersion},
			&patches.EnableCAInjectionPatch{CRDVersion: s.resource.CRDVersion},
		); err != nil {
			return fmt.Errorf("error scaffolding APIs: %v", err)
		}

		if err := machinery.NewScaffold().Execute(
			s.newUniverse(),
			&crd.Kustomization{},
			&crd.KustomizeConfig{CRDVersion: s.resource.CRDVersion},
		); err != nil {
			return fmt.Errorf("error scaffolding kustomization: %v", err)
		}

	}

	if s.doController {
		if err := machinery.NewScaffold(s.plugins...).Execute(
			s.newUniverse(),
			&controllers.SuiteTest{WireResource: s.doResource},
			&controllers.Controller{ControllerRuntimeVersion: ControllerRuntimeVersion, WireResource: s.doResource},
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
