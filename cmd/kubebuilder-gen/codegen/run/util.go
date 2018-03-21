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

package run

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"k8s.io/gengo/generator"
	"k8s.io/gengo/types"
)

// generatorToPackage creates a new package from a generator and package name
func generatorToPackage(pkg string, gen generator.Generator) generator.Package {
	name := strings.Split(filepath.Base(pkg), ".")[0]
	return &generator.DefaultPackage{
		PackageName: name,
		PackagePath: pkg,
		HeaderText:  generatedGoHeader(),
		GeneratorFunc: func(c *generator.Context) (generators []generator.Generator) {
			return []generator.Generator{gen}
		},
		FilterFunc: func(c *generator.Context, t *types.Type) bool {
			// Generators only see Types in the same package as the generator
			return t.Name.Package == pkg
		},
	}
}

// generatedGoHeader returns the header to preprend to generated go files
func generatedGoHeader() []byte {
	cr, err := ioutil.ReadFile(filepath.Join("hack", "boilerplate.go.txt"))
	if err != nil {
		return []byte{}
	}
	return cr
}

// packages wraps a collection of generator.Packages
type packages struct {
	value generator.Packages
}

// add creates a new generator.Package from gen and adds it to the collection
func (g *packages) add(pkg string, gen generator.Generator) {
	g.value = append(g.value, generatorToPackage(pkg, gen))
}
