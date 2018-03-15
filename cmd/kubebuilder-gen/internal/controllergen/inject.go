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
		"time",
		"github.com/kubernetes-sigs/kubebuilder/pkg/controller",
		"k8s.io/client-go/rest",
		"github.com/kubernetes-sigs/kubebuilder/pkg/controller",
		repo + "/pkg/controller/sharedinformers",
		repo + "/pkg/client/informers_generated/externalversions",
		repo + "/pkg/inject/args",
		" rbacv1 \"k8s.io/api/rbac/v1\"",
		"k8s.io/apimachinery/pkg/runtime/schema",
	}

	// Import package for each controller
	repos := map[string]string{}
	for _, c := range d.Controllers {
		repos[c.Pkg.Path] = ""
	}
	for k, _ := range repos {
		im = append(im, k)
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
    // Inject Informers
    SetInformers = func(arguments args.InjectArgs, factory externalversions.SharedInformerFactory) {
        {{ range $group := .APIS.Groups }}{{ range $version := $group.Versions }}{{ range $res := $version.Resources -}}
        arguments.ControllerManager.AddInformerProvider(&{{.Group}}{{.Version}}.{{.Kind}}{}, factory.{{title .Group}}().{{title .Version}}().{{plural .Kind}}())
        {{ end }}{{ end }}{{ end -}}
    }


    // Inject Controllers
    {{ range $c := .Controllers -}}
    Controllers = append(Controllers, {{ $c.Pkg.Name }}.ProvideController)
    {{ end -}}


    // Inject CRDs
    {{ range $group := .APIS.Groups -}}
    {{ range $version := $group.Versions -}}
    {{ range $res := $version.Resources -}}
    CRDs = append(CRDs, &{{ $group.Group }}{{ $version.Version }}.{{$res.Kind}}CRD)
    {{ end }}{{ end }}{{ end -}}


    // Inject PolicyRules
    {{ range $group := .APIS.Groups -}}
    PolicyRules = append(PolicyRules, rbacv1.PolicyRule{
        APIGroups: []string{"{{ $group.Group }}.{{ $group.Domain }}"},
        Resources: []string{"*"},
        Verbs: []string{"*"},
    })
    {{ end -}}
    {{ range $rule := .APIS.GetRules -}}
    PolicyRules = append(PolicyRules, rbacv1.PolicyRule{
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
    GroupVersions = append(GroupVersions, schema.GroupVersion{
        Group: "{{ $group.Group }}.{{ $group.Domain }}",
        Version: "{{ $version.Version }}",
    })
    {{ end }}{{ end -}}
}
`
