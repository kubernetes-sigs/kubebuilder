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

package crd

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &Kustomization{}
var _ file.Inserter = &Kustomization{}

// Kustomization scaffolds the kustomization file in manager folder.
type Kustomization struct {
	file.TemplateMixin
	file.ResourceMixin
}

// SetTemplateDefaults implements file.Template
func (f *Kustomization) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "crd", "kustomization.yaml")
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.TemplateBody = fmt.Sprintf(kustomizationTemplate,
		file.NewMarkerFor(f.Path, resourceMarker),
		file.NewMarkerFor(f.Path, webhookPatchMarker),
		file.NewMarkerFor(f.Path, caInjectionPatchMarker),
	)

	return nil
}

const (
	resourceMarker         = "crdkustomizeresource"
	webhookPatchMarker     = "crdkustomizewebhookpatch"
	caInjectionPatchMarker = "crdkustomizecainjectionpatch"
)

// GetMarkers implements file.Inserter
func (f *Kustomization) GetMarkers() []file.Marker {
	return []file.Marker{
		file.NewMarkerFor(f.Path, resourceMarker),
		file.NewMarkerFor(f.Path, webhookPatchMarker),
		file.NewMarkerFor(f.Path, caInjectionPatchMarker),
	}
}

const (
	resourceCodeFragment = `- bases/%s_%s.yaml
`
	webhookPatchCodeFragment = `#- patches/webhook_in_%s.yaml
`
	caInjectionPatchCodeFragment = `#- patches/cainjection_in_%s.yaml
`
)

// GetCodeFragments implements file.Inserter
func (f *Kustomization) GetCodeFragments() file.CodeFragmentsMap {
	fragments := make(file.CodeFragmentsMap, 3)

	// Generate resource code fragments
	res := make([]string, 0)
	res = append(res, fmt.Sprintf(resourceCodeFragment, f.Resource.Domain, f.Resource.Plural))

	// Generate resource code fragments
	webhookPatch := make([]string, 0)
	webhookPatch = append(webhookPatch, fmt.Sprintf(webhookPatchCodeFragment, f.Resource.Plural))

	// Generate resource code fragments
	caInjectionPatch := make([]string, 0)
	caInjectionPatch = append(caInjectionPatch, fmt.Sprintf(caInjectionPatchCodeFragment, f.Resource.Plural))

	// Only store code fragments in the map if the slices are non-empty
	if len(res) != 0 {
		fragments[file.NewMarkerFor(f.Path, resourceMarker)] = res
	}
	if len(webhookPatch) != 0 {
		fragments[file.NewMarkerFor(f.Path, webhookPatchMarker)] = webhookPatch
	}
	if len(caInjectionPatch) != 0 {
		fragments[file.NewMarkerFor(f.Path, caInjectionPatchMarker)] = caInjectionPatch
	}

	return fragments
}

var kustomizationTemplate = `# This kustomization.yaml is not intended to be run by itself,
# since it depends on service name and namespace that are out of this kustomize package.
# It should be run by config/default
resources:
%s

patchesStrategicMerge:
# [WEBHOOK] To enable webhook, uncomment all the sections with [WEBHOOK] prefix.
# patches here are for enabling the conversion webhook for each CRD
%s

# [CERTMANAGER] To enable webhook, uncomment all the sections with [CERTMANAGER] prefix.
# patches here are for enabling the CA injection for each CRD
%s

# the following config is for teaching kustomize how to do kustomization for CRDs.
configurations:
- kustomizeconfig.yaml
`
