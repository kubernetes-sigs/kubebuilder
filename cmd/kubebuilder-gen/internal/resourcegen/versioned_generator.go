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

	"github.com/kubernetes-sigs/kubebuilder/cmd/internal/codegen"
	"k8s.io/gengo/generator"
)

type versionedGenerator struct {
	generator.DefaultGen
	apiversion *codegen.APIVersion
	apigroup   *codegen.APIGroup
}

var _ generator.Generator = &versionedGenerator{}

//func hasSubresources(version *codegen.APIVersion) bool {
//	for _, v := range version.Resources {
//		if len(v.Subresources) != 0 {
//			return true
//		}
//	}
//	return false
//}

func (d *versionedGenerator) Imports(c *generator.Context) []string {
	imports := []string{
		"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1",
		"metav1 \"k8s.io/apimachinery/pkg/apis/meta/v1\"",
		"k8s.io/apimachinery/pkg/runtime",
		"k8s.io/apimachinery/pkg/runtime/schema",
		d.apigroup.Pkg.Path,
	}
	//if hasSubresources(d.apiversion) {
	//	imports = append(imports, "k8s.io/apiserver/pkg/registry/rest")
	//}

	return imports
}

func (d *versionedGenerator) Finalize(context *generator.Context, w io.Writer) error {
	temp := template.Must(template.New("versioned-template").Parse(versionedAPITemplate))
	return temp.Execute(w, d.apiversion)
}

var versionedAPITemplate = `
// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: "{{.Group}}.{{.Domain}}", Version: "{{.Version}}"}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
    {{ range $api := .Resources -}}
		&{{.Kind}}{},
		&{{.Kind}}List{},
    {{ end -}}
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

{{ range $api := .Resources -}}
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type {{$api.Kind}}List struct {
    metav1.TypeMeta ` + "`json:\",inline\"`" + `
    metav1.ListMeta ` + "`json:\"metadata,omitempty\"`" + `
    Items           []{{$api.Kind}} ` + "`json:\"items\"`" + `
}
{{ end }}

// CRD Generation
func getFloat(f float64) *float64 {
    return &f
}

var (
    {{ range $api := .Resources -}}
    // Define CRDs for resources
    {{.Kind}}CRD = v1beta1.CustomResourceDefinition{
        ObjectMeta: metav1.ObjectMeta{
            Name: "{{.Resource}}.{{.Group}}.{{.Domain}}",
        },
        Spec: v1beta1.CustomResourceDefinitionSpec {
            Group: "{{.Group}}.{{.Domain}}",
            Version: "{{.Version}}",
            Names: v1beta1.CustomResourceDefinitionNames{
                Kind: "{{.Kind}}",
                Plural: "{{.Resource}}",
                {{ if .ShortName -}}
                ShortNames: []string{"{{.ShortName}}"},
                {{ end -}}
            },
            {{ if .NonNamespaced -}}
            Scope: "Cluster",
            {{ else -}}
            Scope: "Namespaced",
            {{ end -}}
            Validation: &v1beta1.CustomResourceValidation{
                OpenAPIV3Schema: &{{.Validation}},
            },
        },
    }
    {{ end -}}
)
`
