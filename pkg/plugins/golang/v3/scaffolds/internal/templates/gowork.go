/*
Copyright 2020 The Kubernetes Authors.

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

const (
	apiModulesMarker = "apis"
)

const defaultWorkPath = "go.work"

var _ machinery.Template = &GoWork{}

// GoWork scaffolds a file that defines the project dependencies
type GoWork struct {
	machinery.TemplateMixin
	machinery.RepositoryMixin
}

// SetTemplateDefaults implements file.Template
func (f *GoWork) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = defaultWorkPath
	}

	f.TemplateBody = fmt.Sprintf(goWorkTemplate,
		machinery.NewMarkerFor(f.Path, apiModulesMarker),
	)

	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

const goWorkTemplate = `
go 1.18

%s

use .
`

// ########################################################
// GoWorkUpdater
// ########################################################

const (
	apisUseCodeFragment = `use ./%s
`
)

var _ machinery.Inserter = &GoWorkUpdater{}

type GoWorkUpdater struct {
	machinery.RepositoryMixin
	machinery.TemplateMixin
	machinery.ResourceMixin

	WireUses bool
}

// SetTemplateDefaults implements file.Template
func (f *GoWorkUpdater) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = defaultWorkPath
	}

	f.TemplateBody = fmt.Sprintf(goWorkTemplate,
		machinery.NewMarkerFor(f.Path, apiModulesMarker),
	)

	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

// GetMarkers implements file.Inserter
func (f *GoWorkUpdater) GetMarkers() []machinery.Marker {
	return []machinery.Marker{
		machinery.NewMarkerFor(f.Path, apiModulesMarker),
	}
}

// GetCodeFragments implements file.Inserter
func (f *GoWorkUpdater) GetCodeFragments() machinery.CodeFragmentsMap {
	fragments := make(machinery.CodeFragmentsMap, 1)

	// If resource is not being provided we are creating the file, not updating it
	if f.Resource == nil {
		return fragments
	}

	// Generate require code fragments
	uses := make([]string, 0)

	if f.WireUses {
		moduleRelativePath, err := filepath.Rel(f.Repo, f.Resource.Path)
		if err != nil {
			return fragments
		}
		dep := fmt.Sprintf(apisUseCodeFragment, moduleRelativePath)
		uses = append(uses, dep)
	}

	// Only store code fragments in the map if the slices are non-empty
	if len(uses) != 0 {
		fragments[machinery.NewMarkerFor(f.Path, apiModulesMarker)] = uses
	}

	return fragments
}
