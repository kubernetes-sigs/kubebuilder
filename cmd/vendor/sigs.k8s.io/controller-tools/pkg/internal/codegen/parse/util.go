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

package parse

import (
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/gengo/types"
)

// Options contains the parser options
type Options struct {
	SkipMapValidation bool

	// SkipRBACValidation flag determines whether to check RBAC annotations
	// for the controller or not at parse stage.
	SkipRBACValidation bool
}

// IsAPIResource returns true if:
// 1. t has a +resource/+kubebuilder:resource comment tag
// 2. t has TypeMeta and ObjectMeta in its member list.
func IsAPIResource(t *types.Type) bool {
	for _, c := range t.CommentLines {
		if strings.Contains(c, "+resource") || strings.Contains(c, "+kubebuilder:resource") {
			return true
		}
	}

	typeMetaFound, objMetaFound := false, false
	for _, m := range t.Members {
		if m.Name == "TypeMeta" && m.Type.String() == "k8s.io/apimachinery/pkg/apis/meta/v1.TypeMeta" {
			typeMetaFound = true
		}
		if m.Name == "ObjectMeta" && m.Type.String() == "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta" {
			objMetaFound = true
		}
		if typeMetaFound && objMetaFound {
			return true
		}
	}
	return false
}

// IsNonNamespaced returns true if t has a +nonNamespaced comment tag
func IsNonNamespaced(t *types.Type) bool {
	if !IsAPIResource(t) {
		return false
	}

	for _, c := range t.CommentLines {
		if strings.Contains(c, "+genclient:nonNamespaced") {
			return true
		}
	}

	for _, c := range t.SecondClosestCommentLines {
		if strings.Contains(c, "+genclient:nonNamespaced") {
			return true
		}
	}

	return false
}

// IsController returns true if t has a +controller or +kubebuilder:controller tag
func IsController(t *types.Type) bool {
	for _, c := range t.CommentLines {
		if strings.Contains(c, "+controller") || strings.Contains(c, "+kubebuilder:controller") {
			return true
		}
	}
	return false
}

// IsRBAC returns true if t has a +rbac or +kubebuilder:rbac tag
func IsRBAC(t *types.Type) bool {
	for _, c := range t.CommentLines {
		if strings.Contains(c, "+rbac") || strings.Contains(c, "+kubebuilder:rbac") {
			return true
		}
	}
	return false
}

// IsInformer returns true if t has a +informers or +kubebuilder:informers tag
func IsInformer(t *types.Type) bool {
	for _, c := range t.CommentLines {
		if strings.Contains(c, "+informers") || strings.Contains(c, "+kubebuilder:informers") {
			return true
		}
	}
	return false
}

// IsAPISubresource returns true if t has a +subresource-request comment tag
func IsAPISubresource(t *types.Type) bool {
	for _, c := range t.CommentLines {
		if strings.Contains(c, "+subresource-request") {
			return true
		}
	}
	return false
}

// HasSubresource returns true if t is an APIResource with one or more Subresources
func HasSubresource(t *types.Type) bool {
	if !IsAPIResource(t) {
		return false
	}
	for _, c := range t.CommentLines {
		if strings.Contains(c, "+subresource") {
			return true
		}
	}
	return false
}

// HasStatusSubresource returns true if t is an APIResource annotated with
// +kubebuilder:subresource:status.
func HasStatusSubresource(t *types.Type) bool {
	if !IsAPIResource(t) {
		return false
	}
	for _, c := range t.CommentLines {
		if strings.Contains(c, "+kubebuilder:subresource:status") {
			return true
		}
	}
	return false
}

// HasCategories returns true if t is an APIResource annotated with
// +kubebuilder:categories.
func HasCategories(t *types.Type) bool {
	if !IsAPIResource(t) {
		return false
	}

	for _, c := range t.CommentLines {
		if strings.Contains(c, "+kubebuilder:categories") {
			return true
		}
	}
	return false
}

// HasDocAnnotation returns true if t is an APIResource with doc annotation
// +kubebuilder:doc
func HasDocAnnotation(t *types.Type) bool {
	if !IsAPIResource(t) {
		return false
	}
	for _, c := range t.CommentLines {
		if strings.Contains(c, "+kubebuilder:doc") {
			return true
		}
	}
	return false
}

// IsUnversioned returns true if t is in given group, and not in versioned path.
func IsUnversioned(t *types.Type, group string) bool {
	return IsApisDir(filepath.Base(filepath.Dir(t.Name.Package))) && GetGroup(t) == group
}

// IsVersioned returns true if t is in given group, and in versioned path.
func IsVersioned(t *types.Type, group string) bool {
	dir := filepath.Base(filepath.Dir(filepath.Dir(t.Name.Package)))
	return IsApisDir(dir) && GetGroup(t) == group
}

// GetVersion returns version of t.
func GetVersion(t *types.Type, group string) string {
	if !IsVersioned(t, group) {
		panic(errors.Errorf("Cannot get version for unversioned type %v", t.Name))
	}
	return filepath.Base(t.Name.Package)
}

// GetGroup returns group of t.
func GetGroup(t *types.Type) string {
	return filepath.Base(GetGroupPackage(t))
}

// GetGroupPackage returns group package of t.
func GetGroupPackage(t *types.Type) string {
	if IsApisDir(filepath.Base(filepath.Dir(t.Name.Package))) {
		return t.Name.Package
	}
	return filepath.Dir(t.Name.Package)
}

// GetKind returns kind of t.
func GetKind(t *types.Type, group string) string {
	if !IsVersioned(t, group) && !IsUnversioned(t, group) {
		panic(errors.Errorf("Cannot get kind for type not in group %v", t.Name))
	}
	return t.Name.Name
}

// IsApisDir returns true if a directory path is a Kubernetes api directory
func IsApisDir(dir string) bool {
	return dir == "apis" || dir == "api"
}

// Comments is a structure for using comment tags on go structs and fields
type Comments []string

// GetTags returns the value for the first comment with a prefix matching "+name="
// e.g. "+name=foo\n+name=bar" would return "foo"
func (c Comments) getTag(name, sep string) string {
	for _, c := range c {
		prefix := fmt.Sprintf("+%s%s", name, sep)
		if strings.HasPrefix(c, prefix) {
			return strings.Replace(c, prefix, "", 1)
		}
	}
	return ""
}

// hasTag returns true if the Comments has a tag with the given name
func (c Comments) hasTag(name string) bool {
	for _, c := range c {
		prefix := fmt.Sprintf("+%s", name)
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

// GetTags returns the value for all comments with a prefix and separator.  E.g. for "name" and "="
// "+name=foo\n+name=bar" would return []string{"foo", "bar"}
func (c Comments) getTags(name, sep string) []string {
	tags := []string{}
	for _, c := range c {
		prefix := fmt.Sprintf("+%s%s", name, sep)
		if strings.HasPrefix(c, prefix) {
			tags = append(tags, strings.Replace(c, prefix, "", 1))
		}
	}
	return tags
}

// getDocAnnotation parse annotations of "+kubebuilder:doc:" with tags of "warning" or "doc" for control generating doc config.
// E.g. +kubebuilder:doc:warning=foo  +kubebuilder:doc:note=bar
func getDocAnnotation(t *types.Type, tags ...string) map[string]string {
	annotation := make(map[string]string)
	for _, tag := range tags {
		for _, c := range t.CommentLines {
			prefix := fmt.Sprintf("+kubebuilder:doc:%s=", tag)
			if strings.HasPrefix(c, prefix) {
				annotation[tag] = strings.Replace(c, prefix, "", 1)
			}
		}
	}
	return annotation
}

// parseByteValue returns the literal digital number values from a byte array
func parseByteValue(b []byte) string {
	elem := strings.Join(strings.Fields(fmt.Sprintln(b)), ",")
	elem = strings.TrimPrefix(elem, "[")
	elem = strings.TrimSuffix(elem, "]")
	return elem
}

// parseEnumToString returns a representive validated go format string from JSONSchemaProps schema
func parseEnumToString(value []v1beta1.JSON) string {
	res := "[]v1beta1.JSON{"
	prefix := "v1beta1.JSON{[]byte{"
	for _, v := range value {
		res = res + prefix + parseByteValue(v.Raw) + "}},"
	}
	return strings.TrimSuffix(res, ",") + "}"
}

// check type of enum element value to match type of field
func checkType(props *v1beta1.JSONSchemaProps, s string, enums *[]v1beta1.JSON) {

	// TODO support more types check
	switch props.Type {
	case "int", "int64", "uint64":
		if _, err := strconv.ParseInt(s, 0, 64); err != nil {
			log.Fatalf("Invalid integer value [%v] for a field of integer type", s)
		}
		*enums = append(*enums, v1beta1.JSON{Raw: []byte(fmt.Sprintf("%v", s))})
	case "int32", "unit32":
		if _, err := strconv.ParseInt(s, 0, 32); err != nil {
			log.Fatalf("Invalid integer value [%v] for a field of integer32 type", s)
		}
		*enums = append(*enums, v1beta1.JSON{Raw: []byte(fmt.Sprintf("%v", s))})
	case "float", "float32":
		if _, err := strconv.ParseFloat(s, 32); err != nil {
			log.Fatalf("Invalid float value [%v] for a field of float32 type", s)
		}
		*enums = append(*enums, v1beta1.JSON{Raw: []byte(fmt.Sprintf("%v", s))})
	case "float64":
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			log.Fatalf("Invalid float value [%v] for a field of float type", s)
		}
		*enums = append(*enums, v1beta1.JSON{Raw: []byte(fmt.Sprintf("%v", s))})
	case "string":
		*enums = append(*enums, v1beta1.JSON{Raw: []byte(`"` + s + `"`)})
	}
}
