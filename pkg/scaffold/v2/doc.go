/*
Copyright 2019 The Kubernetes Authors.

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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

var _ input.File = &Doc{}

// Doc scaffolds the api/doc.go directory so that deep-copy gen can discover all
// the api version pkgs. Once we fix deepcopy-gen, we can remove scaffolding for
// this.
type Doc struct {
	input.Input

	// Comments are additional lines to write to the doc.go file
	Comments []string
}

// GetInput implements input.File
func (a *Doc) GetInput() (input.Input, error) {
	if a.Path == "" {
		a.Path = filepath.Join("api", "doc.go")
	}
	a.TemplateBody = docGoTemplate
	return a.Input, nil
}

// Validate validates the values
func (a *Doc) Validate() error {
	return nil
}

var docGoTemplate = `{{ .Boilerplate }}

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package,register
package api
`
