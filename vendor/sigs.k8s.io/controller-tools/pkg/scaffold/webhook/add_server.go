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

package webhook

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
	"sigs.k8s.io/controller-tools/pkg/scaffold/resource"
)

var _ input.File = &AddServer{}

// AddServer scaffolds adds a new webhook server.
type AddServer struct {
	input.Input

	// Resource is a resource in the API group
	Resource *resource.Resource

	Config
}

// GetInput implements input.File
func (a *AddServer) GetInput() (input.Input, error) {
	if a.Path == "" {
		a.Path = filepath.Join("pkg", "webhook", fmt.Sprintf("add_%s_server.go", a.Server))
	}
	a.TemplateBody = addServerTemplate
	return a.Input, nil
}

var addServerTemplate = `{{ .Boilerplate }}

package webhook

import (
	server "{{ .Repo }}/pkg/webhook/{{ .Server }}_server"
)

func init() {
	// AddToManagerFuncs is a list of functions to create webhook servers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, server.Add)
}
`
