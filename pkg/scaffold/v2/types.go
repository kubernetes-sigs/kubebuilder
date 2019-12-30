/*
Copyright 2018 The Kubernetes Authors.

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

package v2

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
	"unicode"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
)

var _ input.File = &Types{}

// Types scaffolds the api/<version>/<kind>_types.go file to define the schema for an API
type Types struct {
	input.Input

	// Resource is the resource to scaffold the types_test.go file for
	Resource *resource.Resource
}

// GetInput implements input.File
func (t *Types) GetInput() (input.Input, error) {
	if t.Path == "" {
		t.Path = filepath.Join("pkg", "apis", t.Resource.Group, t.Resource.Version,
			fmt.Sprintf("%s_types.go", strings.ToLower(t.Resource.Kind)))
	}
	tmpl, err := t.applyStackfile(typesTemplate)
	if err != nil {
		return input.Input{}, err
	}
	t.TemplateBody = tmpl
	t.IfExistsAction = input.Error
	return t.Input, nil
}

// Validate validates the values
func (t *Types) Validate() error {
	return t.Resource.Validate()
}

type Stackfile struct {
	CRDTemplate `json:"crdTemplate"`
}

type CRDTemplate struct {
	Fields []CRDField `json:"fields,omitempty"`
}

type CRDField struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type"`
	Comment string `json:"comment"`
	Package string `json:"package,omitempty"`
}

func (t *Types) applyStackfile(template string) (string, error) {
	stackFileBytes, err := ioutil.ReadFile(filepath.Join("", "Stackfile.yaml"))
	if err != nil {
		return "", err
	}
	stackFile := &Stackfile{}
	if err := yaml.Unmarshal(stackFileBytes, stackFile); err != nil {
		return "", err
	}
	template, err = injectCustomCRDFields(*stackFile, template)
	if err != nil {
		return "", err
	}
	return template, nil
}

func injectCustomCRDFields(stackFile Stackfile, template string) (string, error) {
	var fieldStrings []string
	importedPackages := map[string]string{
		"k8s.io/apimachinery/pkg/apis/meta/v1": "metav1",
		"github.com/crossplaneio/crossplane-runtime/apis/core/v1alpha1": "runtime",
	}
	packageLabels := map[string]bool{
		"metav1": true,
		"runtime": true,
	}
	for _, field := range stackFile.Fields {
		typeName := field.Type
		label, ok := importedPackages[field.Package]
		if !ok && field.Package != "" {
			label = generateUniqueLabel(field.Package, packageLabels)
			importedPackages[field.Package] = label
			packageLabels[label] = true
		}
		if label != "" {
			typeName = fmt.Sprintf("%v.%v", label, typeName)
		}
		runes := []rune(field.Name)
		runes[0] = unicode.ToUpper(runes[0])
		fieldName := string(runes)
		fieldStrings = append(fieldStrings, fmt.Sprintf(`
	// %v
	%v %v ` + "`" + `json:"%v"` + "`" + `
`, field.Comment, fieldName, typeName, field.Name))
	}
	allFields := strings.Join(fieldStrings, "")
	result := strings.Replace(template, "<PLACEHOLDER_FOR_CRD_TYPES>", allFields, -1)
	var importStrings []string
	for path, label := range importedPackages {
		importStrings = append(importStrings, fmt.Sprintf(`
	%v "%v"`, label, path))
	}
	allImports := strings.Join(importStrings, "")
	return strings.Replace(result, "<PLACEHOLDER_FOR_GO_IMPORTS>", allImports, -1), nil
}

func generateUniqueLabel(packagePath string, existingLabels map[string]bool) string {
	// I know this is inefficient...
	elements := strings.Split(packagePath, "/")
	candidate := elements[len(elements)-1]
	for i := len(elements)-2; i >= 0; i-- {
		if _, ok := existingLabels[candidate]; !ok {
			return candidate
		}
		candidate = fmt.Sprintf("%s%s", elements[i], candidate)
	}
	return candidate
}

const typesTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
<PLACEHOLDER_FOR_GO_IMPORTS>
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// {{.Resource.Kind}}Spec defines the desired state of {{.Resource.Kind}}
type {{.Resource.Kind}}Spec struct {
<PLACEHOLDER_FOR_CRD_TYPES>
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// {{.Resource.Kind}}Status defines the observed state of {{.Resource.Kind}}
type {{.Resource.Kind}}Status struct {
	runtime.ConditionedStatus ` + "`" + `json:",inline"` + "`" + `
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
{{ if not .Resource.Namespaced }} // +kubebuilder:resource:scope=Cluster {{ end }}

// {{.Resource.Kind}} is the Schema for the {{ .Resource.Resource }} API
type {{.Resource.Kind}} struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`" + `
	metav1.ObjectMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `

	Spec   {{.Resource.Kind}}Spec   ` + "`" + `json:"spec,omitempty"` + "`" + `
	Status {{.Resource.Kind}}Status ` + "`" + `json:"status,omitempty"` + "`" + `
}

func (in *{{.Resource.Kind}}) GetCondition(ct runtime.ConditionType) runtime.Condition {
	return in.Status.GetCondition(ct)
}

func (in *{{.Resource.Kind}}) SetConditions(c ...runtime.Condition) {
	in.Status.SetConditions(c...)
}

// +kubebuilder:object:root=true

// {{.Resource.Kind}}List contains a list of {{.Resource.Kind}}
type {{.Resource.Kind}}List struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`" + `
	metav1.ListMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `
	Items           []{{ .Resource.Kind }} ` + "`" + `json:"items"` + "`" + `
}

func init() {
	SchemeBuilder.Register(&{{.Resource.Kind}}{}, &{{.Resource.Kind}}List{})
}
`
