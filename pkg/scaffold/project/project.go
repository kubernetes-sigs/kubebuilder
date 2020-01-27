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

package project

import (
	"fmt"

	internalconfig "sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/yaml"
)

var _ input.File = &Project{}

// Project scaffolds the PROJECT file with project metadata
type Project struct {
	// Path is the output file location - defaults to PROJECT
	Path string

	config.Config
}

// GetInput implements input.File
func (f *Project) GetInput() (input.Input, error) {
	if f.Path == "" {
		f.Path = internalconfig.DefaultPath
	}
	if f.Version == "" {
		f.Version = config.Version1
	}
	if f.Repo == "" {
		return input.Input{}, fmt.Errorf("must specify repository")
	}

	out, err := yaml.Marshal(f.Config)
	if err != nil {
		return input.Input{}, err
	}

	return input.Input{
		Path:           f.Path,
		TemplateBody:   string(out),
		Repo:           f.Repo,
		Version:        f.Version,
		Domain:         f.Domain,
		MultiGroup:     f.MultiGroup,
		IfExistsAction: input.Error,
	}, nil
}
