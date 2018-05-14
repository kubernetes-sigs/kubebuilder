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
    "github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
    injectargs "github.com/kubernetes-sigs/kubebuilder/pkg/inject/args"

    "{{.Repo}}/pkg/inject/args"
)

var (
    // Inject is used to add items to the Injector
    Inject []func(args.InjectArgs) error

    // Injector runs items
    Injector injectargs.Injector
)

// RunAll starts all of the informers and Controllers
func RunAll(rargs run.RunArguments, iargs args.InjectArgs) error {
    // Run functions to initialize injector
    for _, i := range Inject {
        if err := i(iargs); err != nil {
            return err
        }
    }
    Injector.Run(rargs)
    <-rargs.Stop
    return nil
}
`

func doArgs(boilerplate string, controllerOnly bool) bool {
	args := templateArgs{
		Repo:        util.Repo,
		BoilerPlate: boilerplate,
		ControllerOnly: controllerOnly,
	}
	path := filepath.Join("pkg", "inject", "args", "args.go")
	fmt.Printf("\t%s\n", filepath.Join(
		"pkg", "inject", "args", "args.go"))
	return util.WriteIfNotFound(path, "args-controller-template", argsControllerTemplate, args)
}

var argsControllerTemplate = `{{.BoilerPlate}}

package args

import (
    {{ if not .ControllerOnly }}"time"{{ end }}

	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/args"
    "k8s.io/client-go/rest"
    {{ if not .ControllerOnly }}
    clientset "{{.Repo}}/pkg/client/clientset/versioned"
    informer "{{.Repo}}/pkg/client/informers/externalversions"
    {{ end }}
)

// InjectArgs are the arguments need to initialize controllers
type InjectArgs struct {
    args.InjectArgs
    {{ if not .ControllerOnly }}
    Clientset *clientset.Clientset
    Informers informer.SharedInformerFactory
    {{ end }}
}


// CreateInjectArgs returns new controller args
func CreateInjectArgs(config *rest.Config) InjectArgs {
    {{ if not .ControllerOnly }}cs := clientset.NewForConfigOrDie(config){{ end }}
    return InjectArgs{
        InjectArgs: args.CreateInjectArgs(config),
        {{ if not .ControllerOnly }}
        Clientset: cs,
        Informers: informer.NewSharedInformerFactory(cs, 2 * time.Minute), {{ end }}
    }
}
`
