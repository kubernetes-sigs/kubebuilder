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

	"fmt"
	"github.com/kubernetes-sigs/kubebuilder/cmd/internal/codegen"
	"github.com/markbates/inflect"
	"k8s.io/gengo/generator"
	"path"
)

type injectGenerator struct {
	generator.DefaultGen
	Controllers []codegen.Controller
	APIS        *codegen.APIs
}

var _ generator.Generator = &injectGenerator{}

func (d *injectGenerator) Imports(c *generator.Context) []string {
	if len(d.Controllers) == 0 {
		return []string{}
	}

	repo := d.Controllers[0].Repo
	im := []string{
		"github.com/kubernetes-sigs/kubebuilder/pkg/controller",
		"k8s.io/client-go/rest",
		repo + "/pkg/controller/sharedinformers",
		repo + "/pkg/client/informers/externalversions",
		repo + "/pkg/inject/args",
		"rbacv1 \"k8s.io/api/rbac/v1\"",
	}

	if len(d.APIS.Groups) > 0 {
	    im = append(im, []string{
	    	"time",
	    	"k8s.io/client-go/kubernetes/scheme",
	    	"rscheme " + "\"" + repo + "/pkg/client/clientset/versioned/scheme\""}...
	    )
    }
	// Import package for each controller
	repos := map[string]string{}
	for _, c := range d.Controllers {
		repos[c.Pkg.Path] = ""
	}
	for k, _ := range repos {
		im = append(im, k)
	}

	libs := map[string]string{}
	for i := range d.APIS.Informers {
		libs[i.Group+i.Version] = "k8s.io/api/" + i.Group + "/" + i.Version
	}

	for i, d := range libs {
		im = append(im, fmt.Sprintf("%s \"%s\"", i, d))
	}

	// Import package for each API groupversion
	gvk := map[string]string{}
	for _, g := range d.APIS.Groups {
		for _, v := range g.Versions {
			k := fmt.Sprintf("%s%s \"%s\"", g.Group, v.Version,
				path.Join(repo, "pkg", "apis", g.Group, v.Version))
			gvk[k] = ""
		}
	}
	for k, _ := range gvk {
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
    {{ $length := len .APIS.Groups }}{{if ne $length 0 }}rscheme.AddToScheme(scheme.Scheme){{ end }}

    // Inject Informers
    Inject = append(Inject, func(arguments args.InjectArgs) error {
	    Injector.ControllerManager = arguments.ControllerManager

        {{ range $group := .APIS.Groups }}{{ range $version := $group.Versions }}{{ range $res := $version.Resources -}}
        if err := arguments.ControllerManager.AddInformerProvider(&{{.Group}}{{.Version}}.{{.Kind}}{}, arguments.Informers.{{title .Group}}().{{title .Version}}().{{plural .Kind}}()); err != nil {
            return err
        }
        {{ end }}{{ end }}{{ end }}

        // Add Kubernetes informers
        {{ range $informer, $found := .APIS.Informers -}}
        if err := arguments.ControllerManager.AddInformerProvider(&{{$informer.Group}}{{$informer.Version}}.{{$informer.Kind}}{}, arguments.KubernetesInformers.{{title $informer.Group}}().{{title $informer.Version}}().{{plural $informer.Kind}}()); err != nil {
            return err
        }
        {{ end }}

        {{ range $c := .Controllers -}}
        if c, err := {{ $c.Pkg.Name }}.ProvideController(arguments); err != nil {
            return err
        } else {
            arguments.ControllerManager.AddController(c)
        }
        {{ end -}}
        return nil
    })

    // Inject CRDs
    {{ range $group := .APIS.Groups -}}
    {{ range $version := $group.Versions -}}
    {{ range $res := $version.Resources -}}
    Injector.CRDs = append(Injector.CRDs, &{{ $group.Group }}{{ $version.Version }}.{{$res.Kind}}CRD)
    {{ end }}{{ end }}{{ end -}}


    // Inject PolicyRules
    {{ range $group := .APIS.Groups -}}
    Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
        APIGroups: []string{"{{ $group.Group }}.{{ $group.Domain }}"},
        Resources: []string{"*"},
        Verbs: []string{"*"},
    })
    {{ end -}}
    {{ range $rule := .APIS.GetRules -}}
    Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
        APIGroups: []string{
            {{ range $group := $rule.APIGroups -}}"{{ $group }}",{{ end }}
        },
        Resources: []string{
            {{ range $resource := $rule.Resources -}}"{{ $resource }}",{{ end }}
        },
        Verbs: []string{
            {{ range $rule := $rule.Verbs -}}"{{ $rule }}",{{ end }}
        },
    })
    {{ end -}}


    // Inject GroupVersions
    {{ range $group := .APIS.Groups -}}
    {{ range $version := $group.Versions -}}
    Injector.GroupVersions = append(Injector.GroupVersions, schema.GroupVersion{
        Group: "{{ $group.Group }}.{{ $group.Domain }}",
        Version: "{{ $version.Version }}",
    })
    {{ end }}{{ end -}}

	Injector.RunFns = append(Injector.RunFns, func(arguments run.RunArguments) error {
	    Injector.ControllerManager.RunInformersAndControllers(arguments)
	    return nil
    })
}
`
