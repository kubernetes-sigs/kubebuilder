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

	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

const exampleChannel = `# Versions for the stable channel
manifests:
- version: 0.0.1
`

// ExampleChannel adds a model file for the channel
func ExampleChannel(u *model.Universe) error {
	m := &file.File{
		Path:           filepath.Join("channels", "stable"),
		Contents:       exampleChannel,
		IfExistsAction: file.Skip,
	}

	_, err := AddFile(u, m)
	return err
}
