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
	"os"
	"path/filepath"

	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
)

var _ input.File = &Kustomize{}

// Kustomize scaffolds the Kustomization file.
type Kustomize struct {
	input.Input

	// Prefix to use for name prefix customization
	Prefix string
}

// GetInput implements input.File
func (c *Kustomize) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = filepath.Join("config", "default", "kustomization.yaml")
	}
	if c.Prefix == "" {
		// use directory name as prefix
		dir, err := os.Getwd()
		if err != nil {
			return input.Input{}, err
		}
		c.Prefix = filepath.Base(dir)
	}
	c.TemplateBody = kustomizeTemplate
	c.Input.IfExistsAction = input.Error
	return c.Input, nil
}

var kustomizeTemplate = `# Adds namespace to all resources.
namespace: {{.Prefix}}-system

# Value of this field is prepended to the
# names of all resources, e.g. a deployment named
# "wordpress" becomes "alices-wordpress".
# Note that it should also match with the prefix (text before '-') of the namespace
# field above.
namePrefix: {{.Prefix}}-

# Labels to add to all resources and selectors.
#commonLabels:
#  someName: someValue

# Each entry in this list must resolve to an existing
# resource definition in YAML.  These are the resource
# files that kustomize reads, modifies and emits as a
# YAML string, with resources separated by document
# markers ("---").
resources:
- ../rbac/*.yaml
- ../manager/*.yaml

patches:
- manager_image_patch.yaml
`
