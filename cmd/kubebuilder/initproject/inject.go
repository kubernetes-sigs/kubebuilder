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

package initproject

import (
	"fmt"
	"path/filepath"

	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
)

func doInject(boilerplate string) bool {
	args := templateArgs{
		Repo:        util.Repo,
		BoilerPlate: boilerplate,
	}
	path := filepath.Join("pkg", "inject", "inject.go")
	fmt.Printf("\t%s\n", filepath.Join(
		"pkg", "inject", "inject.go"))
	return util.WriteIfNotFound(path, "inject-controller-template", injectControllerTemplate, args)
}

var injectControllerTemplate = `{{.BoilerPlate}}

package inject

import (
    "time"

    "github.com/kubernetes-sigs/kubebuilder/pkg/controller"
    "github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
    apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
    rbacv1 "k8s.io/api/rbac/v1"
    "k8s.io/apimachinery/pkg/runtime/schema"

    "{{.Repo}}/pkg/inject/args"
    "{{.Repo}}/pkg/client/informers_generated/externalversions"

)

var (
    CRDs = []*apiextensionsv1beta1.CustomResourceDefinition{}

    PolicyRules = []rbacv1.PolicyRule{}

    GroupVersions = []schema.GroupVersion{}


    // Controllers provides the controllers to run
    // Should be set by code generation in this package.
    Controllers  = []func(args args.InjectArgs) (*controller.GenericController, error){}

    RunningControllers = map[string]*controller.GenericController{}

    // SetInformers adds the informers for the apis defined in this project.
    // Should be set by code generation in this package.
    SetInformers func(args.InjectArgs, externalversions.SharedInformerFactory)

    // Inject may be set by generated code 
    Inject func(args.InjectArgs) error

    // Run may be set by generated code
    Run func(run.RunArguments) error
)

// RunAll starts all of the informers and Controllers
func RunAll(options run.RunArguments, arguments args.InjectArgs) error {
    if Inject != nil {
        if err := Inject(arguments); err != nil {
            return err
        }
    }
    if SetInformers != nil {
        factory := externalversions.NewSharedInformerFactory(arguments.Clientset, time.Minute * 5)
        SetInformers(arguments, factory)
    }
    for _, fn := range Controllers {
        if c, err := fn(arguments); err != nil {
            return err
        } else {
            arguments.ControllerManager.AddController(c)
        }
    }
    arguments.ControllerManager.RunInformersAndControllers(options)
    
    if Run != nil {
        if err := Run(options); err != nil {
            return err
        }
    }
    <-options.Stop
    return nil
}

type Injector struct {}

func (Injector) GetCRDs() []*apiextensionsv1beta1.CustomResourceDefinition {return CRDs}
func (Injector) GetPolicyRules() []rbacv1.PolicyRule {return PolicyRules}
func (Injector) GetGroupVersions() []schema.GroupVersion {return GroupVersions}
`

func doArgs(boilerplate string) bool {
	args := templateArgs{
		Repo:        util.Repo,
		BoilerPlate: boilerplate,
	}
	path := filepath.Join("pkg", "inject", "args", "args.go")
	fmt.Printf("\t%s\n", filepath.Join(
		"pkg", "inject", "args", "args.go"))
	return util.WriteIfNotFound(path, "args-controller-template", argsControllerTemplate, args)
}

var argsControllerTemplate = `{{.BoilerPlate}}

package args

import (
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/args"
    "k8s.io/client-go/rest"

    "{{.Repo}}/pkg/client/clientset_generated/clientset"
)

// InjectArgs are the arguments need to initialize controllers
type InjectArgs struct {
    args.InjectArgs

    Clientset *clientset.Clientset
}


// CreateInjectArgs returns new controller args
func CreateInjectArgs(config *rest.Config) InjectArgs {
    return InjectArgs{
        InjectArgs: args.CreateInjectArgs(config),
        Clientset: clientset.NewForConfigOrDie(config),
    }
}
`
