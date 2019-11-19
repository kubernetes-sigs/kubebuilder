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

	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/project"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/util"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/controller"
	crdv1 "sigs.k8s.io/kubebuilder/pkg/scaffold/v1/crd"
	scaffoldv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2"
	crdv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2/crd"
)

// API contains configuration for generating scaffolding for Go type
// representing the API and controller that implements the behavior for the API.
type API struct {
	scaffold *Scaffold

	// Plugins is the list of plugins we should allow to transform our generated scaffolding
	Plugins []Plugin

	Resource *resource.Resource

	project *input.ProjectFile

	// DoResource indicates whether to scaffold API Resource or not
	DoResource bool

	// DoController indicates whether to scaffold controller files or not
	DoController bool

	// Force indicates that the resource should be created even if it already exists.
	Force bool
}

// Validate validates whether API scaffold has correct bits to generate
// scaffolding for API.
func (api *API) Validate() error {
	if err := api.setDefaults(); err != nil {
		return err
	}
	if err := api.Resource.Validate(); err != nil {
		return err
	}

	if api.resourceExists() && !api.Force {
		return fmt.Errorf("API resource already exists")
	}

	return nil
}

func (api *API) setDefaults() error {
	if api.project == nil {
		p, err := LoadProjectFile("PROJECT")
		if err != nil {
			return err
		}
		api.project = &p
	}
	return nil
}

func (api *API) Scaffold() error {
	if err := api.setDefaults(); err != nil {
		return err
	}

	if err := api.Resource.Init(); err != nil {
		return err
	}

	switch ver := api.project.Version; ver {
	case project.Version1:
		return api.scaffoldV1()
	case project.Version2:
		return api.scaffoldV2()
	default:
		return fmt.Errorf("")
	}
}

func (api *API) buildUniverse() *model.Universe {
	resourceModel := &model.Resource{
		Namespaced: api.Resource.Namespaced,
		Group:      api.Resource.Group,
		Version:    api.Resource.Version,
		Kind:       api.Resource.Kind,
		Resource:   api.Resource.Resource,
		Plural:     api.Resource.Plural,
	}

	resourceModel.GoPackage, resourceModel.GroupDomain = util.GetResourceInfo(api.Resource, api.project.Repo, api.project.Domain)

	return &model.Universe{
		Resource: resourceModel,
	}
}

func (api *API) scaffoldV1() error {
	r := api.Resource

	if api.DoResource {
		fmt.Println(filepath.Join("pkg", "apis", r.Group, r.Version,
			fmt.Sprintf("%s_types.go", strings.ToLower(r.Kind))))
		fmt.Println(filepath.Join("pkg", "apis", r.Group, r.Version,
			fmt.Sprintf("%s_types_test.go", strings.ToLower(r.Kind))))

		err := (&Scaffold{}).Execute(api.buildUniverse(), input.Options{},
			&crdv1.Register{Resource: r},
			&crdv1.Types{Resource: r},
			&crdv1.VersionSuiteTest{Resource: r},
			&crdv1.TypesTest{Resource: r},
			&crdv1.Doc{Resource: r},
			&crdv1.Group{Resource: r},
			&crdv1.AddToScheme{Resource: r},
			&crdv1.CRDSample{Resource: r},
		)
		if err != nil {
			return fmt.Errorf("error scaffolding APIs: %v", err)
		}
	} else {
		// disable generation of example reconcile body if not scaffolding resource
		// because this could result in a fork-bomb of k8s resources where watching a
		// deployment, replicaset etc. results in generating deployment which
		// end up generating replicaset, pod etc recursively.
		r.CreateExampleReconcileBody = false
	}

	if api.DoController {
		fmt.Println(filepath.Join("pkg", "controller", strings.ToLower(r.Kind),
			fmt.Sprintf("%s_controller.go", strings.ToLower(r.Kind))))
		fmt.Println(filepath.Join("pkg", "controller", strings.ToLower(r.Kind),
			fmt.Sprintf("%s_controller_test.go", strings.ToLower(r.Kind))))

		err := (&Scaffold{}).Execute(api.buildUniverse(), input.Options{},
			&controller.Controller{Resource: r},
			&controller.AddController{Resource: r},
			&controller.Test{Resource: r},
			&controller.SuiteTest{Resource: r},
		)
		if err != nil {
			return fmt.Errorf("error scaffolding controller: %v", err)
		}
	}

	return nil
}

func (api *API) scaffoldV2() error {
	r := api.Resource

	if api.DoResource {
		if err := api.validateResourceGroup(r); err != nil {
			return err
		}

		fmt.Println(filepath.Join("api", r.Version,
			fmt.Sprintf("%s_types.go", strings.ToLower(r.Kind))))

		files := []input.File{
			&scaffoldv2.Types{
				Input: input.Input{
					Path: filepath.Join("api", r.Version, fmt.Sprintf("%s_types.go", strings.ToLower(r.Kind))),
				},
				Resource: r},
			&scaffoldv2.Group{Resource: r},
			&scaffoldv2.CRDSample{Resource: r},
			&scaffoldv2.CRDEditorRole{Resource: r},
			&scaffoldv2.CRDViewerRole{Resource: r},
			&crdv2.EnableWebhookPatch{Resource: r},
			&crdv2.EnableCAInjectionPatch{Resource: r},
		}

		scaffold := &Scaffold{
			Plugins: api.Plugins,
		}

		if err := scaffold.Execute(api.buildUniverse(), input.Options{}, files...); err != nil {
			return fmt.Errorf("error scaffolding APIs: %v", err)
		}

		crdKustomization := &crdv2.Kustomization{Resource: r}
		err := (&Scaffold{}).Execute(api.buildUniverse(),
			input.Options{},
			crdKustomization,
			&crdv2.KustomizeConfig{},
		)
		if err != nil && !isAlreadyExistsError(err) {
			return fmt.Errorf("error scaffolding kustomization: %v", err)
		}

		err = crdKustomization.Update()
		if err != nil {
			return fmt.Errorf("error updating kustomization.yaml: %v", err)
		}

		if !api.resourceExists() {
			// update scaffolded resource in project file
			api.project.Resources = append(api.project.Resources,
				input.Resource{Group: r.Group, Version: r.Version, Kind: r.Kind})
			err = saveProjectFile("PROJECT", api.project)
			if err != nil {
				fmt.Printf("error updating project file with resource information : %v \n", err)
			}
		}

	} else {
		// disable generation of example reconcile body if not scaffolding resource
		// because this could result in a fork-bomb of k8s resources where watching a
		// deployment, replicaset etc. results in generating deployment which
		// end up generating replicaset, pod etc recursively.
		r.CreateExampleReconcileBody = false
	}

	if api.DoController {
		fmt.Println(filepath.Join("controllers", fmt.Sprintf("%s_controller.go", strings.ToLower(r.Kind))))

		scaffold := &Scaffold{
			Plugins: api.Plugins,
		}

		ctrlScaffolder := &scaffoldv2.Controller{Resource: r}
		testsuiteScaffolder := &scaffoldv2.ControllerSuiteTest{Resource: r}
		err := scaffold.Execute(
			api.buildUniverse(),
			input.Options{},
			testsuiteScaffolder,
			ctrlScaffolder,
		)
		if err != nil {
			return fmt.Errorf("error scaffolding controller: %v", err)
		}

		err = testsuiteScaffolder.Update()
		if err != nil {
			return fmt.Errorf("error updating suite_test.go under controllers pkg: %v", err)
		}
	}

	err := (&scaffoldv2.Main{}).Update(
		&scaffoldv2.MainUpdateOptions{
			Project:        api.project,
			WireResource:   api.DoResource,
			WireController: api.DoController,
			Resource:       r,
		})
	if err != nil {
		return fmt.Errorf("error updating main.go: %v", err)
	}

	return nil
}

// Since we support single group only in v2 scaffolding, validate if resource
// being created belongs to existing group.
func (api *API) validateResourceGroup(r *resource.Resource) error {
	for _, existingGroup := range api.project.ResourceGroups() {
		if strings.ToLower(r.Group) != strings.ToLower(existingGroup) {
			return fmt.Errorf("group '%s' is not same as existing group '%s'. Multiple groups are not supported yet.", r.Group, existingGroup)
		}
	}
	return nil
}

// resourceExists returns true if API resource is already tracked by the PROJECT file.
// Note that this works only for v2, since in v1 resources are not tracked by the PROJECT file.
func (api *API) resourceExists() bool {
	for _, r := range api.project.Resources {
		if r.Group == api.Resource.Group &&
			r.Version == api.Resource.Version &&
			r.Kind == api.Resource.Kind {
			return true
		}
	}

	return false
}
