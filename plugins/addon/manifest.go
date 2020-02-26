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
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

const exampleManifestVersion = "0.0.1"

const exampleManifestContents = `# Placeholder manifest - replace with the manifest for your addon
`

// ExampleManifest adds a model file for the manifest placeholder
func ExampleManifest(u *model.Universe) error {
	packageName := getPackageName(u)

	m := &file.File{
		Path:           filepath.Join("channels", "packages", packageName, exampleManifestVersion, "manifest.yaml"),
		Contents:       exampleManifestContents,
		IfExistsAction: file.Skip,
	}

	_, err := AddFile(u, m)

	return err
}

// getPackageName returns the (default) name of the declarative package
func getPackageName(u *model.Universe) string {
	return strings.ToLower(u.Resource.Kind)
}
