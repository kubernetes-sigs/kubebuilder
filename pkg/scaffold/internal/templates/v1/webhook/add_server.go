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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v2/pkg/model/file"
)

var _ file.Template = &AddServer{}

// AddServer scaffolds adds a new webhook server.
type AddServer struct {
	file.TemplateMixin
	file.RepositoryMixin
	file.BoilerplateMixin

	Config
}

// SetTemplateDefaults implements input.Template
func (f *AddServer) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("pkg", "webhook", "add_"+f.Server+"_server.go")
	}

	f.TemplateBody = addServerTemplate

	return nil
}

const addServerTemplate = `{{ .Boilerplate }}

package webhook

import (
	server "{{ .Repo }}/pkg/webhook/{{ .Server }}_server"
)

func init() {
	// AddToManagerFuncs is a list of functions to create webhook servers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, server.Add)
}
`
