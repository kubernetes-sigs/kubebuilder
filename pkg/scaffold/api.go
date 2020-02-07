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
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/internal/machinery"
	controllerv1 "sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/v1/controller"
	crdv1 "sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/v1/crd"
	templatesv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/v2"
	controllerv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/v2/controller"
	crdv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/v2/crd"
)

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

func (s *apiScaffolder) Scaffold() error {
	fmt.Println("Writing scaffold for you to edit...")

	switch {
	case s.config.IsV1():
		return s.scaffoldV1()
	case s.config.IsV2():
		return s.scaffoldV2()
	default:
		return fmt.Errorf("unknown project version %v", s.config.Version)
	}
}

func (s *apiScaffolder) newUniverse() *model.Universe {
	return model.NewUniverse(
		model.WithConfig(&s.config.Config),
		model.WithBoilerplate(s.boilerplate),
		model.WithResource(s.resource),
	)
}

func (s *apiScaffolder) scaffoldV1() error {
	if s.doResource {
		fmt.Println(filepath.Join("pkg", "apis", s.resource.GroupPackageName, s.resource.Version,
			fmt.Sprintf("%s_types.go", strings.ToLower(s.resource.Kind))))
		fmt.Println(filepath.Join("pkg", "apis", s.resource.GroupPackageName, s.resource.Version,
			fmt.Sprintf("%s_types_test.go", strings.ToLower(s.resource.Kind))))

		if err := machinery.NewScaffold().Execute(
			s.newUniverse(),
			&crdv1.Register{},
			&crdv1.Types{},
			&crdv1.VersionSuiteTest{},
			&crdv1.TypesTest{},
			&crdv1.Doc{},
			&crdv1.Group{},
			&crdv1.AddToScheme{},
			&crdv1.CRDSample{},
		); err != nil {
			return fmt.Errorf("error scaffolding APIs: %v", err)
		}
	} else {
		// disable generation of example reconcile body if not scaffolding resource
		// because this could result in a fork-bomb of k8s resources where watching a
		// deployment, replicaset etc. results in generating deployment which
		// end up generating replicaset, pod etc recursively.
		s.resource.CreateExampleReconcileBody = false
	}

	if s.doController {
		fmt.Println(filepath.Join("pkg", "controller", strings.ToLower(s.resource.Kind),
			fmt.Sprintf("%s_controller.go", strings.ToLower(s.resource.Kind))))
		fmt.Println(filepath.Join("pkg", "controller", strings.ToLower(s.resource.Kind),
			fmt.Sprintf("%s_controller_test.go", strings.ToLower(s.resource.Kind))))

		if err := machinery.NewScaffold().Execute(
			s.newUniverse(),
			&controllerv1.Controller{},
			&controllerv1.AddController{},
			&controllerv1.Test{},
			&controllerv1.SuiteTest{},
		); err != nil {
			return fmt.Errorf("error scaffolding controller: %v", err)
		}
	}

	return nil
}

func (s *apiScaffolder) scaffoldV2() error {
	if s.doResource {
		// Only save the resource in the config file if it didn't exist
		if s.config.AddResource(s.resource.GVK()) {
			if err := s.config.Save(); err != nil {
				return fmt.Errorf("error updating project file with resource information : %v", err)
			}
		}

		if s.config.MultiGroup {
			fmt.Println(filepath.Join("apis", s.resource.Group, s.resource.Version,
				fmt.Sprintf("%s_types.go", strings.ToLower(s.resource.Kind))))
		} else {
			fmt.Println(filepath.Join("api", s.resource.Version,
				fmt.Sprintf("%s_types.go", strings.ToLower(s.resource.Kind))))
		}

		if err := machinery.NewScaffold(s.plugins...).Execute(
			s.newUniverse(),
			&templatesv2.Types{},
			&templatesv2.Group{},
			&templatesv2.CRDSample{},
			&templatesv2.CRDEditorRole{},
			&templatesv2.CRDViewerRole{},
			&crdv2.EnableWebhookPatch{},
			&crdv2.EnableCAInjectionPatch{},
		); err != nil {
			return fmt.Errorf("error scaffolding APIs: %v", err)
		}

		kustomizationFile := &crdv2.Kustomization{}
		if err := machinery.NewScaffold().Execute(
			s.newUniverse(),
			kustomizationFile,
			&crdv2.KustomizeConfig{},
		); err != nil {
			return fmt.Errorf("error scaffolding kustomization: %v", err)
		}

		if err := kustomizationFile.Update(); err != nil {
			return fmt.Errorf("error updating kustomization.yaml: %v", err)
		}

	} else {
		// disable generation of example reconcile body if not scaffolding resource
		// because this could result in a fork-bomb of k8s resources where watching a
		// deployment, replicaset etc. results in generating deployment which
		// end up generating replicaset, pod etc recursively.
		s.resource.CreateExampleReconcileBody = false
	}

	if s.doController {
		if s.config.MultiGroup {
			fmt.Println(filepath.Join("controllers", s.resource.Group,
				fmt.Sprintf("%s_controller.go", strings.ToLower(s.resource.Kind))))
		} else {
			fmt.Println(filepath.Join("controllers",
				fmt.Sprintf("%s_controller.go", strings.ToLower(s.resource.Kind))))
		}

		suiteTestFile := &controllerv2.SuiteTest{}
		if err := machinery.NewScaffold(s.plugins...).Execute(
			s.newUniverse(),
			suiteTestFile,
			&controllerv2.Controller{},
		); err != nil {
			return fmt.Errorf("error scaffolding controller: %v", err)
		}

		if err := suiteTestFile.Update(); err != nil {
			return fmt.Errorf("error updating suite_test.go under controllers pkg: %v", err)
		}
	}

	if err := (&templatesv2.Main{}).Update(
		&templatesv2.MainUpdateOptions{
			Config:         &s.config.Config,
			WireResource:   s.doResource,
			WireController: s.doController,
			Resource:       s.resource,
		},
	); err != nil {
		return fmt.Errorf("error updating main.go: %v", err)
	}

	return nil
}
