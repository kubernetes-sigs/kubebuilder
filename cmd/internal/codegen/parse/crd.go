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

package parse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/gengo/types"
)

// parseCRDs populates the CRD field of each Group.Version.Resource,
// creating validations using the annotations on type fields.
func (b *APIs) parseCRDs() {
	for _, group := range b.APIs.Groups {
		for _, version := range group.Versions {
			for _, resource := range version.Resources {
				if IsAPIResource(resource.Type) {
					resource.JSONSchemaProps, resource.Validation =
						b.typeToJSONSchemaProps(resource.Type, sets.NewString(), []string{})

					j, err := json.MarshalIndent(resource.JSONSchemaProps, "", "    ")
					if err != nil {
						log.Fatalf("Could not Marshall validation %v\n", err)
					}
					resource.ValidationComments = string(j)

					resource.CRD = v1beta1.CustomResourceDefinition{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "apiextensions.k8s.io/v1beta1",
							Kind:       "CustomResourceDefinition",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: fmt.Sprintf("%s.%s.%s", resource.Resource, resource.Group, resource.Domain),
						},
						Spec: v1beta1.CustomResourceDefinitionSpec{
							Group:   fmt.Sprintf("%s.%s", resource.Group, resource.Domain),
							Version: resource.Version,
							Names: v1beta1.CustomResourceDefinitionNames{
								Kind:   resource.Kind,
								Plural: resource.Resource,
							},
							Validation: &v1beta1.CustomResourceValidation{
								&resource.JSONSchemaProps,
							},
						},
					}
					if resource.NonNamespaced {
						resource.CRD.Spec.Scope = "Cluster"
					} else {
						resource.CRD.Spec.Scope = "Namespaced"
					}

					if HasCategories(resource.Type) {
						categoriesTag := getCategoriesTag(resource.Type)
						resource.CRD.Spec.Names.Categories = strings.Split(categoriesTag, ",")
					}

					if len(resource.ShortName) > 0 {
						resource.CRD.Spec.Names.ShortNames = []string{resource.ShortName}
					}
				}
			}
		}
	}
}

func (b *APIs) getTime() string {
	return `v1beta1.JSONSchemaProps{
    Type:   "string",
    Format: "date-time",
}`
}

func (b *APIs) getMeta() string {
	return `v1beta1.JSONSchemaProps{
    Type:   "object",
}`
}

// typeToJSONSchemaProps returns a JSONSchemaProps object and its serialization
// in Go that describe the JSONSchema validations for the given type.
func (b *APIs) typeToJSONSchemaProps(t *types.Type, found sets.String, comments []string) (v1beta1.JSONSchemaProps, string) {
	// Special cases
	time := types.Name{Name: "Time", Package: "k8s.io/apimachinery/pkg/apis/meta/v1"}
	meta := types.Name{Name: "ObjectMeta", Package: "k8s.io/apimachinery/pkg/apis/meta/v1"}
	switch t.Name {
	case time:
		return v1beta1.JSONSchemaProps{
			Type:   "string",
			Format: "date-time",
		}, b.getTime()
	case meta:
		return v1beta1.JSONSchemaProps{
			Type: "object",
		}, b.getMeta()
	}

	var v v1beta1.JSONSchemaProps
	var s string
	switch t.Kind {
	case types.Builtin:
		v, s = b.parsePrimitiveValidation(t, found, comments)
	case types.Struct:
		v, s = b.parseObjectValidation(t, found, comments)
	case types.Map:
		v, s = b.parseMapValidation(t, found, comments)
	case types.Slice:
		v, s = b.parseArrayValidation(t, found, comments)
	case types.Array:
		v, s = b.parseArrayValidation(t, found, comments)
	case types.Pointer:
		v, s = b.typeToJSONSchemaProps(t.Elem, found, comments)
	case types.Alias:
		v, s = b.typeToJSONSchemaProps(t.Underlying, found, comments)
	default:
		log.Fatalf("Unknown supported Kind %v\n", t.Kind)
	}

	return v, s
}

var jsonRegex = regexp.MustCompile("json:\"([a-zA-Z,]+)\"")

type primitiveTemplateArgs struct {
	v1beta1.JSONSchemaProps
	Value  string
	Format string
}

var primitiveTemplate = template.Must(template.New("map-template").Parse(
	`v1beta1.JSONSchemaProps{
    {{ if .Pattern -}}
    Pattern: "{{ .Pattern }}",
    {{ end -}}
    {{ if .Maximum -}}
    Maximum: getFloat({{ .Maximum }}),
    {{ end -}}
    {{ if .ExclusiveMaximum -}}
    ExclusiveMaximum: {{ .ExclusiveMaximum }},
    {{ end -}}
    {{ if .Minimum -}}
    Minimum: getFloat({{ .Minimum }}),
    {{ end -}}
    {{ if .ExclusiveMinimum -}}
    ExclusiveMinimum: {{ .ExclusiveMinimum }},
    {{ end -}}
    Type: "{{ .Value }}",
    {{ if .Format -}}
    Format: "{{ .Format }}",
    {{ end -}}
}`))

// parsePrimitiveValidation returns a JSONSchemaProps object and its
// serialization in Go that describe the validations for the given primitive
// type.
func (b *APIs) parsePrimitiveValidation(t *types.Type, found sets.String, comments []string) (v1beta1.JSONSchemaProps, string) {
	props := v1beta1.JSONSchemaProps{Type: string(t.Name.Name)}

	for _, l := range comments {
		getValidation(l, &props)
	}

	buff := &bytes.Buffer{}

	var n, f string
	switch t.Name.Name {
	case "int", "int64", "uint64":
		n = "integer"
		f = "int64"
	case "int32", "uint32":
		n = "integer"
		f = "int32"
	case "float", "float32":
		n = "number"
		f = "float"
	case "float64":
		n = "number"
		f = "double"
	case "bool":
		n = "boolean"
	case "string":
		n = "string"
	default:
		n = t.Name.Name
	}
	if err := primitiveTemplate.Execute(buff, primitiveTemplateArgs{props, n, f}); err != nil {
		log.Fatalf("%v", err)
	}

	return props, buff.String()
}

var mapTemplate = template.Must(template.New("map-template").Parse(
	`v1beta1.JSONSchemaProps{
    Type:                 "object",
    AdditionalProperties: &v1beta1.JSONSchemaPropsOrBool{
        Allows: true,
        //Schema: &{{.}},
    },
}`))

// parseMapValidation returns a JSONSchemaProps object and its serialization in
// Go that describe the validations for the given map type.
func (b *APIs) parseMapValidation(t *types.Type, found sets.String, comments []string) (v1beta1.JSONSchemaProps, string) {
	additionalProps, _ := b.typeToJSONSchemaProps(t.Elem, found, comments)
	props := v1beta1.JSONSchemaProps{
		Type: "object",
		AdditionalProperties: &v1beta1.JSONSchemaPropsOrBool{
			Allows: true,
			Schema: &additionalProps},
	}

	buff := &bytes.Buffer{}
	if err := mapTemplate.Execute(buff, ""); err != nil {
		log.Fatalf("%v", err)
	}
	return props, buff.String()
}

var arrayTemplate = template.Must(template.New("array-template").Parse(
	`v1beta1.JSONSchemaProps{
    Type:                 "array",
    Items: &v1beta1.JSONSchemaPropsOrArray{
        Schema: &{{.}},
    },
}`))

// parseArrayValidation returns a JSONSchemaProps object and its serialization in
// Go that describe the validations for the given array type.
func (b *APIs) parseArrayValidation(t *types.Type, found sets.String, comments []string) (v1beta1.JSONSchemaProps, string) {
	items, result := b.typeToJSONSchemaProps(t.Elem, found, comments)
	props := v1beta1.JSONSchemaProps{
		Type:  "array",
		Items: &v1beta1.JSONSchemaPropsOrArray{Schema: &items},
	}

	buff := &bytes.Buffer{}
	if err := arrayTemplate.Execute(buff, result); err != nil {
		log.Fatalf("%v", err)
	}
	return props, buff.String()
}

type objectTemplateArgs struct {
	v1beta1.JSONSchemaProps
	Fields map[string]string
}

var objectTemplate = template.Must(template.New("object-template").Parse(
	`v1beta1.JSONSchemaProps{
    Type:                 "object",
    Properties: map[string]v1beta1.JSONSchemaProps{
        {{ range $k, $v := .Fields -}}
        "{{ $k }}": {{ $v }},
        {{ end -}}
    },
}`))

// parseObjectValidation returns a JSONSchemaProps object and its serialization in
// Go that describe the validations for the given object type.
func (b *APIs) parseObjectValidation(t *types.Type, found sets.String, comments []string) (v1beta1.JSONSchemaProps, string) {
	buff := &bytes.Buffer{}
	props := v1beta1.JSONSchemaProps{
		Type: "object",
	}

	if strings.HasPrefix(t.Name.String(), "k8s.io/api") {
		if err := objectTemplate.Execute(buff, objectTemplateArgs{props, nil}); err != nil {
			log.Fatalf("%v", err)
		}
	} else {
		m, result := b.getMembers(t, found)
		props.Properties = m

		// Only add field validation for non-inlined fields
		for _, l := range comments {
			getValidation(l, &props)
		}

		if err := objectTemplate.Execute(buff, objectTemplateArgs{props, result}); err != nil {
			log.Fatalf("%v", err)
		}
	}
	return props, buff.String()
}

// getValidation parses the validation tags from the comment and sets the
// validation rules on the given JSONSchemaProps.
func getValidation(comment string, props *v1beta1.JSONSchemaProps) {
	comment = strings.TrimLeft(comment, " ")
	if !strings.HasPrefix(comment, "+kubebuilder:validation:") {
		return
	}
	log.Printf("Doing %s\n", comment)
	c := strings.Replace(comment, "+kubebuilder:validation:", "", -1)
	parts := strings.Split(c, "=")
	if len(parts) != 2 {
		log.Fatalf("Expected +kubebuilder:validation:<key>=<value> actual: %s", comment)
		return
	}
	log.Printf("Switch %v\n", parts)
	switch parts[0] {
	case "Maximum":
		f, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			log.Fatalf("Could not parse float from %s: %v", comment, err)
			return
		}
		props.Maximum = &f
	case "ExclusiveMaximum":
		b, err := strconv.ParseBool(parts[1])
		if err != nil {
			log.Fatalf("Could not parse bool from %s: %v", comment, err)
			return
		}
		props.ExclusiveMaximum = b
	case "Minimum":
		f, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			log.Fatalf("Could not parse float from %s: %v", comment, err)
			return
		}
		props.Minimum = &f
	case "ExclusiveMinimum":
		b, err := strconv.ParseBool(parts[1])
		if err != nil {
			log.Fatalf("Could not parse bool from %s: %v", comment, err)
			return
		}
		props.ExclusiveMinimum = b
	case "MaxLength":
		i, err := strconv.Atoi(parts[1])
		v := int64(i)
		if err != nil {
			log.Fatalf("Could not parse int from %s: %v", comment, err)
			return
		}
		props.MaxLength = &v
	case "MinLength":
		i, err := strconv.Atoi(parts[1])
		v := int64(i)
		if err != nil {
			log.Fatalf("Could not parse int from %s: %v", comment, err)
			return
		}
		props.MinLength = &v
	case "Pattern":
		props.Pattern = parts[1]
	case "MaxItems":
		i, err := strconv.Atoi(parts[1])
		v := int64(i)
		if err != nil {
			log.Fatalf("Could not parse int from %s: %v", comment, err)
			return
		}
		props.MaxItems = &v
	case "MinItems":
		i, err := strconv.Atoi(parts[1])
		v := int64(i)
		if err != nil {
			log.Fatalf("Could not parse int from %s: %v", comment, err)
			return
		}
		props.MinItems = &v
	case "UniqueItems":
		b, err := strconv.ParseBool(parts[1])
		if err != nil {
			log.Fatalf("Could not parse bool from %s: %v", comment, err)
			return
		}
		props.ExclusiveMinimum = b
	case "MultipleOf":
		f, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			log.Fatalf("Could not parse float from %s: %v", comment, err)
			return
		}
		props.MultipleOf = &f
	case "Enum":
		enums := strings.Split(parts[1], ",")
		for i := range enums {
			props.Enum = append(props.Enum, v1beta1.JSON{[]byte(enums[i])})
		}
	case "Format":
		props.Format = parts[1]
	default:
		log.Fatalf("Unsupport validation: %s", comment)
	}
}

// getMembers builds maps by field name of the JSONSchemaProps and their Go
// serializations.
func (b *APIs) getMembers(t *types.Type, found sets.String) (map[string]v1beta1.JSONSchemaProps, map[string]string) {
	members := map[string]v1beta1.JSONSchemaProps{}
	result := map[string]string{}

	// Don't allow recursion until we support it through refs
	// TODO: Support recursion
	if found.Has(t.Name.String()) {
		fmt.Printf("Breaking recursion for type %s", t.Name.String())
		return members, result
	}
	found.Insert(t.Name.String())

	for _, member := range t.Members {
		tags := jsonRegex.FindStringSubmatch(member.Tags)
		if len(tags) == 0 {
			// Skip fields without json tags
			//fmt.Printf("Skipping member %s %s\n", member.Name, member.Type.Name.String())
			continue
		}
		ts := strings.Split(tags[1], ",")
		name := member.Name
		strat := ""
		if len(ts) > 0 && len(ts[0]) > 0 {
			name = ts[0]
		}
		if len(ts) > 1 {
			strat = ts[1]
		}

		// Inline "inline" structs
		if strat == "inline" {
			m, r := b.getMembers(member.Type, found)
			for n, v := range m {
				members[n] = v
			}
			for n, v := range r {
				result[n] = v
			}
		} else {
			m, r := b.typeToJSONSchemaProps(member.Type, found, member.CommentLines)
			members[name] = m
			result[name] = r
		}
	}

	defer found.Delete(t.Name.String())
	return members, result
}

// getCategoriesTag returns the value of the +kubebuilder:categories tags
func getCategoriesTag(c *types.Type) string {
	comments := Comments(c.CommentLines)
	resource := comments.getTag("kubebuilder:categories", "=")
	if len(resource) == 0 {
		panic(errors.Errorf("Must specify +kubebuilder:categories comment for type %v", c.Name))
	}
	return resource
}
