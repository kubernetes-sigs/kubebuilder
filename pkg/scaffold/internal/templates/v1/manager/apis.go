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

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &APIs{}

// APIs scaffolds a apis.go to register types with a Scheme
type APIs struct {
	file.Input
	// BoilerplatePath is the path to the boilerplate file
	BoilerplatePath string
	// Comments is a list of comments to add to the apis.go
	Comments []string
}

var deepCopy = strings.Join([]string{
	"//go:generate go run",
	"../../vendor/k8s.io/code-generator/cmd/deepcopy-gen/main.go",
	"-O zz_generated.deepcopy",
	"-i ./..."}, " ")

// GetInput implements input.Template
func (f *APIs) GetInput() (file.Input, error) {
	if f.Path == "" {
		f.Path = filepath.Join("pkg", "apis", "apis.go")
	}

	relPath, err := filepath.Rel(filepath.Dir(f.Path), f.BoilerplatePath)
	if err != nil {
		return file.Input{}, err
	}
	if len(f.Comments) == 0 {
		f.Comments = append(f.Comments,
			"// Generate deepcopy for apis", fmt.Sprintf("%s -h %s", deepCopy, filepath.ToSlash(relPath)))
	}
	f.TemplateBody = apisTemplate
	return f.Input, nil
}

const apisTemplate = `{{ .Boilerplate }}

{{ range $line := .Comments -}}
{{ $line }}
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
