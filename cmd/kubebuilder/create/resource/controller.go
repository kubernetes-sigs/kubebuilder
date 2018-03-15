/*
Copyright 2017 The Kubernetes Authors.

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

package resource

import (
	"fmt"
	"path/filepath"
	"strings"

	createutil "github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/create/util"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
)

func doController(dir string, args resourceTemplateArgs) bool {
	path := filepath.Join(dir, "pkg", "controller", strings.ToLower(createutil.KindName), "controller.go")
	fmt.Printf("\t%s\n", filepath.Join(
		"pkg", "controller", strings.ToLower(createutil.KindName), "controller.go"))
	return util.WriteIfNotFound(path, "resource-controller-template", resourceControllerTemplate, args)
}

var resourceControllerTemplate = `{{.BoilerPlate}}

package {{lower .Kind}}

import (
    "log"

    "github.com/kubernetes-sigs/kubebuilder/pkg/controller"
    "github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"

    {{.Group}}{{.Version}}client "{{.Repo}}/pkg/client/clientset_generated/clientset/typed/{{.Group}}/{{.Version}}"
    {{.Group}}{{.Version}}lister "{{.Repo}}/pkg/client/listers_generated/{{.Group}}/{{.Version}}"
    {{.Group}}{{.Version}} "{{.Repo}}/pkg/apis/{{.Group}}/{{.Version}}"
    {{.Group}}{{.Version}}informer "{{.Repo}}/pkg/client/informers_generated/externalversions/{{.Group}}/{{.Version}}"
    "{{.Repo}}/pkg/inject/args"
)

// EDIT THIS FILE
// This files was created by "kubebuilder create resource" for you to edit.
// Controller implementation logic for {{.Kind}} resources goes here.

func (bc *{{.Kind}}Controller) Reconcile(k types.ReconcileKey) error {
    // INSERT YOUR CODE HERE
    log.Printf("Implement the Reconcile function on {{lower .Kind}}.{{.Kind}}Controller to reconcile %s\n", k.Name)
    return nil
}

// +controller:group={{ .Group }},version={{ .Version }},kind={{ .Kind}},resource={{ .Resource }}
type {{.Kind}}Controller struct {
    // INSERT ADDITIONAL FIELDS HERE
    {{lower .Kind}}Lister {{.Group}}{{.Version}}lister.{{.Kind}}Lister
    {{lower .Kind}}client {{.Group}}{{.Version}}client.{{title .Group}}{{title .Version}}Interface
}

// ProvideController provides a controller that will be run at startup.  Kubebuilder will use codegeneration
// to automatically register this controller in the inject package
func ProvideController(arguments args.InjectArgs) (*controller.GenericController, error) {
    // INSERT INITIALIZATIONS FOR ADDITIONAL FIELDS HERE
    bc := &{{.Kind}}Controller{
        {{lower .Kind}}Lister: arguments.ControllerManager.GetInformerProvider(&{{.Group}}{{.Version}}.{{.Kind}}{}).({{.Group}}{{.Version}}informer.{{.Kind}}Informer).Lister(),
        {{lower .Kind}}client: arguments.Clientset.{{title .Group}}{{title .Version}}(),
    }

    // Create a new controller that will call {{.Kind}}Controller.Reconcile on changes to {{.Kind}}s
    gc := &controller.GenericController{
        Name: "{{.Kind}}Controller",
        Reconcile: bc.Reconcile,
        InformerRegistry: arguments.ControllerManager,
    }
    if err := gc.Watch(&{{.Group}}{{.Version}}.{{.Kind}}{}); err != nil {
        return gc, err
    }

    // INSERT ADDITIONAL WATCHES HERE BY CALLING gc.Watch.*() FUNCTIONS
    // NOTE: Informers for Kubernetes resources *MUST* be registered in the pkg/inject package so that they are started.
    return gc, nil
}
`
