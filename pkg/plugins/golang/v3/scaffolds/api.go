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

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates/api"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates/config/crd"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates/config/crd/patches"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates/config/rbac"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates/config/samples"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates/controllers"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates/hack"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/internal/cmdutil"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/internal/machinery"
)

var _ cmdutil.Scaffolder = &apiScaffolder{}

// apiScaffolder contains configuration for generating scaffolding for Go type
// representing the API and controller that implements the behavior for the API.
type apiScaffolder struct {
	config      config.Config
	boilerplate string
	resource    resource.Resource

	// fs is the filesystem that will be used by the scaffolder
	fs afero.Fs

	// plugins is the list of plugins we should allow to transform our generated scaffolding
	plugins []model.Plugin

	// force indicates whether to scaffold controller files even if it exists or not
	force bool
}

// NewAPIScaffolder returns a new Scaffolder for API/controller creation operations
func NewAPIScaffolder(
	config config.Config,
	res resource.Resource,
	force bool,
	plugins []model.Plugin,
) cmdutil.Scaffolder {
	return &apiScaffolder{
		config:   config,
		resource: res,
		plugins:  plugins,
		force:    force,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *apiScaffolder) InjectFS(fs afero.Fs) {
	s.fs = fs
}

func (s *apiScaffolder) newUniverse() *model.Universe {
	return model.NewUniverse(
		model.WithConfig(s.config),
		model.WithBoilerplate(s.boilerplate),
		model.WithResource(&s.resource),
	)
}

// Scaffold implements cmdutil.Scaffolder
func (s *apiScaffolder) Scaffold() error {
	fmt.Println("Writing scaffold for you to edit...")

	// Load the boilerplate
	bp, err := afero.ReadFile(s.fs, hack.DefaultBoilerplatePath)
	if err != nil {
		return fmt.Errorf("error scaffolding API/controller: unable to load boilerplate: %w", err)
	}
	s.boilerplate = string(bp)

	// Keep track of these values before the update
	doAPI := s.resource.HasAPI()
	doController := s.resource.HasController()

	if doAPI {

		if err := s.config.UpdateResource(s.resource); err != nil {
			return fmt.Errorf("error updating resource: %w", err)
		}

		if err := machinery.NewScaffold(s.fs, s.plugins...).Execute(
			s.newUniverse(),
			&api.Types{Force: s.force},
			&api.Group{},
			&samples.CRDSample{Force: s.force},
			&rbac.CRDEditorRole{},
			&rbac.CRDViewerRole{},
			&patches.EnableWebhookPatch{},
			&patches.EnableCAInjectionPatch{},
		); err != nil {
			return fmt.Errorf("error scaffolding APIs: %v", err)
		}

		if err := machinery.NewScaffold(s.fs).Execute(
			s.newUniverse(),
			&crd.Kustomization{},
			&crd.KustomizeConfig{},
		); err != nil {
			return fmt.Errorf("error scaffolding kustomization: %v", err)
		}

	}

	if doController {
		if err := machinery.NewScaffold(s.fs, s.plugins...).Execute(
			s.newUniverse(),
			&controllers.SuiteTest{Force: s.force},
			&controllers.Controller{ControllerRuntimeVersion: ControllerRuntimeVersion, Force: s.force},
		); err != nil {
			return fmt.Errorf("error scaffolding controller: %v", err)
		}
	}

	if err := machinery.NewScaffold(s.fs, s.plugins...).Execute(
		s.newUniverse(),
		&templates.MainUpdater{WireResource: doAPI, WireController: doController},
	); err != nil {
		return fmt.Errorf("error updating main.go: %v", err)
	}

	return nil
}
