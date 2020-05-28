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

package v3

import (
	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &DockerignoreFile{}

// Dockerfile scaffolds a DockerignoreFile for building a main
type DockerignoreFile struct {
	file.TemplateMixin
}

// SetTemplateDefaults implements input.Template
func (f *DockerignoreFile) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = ".dockerignore"
	}

	f.TemplateBody = dockerignorefileTemplate

	return nil
}

const dockerignorefileTemplate = `# More info: https://docs.docker.com/engine/reference/builder/#dockerignore-file
# Ignore all files which are not go type
!**/*.go
!**/*.mod
!**/*.sum
`
