/*
Copyright 2021 The Kubernetes Authors.

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

package templates

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &Channel{}

// Channel scaffolds the file for the channel
type Channel struct {
	machinery.TemplateMixin

	ManifestVersion string
}

// SetTemplateDefaults implements file.Template
func (f *Channel) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("channels", "stable")
	}
	fmt.Println(f.Path)

	f.TemplateBody = channelTemplate

	f.IfExistsAction = machinery.SkipFile

	return nil
}

const channelTemplate = `# Versions for the stable channel
manifests:
- version: {{ .ManifestVersion }}
`
