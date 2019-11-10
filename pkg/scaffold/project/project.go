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

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/yaml"
)

// constants for scaffolding version
const (
	Version1 = "1"
	Version2 = "2"
)

var _ input.File = &Project{}

// Project scaffolds the PROJECT file with project metadata
type Project struct {
	// Path is the output file location - defaults to PROJECT
	Path string
	input.ProjectFile
}

// GetInput implements input.File
func (c *Project) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = "PROJECT"
	}
	if c.Version == "" {
		c.Version = Version1
	}

	if c.ProjectType == "" {
		c.ProjectType = Go
	}

	if c.Repo == "" {
		return input.Input{}, fmt.Errorf("must specify repository")
	}

	out, err := yaml.Marshal(c.ProjectFile)
	if err != nil {
		return input.Input{}, err
	}

	return input.Input{
		Path:           c.Path,
		TemplateBody:   string(out),
		Repo:           c.Repo,
		Version:        c.Version,
		ProjectType:    c.ProjectType,
		Domain:         c.Domain,
		IfExistsAction: input.Error,
	}, nil
}
