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

// Package rbac contain libraries for generating RBAC manifests from RBAC
// annotations in Go source files.
package rbac

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ParseDir parses the Go files under given directory and parses the RBAC
// annotations in to RBAC rules.
// TODO(droot): extend it to multiple dirs
func ParseDir(dir string) ([]rbacv1.PolicyRule, error) {
	var rbacRules []rbacv1.PolicyRule
	fset := token.NewFileSet()

	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if !isGoFile(info) {
				// TODO(droot): enable this output based on verbose flag
				// fmt.Println("skipping non-go file", path)
				return nil
			}
			rules, err := parseFile(fset, path, nil)
			if err == nil {
				rbacRules = append(rbacRules, rules...)
			}
			return err
		})
	return rbacRules, err
}

// filter function to ignore files from parsing.
func isGoFile(f os.FileInfo) bool {
	// ignore non-Go or Go test files
	name := f.Name()
	return !f.IsDir() &&
		!strings.HasPrefix(name, ".") &&
		!strings.HasSuffix(name, "_test.go") &&
		strings.HasSuffix(name, ".go")
}

// parseFile parses given filename or content src and parses RBAC annotations
// into RBAC rules.
func parseFile(fset *token.FileSet, filename string, src interface{}) ([]rbacv1.PolicyRule, error) {
	var rules []rbacv1.PolicyRule

	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// using commentMaps here because it sanitizes the comment text by removing
	// comment markers, compresses newlines etc.
	cmap := ast.NewCommentMap(fset, f, f.Comments)

	for _, commentGroup := range cmap.Comments() {
		for _, comment := range strings.Split(commentGroup.Text(), "\n") {
			comment := strings.TrimSpace(comment)
			if strings.HasPrefix(comment, "+rbac") {
				if ann := getAnnotation(comment, "rbac"); ann != "" {
					rules = append(rules, parseRBACTag(ann))
				}
			}
			if strings.HasPrefix(comment, "+kubebuilder:rbac") {
				if ann := getAnnotation(comment, "kubebuilder:rbac"); ann != "" {
					rules = append(rules, parseRBACTag(ann))
				}
			}
		}
	}
	return rules, nil
}

// getAnnotation extracts the RBAC annotation from comment text. It will return
// "foo" for comment "+rbac:foo" .
func getAnnotation(c, name string) string {
	prefix := fmt.Sprintf("+%s:", name)
	if strings.HasPrefix(c, prefix) {
		return strings.TrimPrefix(c, prefix)
	}
	return ""
}

// parseRBACTag parses the given RBAC annotation in to an RBAC PolicyRule.
// This is copied from Kubebuilder code.
func parseRBACTag(tag string) rbacv1.PolicyRule {
	result := rbacv1.PolicyRule{}
	for _, elem := range strings.Split(tag, ",") {
		key, value, err := parseKV(elem)
		if err != nil {
			log.Fatalf("// +kubebuilder:rbac: tags must be key value pairs.  Expected "+
				"keys [groups=<group1;group2>,resources=<resource1;resource2>,verbs=<verb1;verb2>] "+
				"Got string: [%s]", tag)
		}
		values := strings.Split(value, ";")
		switch key {
		case "groups":
			normalized := []string{}
			for _, v := range values {
				if v == "core" {
					normalized = append(normalized, "")
				} else {
					normalized = append(normalized, v)
				}
			}
			result.APIGroups = normalized
		case "resources":
			result.Resources = values
		case "verbs":
			result.Verbs = values
		case "urls":
			result.NonResourceURLs = values
		}
	}
	return result
}

func parseKV(s string) (key, value string, err error) {
	kv := strings.Split(s, "=")
	if len(kv) != 2 {
		err = fmt.Errorf("invalid key value pair")
		return key, value, err
	}
	key, value = kv[0], kv[1]
	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		value = value[1 : len(value)-1]
	}
	return key, value, err
}
