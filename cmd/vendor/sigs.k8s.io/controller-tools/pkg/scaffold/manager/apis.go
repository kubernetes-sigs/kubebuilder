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

package manager

import (
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
)

var _ input.File = &APIs{}

// APIs scaffolds a apis.go to register types with a Scheme
type APIs struct {
	input.Input

	// Comments is a list of comments to add to the apis.go
	Comments []string
}

var deepCopy = strings.Join([]string{
	"//go:generate go run",
	"../../vendor/k8s.io/code-generator/cmd/deepcopy-gen/main.go",
	"-O zz_generated.deepcopy",
	"-i ./..."}, " ")

// GetInput implements input.File
func (a *APIs) GetInput() (input.Input, error) {
	if a.Path == "" {
		a.Path = filepath.Join("pkg", "apis", "apis.go")
	}

	b, err := filepath.Rel(filepath.Join(a.Input.ProjectPath, "pkg", "apis"), a.BoilerplatePath)
	if err != nil {
		return input.Input{}, err
	}
	if len(a.Comments) == 0 {
		a.Comments = append(a.Comments,
			"// Generate deepcopy for apis", fmt.Sprintf("%s -h %s", deepCopy, b))
	}
	a.TemplateBody = apisTemplate
	return a.Input, nil
}

var apisTemplate = `{{ .Boilerplate }}

{{ range $line := .Comments }}{{ $line }}
{{ end }}
// Package apis contains Kubernetes API groups.
package apis

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
`
