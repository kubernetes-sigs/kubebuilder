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
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/project"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/controller"
	resourcev1 "sigs.k8s.io/kubebuilder/pkg/scaffold/v1/resource"
	resourcev2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2"
)

// API contains configuration for generating scaffolding for Go type
// representing the API and controller that implements the behavior for the API.
type API struct {
	scaffold *Scaffold

	Resource *resourcev1.Resource

	project *input.ProjectFile

	// DoResource indicates whether to scaffold API Resource or not
	DoResource bool

	// DoController indicates whether to scaffold controller files or not
	DoController bool

	// RunMake indicates whether to run make or not after scaffolding APIs
	RunMake bool
}

// Validate validates whether API scaffold has correct bits to generate
// scaffolding for API.
func (api *API) Validate() error {
	if err := api.setDefaults(); err != nil {
		return err
	}
	if api.Resource.Group == "" {
		return fmt.Errorf("missing group information for resource")
	}
	if api.Resource.Version == "" {
		return fmt.Errorf("missing version information for resource")
	}
	if api.Resource.Kind == "" {
		return fmt.Errorf("missing kind information for resource")
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

func (api *API) scaffoldV1() error {
	r := api.Resource

	if api.DoResource {
		fmt.Println(filepath.Join("pkg", "apis", r.Group, r.Version,
			fmt.Sprintf("%s_types.go", strings.ToLower(r.Kind))))
		fmt.Println(filepath.Join("pkg", "apis", r.Group, r.Version,
			fmt.Sprintf("%s_types_test.go", strings.ToLower(r.Kind))))

		err := (&Scaffold{}).Execute(input.Options{},
			&resourcev1.Register{Resource: r},
			&resourcev1.Types{Resource: r},
			&resourcev1.VersionSuiteTest{Resource: r},
			&resourcev1.TypesTest{Resource: r},
			&resourcev1.Doc{Resource: r},
			&resourcev1.Group{Resource: r},
			&resourcev1.AddToScheme{Resource: r},
			&resourcev1.CRDSample{Resource: r},
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

		err := (&Scaffold{}).Execute(input.Options{},
			&controller.Controller{Resource: r},
			&controller.AddController{Resource: r},
			&controller.Test{Resource: r},
			&controller.SuiteTest{Resource: r},
		)
		if err != nil {
			return fmt.Errorf("error scaffolding controller: %v", err)
		}
	}

	if api.RunMake {
		fmt.Println("Running make...")
		cm := exec.Command("make") // #nosec
		cm.Stderr = os.Stderr
		cm.Stdout = os.Stdout
		if err := cm.Run(); err != nil {
			return fmt.Errorf("error running make: %v", err)
		}
	}

	return nil
}

func (api *API) scaffoldV2() error {
	r := api.Resource

	if api.DoResource {
		fmt.Println(filepath.Join("api", r.Version,
			fmt.Sprintf("%s_types.go", strings.ToLower(r.Kind))))
		// fmt.Println(filepath.Join("pkg", "apis", r.Group, r.Version,
		// 	fmt.Sprintf("%s_types_test.go", strings.ToLower(r.Kind))))

		err := (&Scaffold{}).Execute(
			input.Options{},
			// &resourcev1.Register{Resource: r},
			&resourcev1.Doc{
				Input: input.Input{
					Path: filepath.Join("api", r.Version, "doc.go"),
				},
				Resource: r},
			&resourcev1.Types{
				Input: input.Input{
					Path: filepath.Join("api", r.Version, fmt.Sprintf("%s_types.go", strings.ToLower(r.Kind))),
				},
				Resource: r},
			&resourcev2.Group{Resource: r},
			&resourcev1.CRDSample{Resource: r},
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
		fmt.Println(filepath.Join("controllers", fmt.Sprintf("%s_controller.go", strings.ToLower(r.Kind))))

		ctrlScaffolder := &resourcev2.Controller{Resource: r}
		err := (&Scaffold{}).Execute(
			input.Options{},
			ctrlScaffolder,
		)
		if err != nil {
			return fmt.Errorf("error scaffolding controller: %v", err)
		}

		err = ctrlScaffolder.UpdateMain("main.go")
		if err != nil {
			return fmt.Errorf("error updating main.go with reconciler code: %v", err)
		}
	}
	//
	// if api.RunMake {
	// 	fmt.Println("Running make...")
	// 	cm := exec.Command("make") // #nosec
	// 	cm.Stderr = os.Stderr
	// 	cm.Stdout = os.Stdout
	// 	if err := cm.Run(); err != nil {
	// 		return fmt.Errorf("error running make: %v", err)
	// 	}
	// }
	return nil
}
