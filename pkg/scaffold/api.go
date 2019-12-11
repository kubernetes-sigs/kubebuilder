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

	"github.com/gobuffalo/flect"

	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/project"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/util"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/controller"
	crdv1 "sigs.k8s.io/kubebuilder/pkg/scaffold/v1/crd"
	scaffoldv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2"
	controllerv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2/controller"
	crdv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2/crd"
)

// API contains configuration for generating scaffolding for Go type
// representing the API and controller that implements the behavior for the API.
type API struct {
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
		Plural:     flect.Pluralize(strings.ToLower(api.Resource.Kind)),
	}

	resourceModel.GoPackage, resourceModel.GroupDomain = util.GetResourceInfo(api.Resource, api.project.Repo, api.project.Domain, api.project.MultiGroup)

	return &model.Universe{
		Resource:   resourceModel,
		MultiGroup: api.project.MultiGroup,
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

		if !api.resourceExists() {
			api.project.Resources = append(api.project.Resources,
				input.Resource{Group: r.Group, Version: r.Version, Kind: r.Kind})

			// If the --force was used to re-crete a resource that was created before then,
			// the PROJECT file will not be updated.
			if err := saveProjectFile("PROJECT", api.project); err != nil {
				return fmt.Errorf("error updating project file with resource information : %v \n", err)
			}
		}

		var path string
		if api.project.MultiGroup {
			path = filepath.Join("apis", r.Group, r.Version, fmt.Sprintf("%s_types.go", strings.ToLower(r.Kind)))
		} else {
			path = filepath.Join("api", r.Version, fmt.Sprintf("%s_types.go", strings.ToLower(r.Kind)))
		}
		fmt.Println(path)

		files := []input.File{
			&scaffoldv2.Types{
				Input: input.Input{
					Path: path,
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

		if err := crdKustomization.Update(); err != nil {
			return fmt.Errorf("error updating kustomization.yaml: %v", err)
		}

	} else {
		// disable generation of example reconcile body if not scaffolding resource
		// because this could result in a fork-bomb of k8s resources where watching a
		// deployment, replicaset etc. results in generating deployment which
		// end up generating replicaset, pod etc recursively.
		r.CreateExampleReconcileBody = false
	}

	if api.DoController {
		if api.project.MultiGroup {
			fmt.Println(filepath.Join("controllers", fmt.Sprintf("%s/%s_controller.go", r.Group, strings.ToLower(r.Kind))))
		} else {
			fmt.Println(filepath.Join("controllers", fmt.Sprintf("%s_controller.go", strings.ToLower(r.Kind))))
		}

		scaffold := &Scaffold{
			Plugins: api.Plugins,
		}

		ctrlScaffolder := &controllerv2.Controller{Resource: r}
		testsuiteScaffolder := &controllerv2.ControllerSuiteTest{Resource: r}
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

// isGroupAllowed will check if the group is == the group used before
// and not allow new groups if the project is not enabled to use multigroup layout
func (api *API) isGroupAllowed(r *resource.Resource) bool {
	if api.project.MultiGroup {
		return true
	}
	for _, existingGroup := range api.project.ResourceGroups() {
		if !strings.EqualFold(r.Group, existingGroup) {
			return false
		}
	}
	return true
}

// validateResourceGroup will return an error if the group cannot be created
func (api *API) validateResourceGroup(r *resource.Resource) error {
	if api.resourceExists() && !api.Force {
		return fmt.Errorf("group '%s', version '%s' and kind '%s' already exists.", r.Group, r.Version, r.Kind)
	}
	if !api.isGroupAllowed(r) {
		return fmt.Errorf("group '%s' is not same as existing group. Multiple groups are not enabled in this project. To enable, use the multigroup command.", r.Group)
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
