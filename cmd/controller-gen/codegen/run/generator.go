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

package run

import (
	"github.com/golang/glog"
	"github.com/kubernetes-sigs/kubebuilder/cmd/internal/codegen/parse"
	"github.com/kubernetes-sigs/kubebuilder/cmd/controller-gen/codegen"
	"k8s.io/gengo/args"
	"k8s.io/gengo/generator"
)

// CodeGenerator generates code for Kubernetes resources and controllers
type CodeGenerator struct {
	controllerGenerators []codegen.ControllerGenerator

	// OutputFileBaseName is the base name used for output files
	OutputFileBaseName string
}

// AddControllerGenerator adds a controller generator that will be called with parsed controllers
func (g *CodeGenerator) AddControllerGenerator(generator codegen.ControllerGenerator) *CodeGenerator {
	g.controllerGenerators = append(g.controllerGenerators, generator)
	return g
}

type customArgs struct{}

// Execute parses packages and executes the code generators against the resource and controller packages
func (g *CodeGenerator) Execute() error {
	arguments := args.Default()
	arguments.CustomArgs = &customArgs{}
	arguments.OutputFileBaseName = g.OutputFileBaseName

	err := arguments.Execute(parse.NameSystems(), parse.DefaultNameSystem(), g.packages)
	if err != nil {
		glog.Fatalf("Error: %v", err)
	}
	return nil
}

// packages parses the observed packages and creates code generators
func (g *CodeGenerator) packages(context *generator.Context, arguments *args.GeneratorArgs) generator.Packages {
	p := packages{}

	b := parse.NewControllers(context, arguments)

	repo := ""
	for _, c := range b.Controllers {
		repo = c.Repo
	}

	// Do inject package
	if len(b.Controllers) > 0 {
		for _, cg := range g.controllerGenerators {
			g := cg.GenerateInject(b.Controllers, arguments.OutputFileBaseName)
			if g != nil {
				p.add(context.Universe[repo+"/pkg/inject"].Path, g)
			}
		}
	}

	return p.value
}
