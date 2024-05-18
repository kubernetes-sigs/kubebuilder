/*
Copyright 2022 The Kubernetes Authors.

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

package samples

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var (
	_ machinery.Template = &Kustomization{}
	_ machinery.Inserter = &Kustomization{}
)

// Kustomization scaffolds a kustomization.yaml for the manifests overlay folder.
type Kustomization struct {
	machinery.TemplateMixin
	machinery.ResourceMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *Kustomization) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "samples", "kustomization.yaml")
	}
	f.TemplateBody = fmt.Sprintf(kustomizationTemplate, machinery.NewMarkerFor(f.Path, samplesMarker))

	return nil
}

const (
	samplesMarker = "manifestskustomizesamples"
)

// GetMarkers implements file.Inserter
func (f *Kustomization) GetMarkers() []machinery.Marker {
	return []machinery.Marker{machinery.NewMarkerFor(f.Path, samplesMarker)}
}

const samplesCodeFragment = `- %s
`

// makeCRFileName returns a Custom Resource example file name in the same format
// as kubebuilder's CreateAPI plugin for a gvk.
func (f Kustomization) makeCRFileName() string {
	if f.Resource.Group != "" {
		return f.Resource.Replacer().Replace("%[group]_%[version]_%[kind].yaml")
	}
	return f.Resource.Replacer().Replace("%[version]_%[kind].yaml")

}

// GetCodeFragments implements file.Inserter
func (f *Kustomization) GetCodeFragments() machinery.CodeFragmentsMap {
	return machinery.CodeFragmentsMap{
		machinery.NewMarkerFor(f.Path, samplesMarker): []string{fmt.Sprintf(samplesCodeFragment, f.makeCRFileName())},
	}
}

const kustomizationTemplate = `## Append samples of your project ##
resources:
%s
`
