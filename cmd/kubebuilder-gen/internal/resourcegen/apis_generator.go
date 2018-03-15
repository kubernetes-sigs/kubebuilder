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

package resourcegen

import (
	"io"
	"text/template"

	"fmt"

	"github.com/kubernetes-sigs/kubebuilder/cmd/internal/codegen"
	"k8s.io/gengo/generator"
)

type apiGenerator struct {
	generator.DefaultGen
	apis *codegen.APIs
}

var _ generator.Generator = &apiGenerator{}

func (d *apiGenerator) Imports(c *generator.Context) []string {
	imports := []string{
		"k8s.io/apimachinery/pkg/runtime/schema",
		"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1",
		"github.com/kubernetes-sigs/kubebuilder/pkg/builders",
		"rbacv1 \"k8s.io/api/rbac/v1\"",
	}
	for _, group := range d.apis.Groups {
		imports = append(imports, group.PkgPath)
		for _, version := range group.Versions {
			imports = append(imports, fmt.Sprintf(
				"%s%s \"%s\"", group.Group, version.Version, version.Pkg.Path))
		}
	}
	return imports
}

func (d *apiGenerator) Finalize(context *generator.Context, w io.Writer) error {
	temp := template.Must(template.New("apis-template").Parse(APIsTemplate))
	err := temp.Execute(w, d.apis)
	if err != nil {
		return err
	}
	return err
}

var APIsTemplate = `
type MetaData struct {}
var DefaultMetaData = MetaData{}

// GetCRDs returns all the CRDs for known resource types
func (MetaData) GetCRDs() []v1beta1.CustomResourceDefinition {
    return []v1beta1.CustomResourceDefinition{
    {{ range $group := .Groups -}}
    {{ range $version := $group.Versions -}}
    {{ range $res := $version.Resources -}}
        {{ $group.Group }}{{ $version.Version }}.{{$res.Kind}}CRD,
    {{ end }}{{ end }}{{ end -}}
    }
}

func (MetaData) GetRules() []rbacv1.PolicyRule {
    return []rbacv1.PolicyRule{
        {{ range $group := .Groups -}}
        {
            APIGroups: []string{"{{ $group.Group }}.{{ $group.Domain }}"},
	        Resources: []string{"*"},
	        Verbs: []string{"*"},
        },
        {{ end -}}
        {{ range $rule := .GetRules -}}
        {
            APIGroups: []string{
                {{ range $group := $rule.APIGroups -}}"{{ $group }}",{{ end }}
            },
	        Resources: []string{
                {{ range $resource := $rule.Resources -}}"{{ $resource }}",{{ end }}
            },
	        Verbs: []string{
                {{ range $rule := $rule.Verbs -}}"{{ $rule }}",{{ end }}
            },
        },
        {{ end -}}
    }
}

func (MetaData) GetGroupVersions() []schema.GroupVersion {
    return []schema.GroupVersion{
    {{ range $group := .Groups -}}
    {{ range $version := $group.Versions -}}
    {
        Group: "{{ $group.Group }}.{{ $group.Domain }}",
        Version: "{{ $version.Version }}",
    },
    {{ end }}{{ end -}}
    }
}
`
