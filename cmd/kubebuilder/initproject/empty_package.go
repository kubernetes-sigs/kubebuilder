/*
Copyright 2017 The Kubernetes Authors.

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

package initproject

import (
	"path/filepath"
)

// createPackage creates a new go package with a doc.go file
func createPackage(boilerplate, path string) {
	pkg := filepath.Base(path)
	execute(
		filepath.Join(path, "doc.go"),
		"pkg-template", packageDocTemplate,
		packageDocTemplateArguments{
			boilerplate,
			pkg,
		})
}

type packageDocTemplateArguments struct {
	BoilerPlate string
	Package     string
}

var packageDocTemplate = `
{{.BoilerPlate}}


package {{.Package}}

`

func createBoilerplate() {
	execute(
		filepath.Join("hack", "boilerplate.go.txt"),
		"boilerplate-template", boilerplateTemplate, nil)
}

var boilerplateTemplate = ``
