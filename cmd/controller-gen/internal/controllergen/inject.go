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

package controllergen

import (
	"io"
	"strings"
	"text/template"

	"github.com/kubernetes-sigs/kubebuilder/cmd/internal/codegen"
	"github.com/markbates/inflect"
	"k8s.io/gengo/generator"
)

type injectGenerator struct {
	generator.DefaultGen
	Controllers []codegen.Controller
}

var _ generator.Generator = &injectGenerator{}

func (d *injectGenerator) Imports(c *generator.Context) []string {
	if len(d.Controllers) == 0 {
		return []string{}
	}

	repo := d.Controllers[0].Repo
	im := []string{
		"time",
		"github.com/kubernetes-sigs/kubebuilder/pkg/controller",
		"k8s.io/client-go/rest",
		repo + "/pkg/controller/sharedinformers",
		repo + "/pkg/client/informers/externalversions",
		repo + "/pkg/inject/args",
		"rbacv1 \"k8s.io/api/rbac/v1\"",
	}

	// Import package for each controller
	repos := map[string]string{}
	for _, c := range d.Controllers {
		repos[c.Pkg.Path] = ""
	}
	for k, _ := range repos {
		im = append(im, k)
	}

	return im
}

func (d *injectGenerator) Finalize(context *generator.Context, w io.Writer) error {
	temp := template.Must(template.New("all-controller-template").Funcs(
		template.FuncMap{
			"title":  strings.Title,
			"plural": inflect.NewDefaultRuleset().Pluralize,
		},
	).Parse(injectAPITemplate))
	return temp.Execute(w, d)
}

var injectAPITemplate = `
func init() {

    Inject = append(Inject, func(arguments args.InjectArgs) error {
	    Injector.ControllerManager = arguments.ControllerManager

        {{ range $c := .Controllers -}}
        if c, err := {{ $c.Pkg.Name }}.ProvideController(arguments); err != nil {
            return err
        } else {
            arguments.ControllerManager.AddController(c)
        }
        {{ end -}}
        return nil
    })

	Injector.RunFns = append(Injector.RunFns, func(arguments run.RunArguments) error {
	    Injector.ControllerManager.RunInformersAndControllers(arguments)
	    return nil
    })
}
`
