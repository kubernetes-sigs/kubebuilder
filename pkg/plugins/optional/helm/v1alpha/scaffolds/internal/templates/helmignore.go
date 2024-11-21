/*
Copyright 2024 The Kubernetes Authors.

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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &HelmIgnore{}

// HelmIgnore scaffolds a file that defines the .helmignore for Helm packaging
type HelmIgnore struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *HelmIgnore) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("dist", "chart", ".helmignore")
	}

	f.TemplateBody = helmIgnoreTemplate

	f.IfExistsAction = machinery.SkipFile

	return nil
}

const helmIgnoreTemplate = `# Patterns to ignore when building Helm packages.
# Operating system files
.DS_Store

# Version control directories
.git/
.gitignore
.bzr/
.hg/
.hgignore
.svn/

# Backup and temporary files
*.swp
*.tmp
*.bak
*.orig
*~

# IDE and editor-related files
.idea/
.vscode/

# Helm chart artifacts
dist/chart/*.tgz
`
