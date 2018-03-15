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

package resourcegen

import (
	"io"
	"text/template"

	"github.com/kubernetes-sigs/kubebuilder/cmd/internal/codegen"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/gengo/generator"
)

type unversionedGenerator struct {
	generator.DefaultGen
	apigroup *codegen.APIGroup
}

var _ generator.Generator = &unversionedGenerator{}

func (d *unversionedGenerator) Imports(c *generator.Context) []string {
	imports := sets.NewString()
	return imports.List()
}

func (d *unversionedGenerator) Finalize(context *generator.Context, w io.Writer) error {
	temp := template.Must(template.New("unversioned-wiring-template").Parse(UnversionedAPITemplate))
	err := temp.Execute(w, d.apigroup)
	if err != nil {
		return err
	}
	return err
}

var UnversionedAPITemplate = `
const (
	GroupName = "{{.Group}}.{{.Domain}}"
)
`
