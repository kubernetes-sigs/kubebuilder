/*
Copyright 2020 The Kubernetes Authors.

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

package machinery

import (
	"fmt"
	"hash/fnv"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
)

// DefaultFuncMap returns the default template.FuncMap for rendering the template.
func DefaultFuncMap() template.FuncMap {
	return template.FuncMap{
		"title":      cases.Title,
		"lower":      strings.ToLower,
		"upper":      strings.ToUpper,
		"isEmptyStr": isEmptyString,
		"hashFNV":    hashFNV,
	}
}

// isEmptyString returns whether the string is empty
func isEmptyString(s string) bool {
	return s == ""
}

// hashFNV will generate a random string useful for generating a unique string
func hashFNV(s string) string {
	hasher := fnv.New32a()
	// Hash.Write never returns an error
	_, _ = hasher.Write([]byte(s))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}
