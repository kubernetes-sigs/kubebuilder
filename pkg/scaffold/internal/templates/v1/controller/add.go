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

package controller

import (
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &AddController{}

// AddController scaffolds adds a new Controller.
type AddController struct {
	file.Input
	file.RepositoryMixin
	file.BoilerplateMixin
	file.ResourceMixin
}

// GetInput implements input.Template
func (f *AddController) GetInput() (file.Input, error) {
	if f.Path == "" {
		f.Path = filepath.Join("pkg", "controller", fmt.Sprintf(
			"add_%s.go", strings.ToLower(f.Resource.Kind)))
	}
	f.TemplateBody = addControllerTemplate
	return f.Input, nil
}

const addControllerTemplate = `{{ .Boilerplate }}

package controller

import (
	"{{ .Repo }}/pkg/controller/{{ lower .Resource.Kind }}"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, {{ lower .Resource.Kind }}.Add)
}
`
