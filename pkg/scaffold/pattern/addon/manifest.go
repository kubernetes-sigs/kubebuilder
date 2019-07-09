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

package addon

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

const exampleManifestVersion = "0.0.1"

var _ input.File = &ExampleManifest{}

// ExampleManifest creates an example manifest file in the channels folder.
type ExampleManifest struct {
	input.Input

	PackageName string
}

// GetInput implements input.File
func (c *ExampleManifest) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = filepath.Join("channels", "packages", c.PackageName, exampleManifestVersion, "manifest.yaml")
	}
	c.TemplateBody = exampleManifestTemplate
	c.Input.IfExistsAction = input.Error
	return c.Input, nil
}

var exampleManifestTemplate = `# Placeholder manifest - replace with the manifest for your addon
`
