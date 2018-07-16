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
	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
)

var _ input.File = &GitIgnore{}

// GitIgnore scaffolds the .gitignore file
type GitIgnore struct {
	input.Input
}

// GetInput implements input.File
func (c *GitIgnore) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = ".gitignore"
	}
	c.TemplateBody = gitignoreTemplate
	return c.Input, nil
}

var gitignoreTemplate = `
# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, build with ` + "`go test -c`" + `
*.test

# Output of the go coverage tool, specifically when used with LiteIDE
*.out

# Kubernetes Generated files - skip generated files, except for vendored files

zz_generated.*
!vendor/**/zz_generated.*

# editor and IDE paraphernalia
.idea
*.swp
*.swo
*~
`
