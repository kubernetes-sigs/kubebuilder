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

package controller

import (
	"fmt"
	"path/filepath"
	"strings"

	createutil "github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/create/util"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
)

type controllerTemplateArgs struct {
	BoilerPlate       string
	Domain            string
	Group             string
	Version           string
	Kind              string
	Resource          string
	Repo              string
	PluralizedKind    string
	NonNamespacedKind bool
	CoreType          bool
}

func doController(dir string, args controllerTemplateArgs) bool {
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
    "k8s.io/client-go/tools/record"
    {{if .CoreType}}
    {{.Group}}{{.Version}}client "k8s.io/client-go/kubernetes/typed/{{.Group}}/{{.Version}}"
    {{.Group}}{{.Version}}lister "k8s.io/client-go/listers/{{.Group}}/{{.Version}}"
    {{.Group}}{{.Version}} "k8s.io/api/{{.Group}}/{{.Version}}"
    {{.Group}}{{.Version}}informer "k8s.io/client-go/informers/{{.Group}}/{{.Version}}"
    {{else}}
    {{.Group}}{{.Version}}client "{{.Repo}}/pkg/client/clientset/versioned/typed/{{.Group}}/{{.Version}}"
    {{.Group}}{{.Version}}lister "{{.Repo}}/pkg/client/listers/{{.Group}}/{{.Version}}"
    {{.Group}}{{.Version}} "{{.Repo}}/pkg/apis/{{.Group}}/{{.Version}}"
    {{.Group}}{{.Version}}informer "{{.Repo}}/pkg/client/informers/externalversions/{{.Group}}/{{.Version}}"
    {{end}}
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
{{if .CoreType}}// +kubebuilder:informers:group={{ .Group }},version={{ .Version }},kind={{ .Kind }}{{end}}
// +kubebuilder:controller:group={{ .Group }},version={{ .Version }},kind={{ .Kind}},resource={{ .Resource }}
type {{.Kind}}Controller struct {
    // INSERT ADDITIONAL FIELDS HERE
    {{lower .Kind}}Lister {{.Group}}{{.Version}}lister.{{.Kind}}Lister
    {{lower .Kind}}client {{.Group}}{{.Version}}client.{{title .Group}}{{title .Version}}Interface
    // recorder is an event recorder for recording Event resources to the
    // Kubernetes API.
    {{lower .Kind}}recorder record.EventRecorder
}

// ProvideController provides a controller that will be run at startup.  Kubebuilder will use codegeneration
// to automatically register this controller in the inject package
func ProvideController(arguments args.InjectArgs) (*controller.GenericController, error) {
    // INSERT INITIALIZATIONS FOR ADDITIONAL FIELDS HERE
    bc := &{{.Kind}}Controller{
        {{lower .Kind}}Lister: arguments.ControllerManager.GetInformerProvider(&{{.Group}}{{.Version}}.{{.Kind}}{}).({{.Group}}{{.Version}}informer.{{.Kind}}Informer).Lister(),
        {{if .CoreType}}{{lower .Kind}}client: arguments.KubernetesClientSet.{{title .Group}}{{title .Version}}(),{{else}}
        {{lower .Kind}}client: arguments.Clientset.{{title .Group}}{{title .Version}}(),{{end}}
        {{lower .Kind}}recorder: arguments.CreateRecorder("{{.Kind}}Controller"),
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

	// IMPORTANT:
	// To watch additional resource types - such as those created by your controller - add gc.Watch* function calls here
	// Watch function calls will transform each object event into a {{.Kind}} Key to be reconciled by the controller.
	//
	// **********
	// For any new Watched types, you MUST add the appropriate // +kubebuilder:informer and // +kubebuilder:rbac
	// annotations to the {{.Kind}}Controller and run "kubebuilder generate.
	// This will generate the code to start the informers and create the RBAC rules needed for running in a cluster.
	// See:
	// https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/controller#example-package
	// **********

    return gc, nil
}
`
