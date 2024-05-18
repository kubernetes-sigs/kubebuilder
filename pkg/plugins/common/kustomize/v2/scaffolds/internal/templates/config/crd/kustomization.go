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

package crd

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var (
	_ machinery.Template = &Kustomization{}
	_ machinery.Inserter = &Kustomization{}
)

// Kustomization scaffolds a file that defines the kustomization scheme for the crd folder
type Kustomization struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.ResourceMixin
}

// SetTemplateDefaults implements file.Template
func (f *Kustomization) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "crd", "kustomization.yaml")
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.TemplateBody = fmt.Sprintf(kustomizationTemplate,
		machinery.NewMarkerFor(f.Path, resourceMarker),
		machinery.NewMarkerFor(f.Path, webhookPatchMarker),
		machinery.NewMarkerFor(f.Path, caInjectionPatchMarker),
	)

	return nil
}

//nolint:gosec to ignore false complain G101: Potential hardcoded credentials (gosec)
const (
	resourceMarker         = "crdkustomizeresource"
	webhookPatchMarker     = "crdkustomizewebhookpatch"
	caInjectionPatchMarker = "crdkustomizecainjectionpatch"
)

// GetMarkers implements file.Inserter
func (f *Kustomization) GetMarkers() []machinery.Marker {
	return []machinery.Marker{
		machinery.NewMarkerFor(f.Path, resourceMarker),
		machinery.NewMarkerFor(f.Path, webhookPatchMarker),
		machinery.NewMarkerFor(f.Path, caInjectionPatchMarker),
	}
}

const (
	resourceCodeFragment = `- bases/%s_%s.yaml
`
	webhookPatchCodeFragment = `#- path: patches/webhook_in_%s.yaml
`
	caInjectionPatchCodeFragment = `#- path: patches/cainjection_in_%s.yaml
`
)

// GetCodeFragments implements file.Inserter
func (f *Kustomization) GetCodeFragments() machinery.CodeFragmentsMap {
	fragments := make(machinery.CodeFragmentsMap, 3)

	// Generate resource code fragments
	res := make([]string, 0)
	res = append(res, fmt.Sprintf(resourceCodeFragment, f.Resource.QualifiedGroup(), f.Resource.Plural))

	suffix := f.Resource.Plural
	if f.MultiGroup && f.Resource.Group != "" {
		suffix = f.Resource.Group + "_" + f.Resource.Plural
	}

	if !f.Resource.Webhooks.IsEmpty() {
		webhookPatch := fmt.Sprintf(webhookPatchCodeFragment, suffix)
		fragments[machinery.NewMarkerFor(f.Path, webhookPatchMarker)] = []string{webhookPatch}
	}

	// Generate resource code fragments
	caInjectionPatch := make([]string, 0)
	caInjectionPatch = append(caInjectionPatch, fmt.Sprintf(caInjectionPatchCodeFragment, suffix))

	// Only store code fragments in the map if the slices are non-empty
	if len(res) != 0 {
		fragments[machinery.NewMarkerFor(f.Path, resourceMarker)] = res
	}

	if len(caInjectionPatch) != 0 {
		fragments[machinery.NewMarkerFor(f.Path, caInjectionPatchMarker)] = caInjectionPatch
	}

	return fragments
}

var kustomizationTemplate = `# This kustomization.yaml is not intended to be run by itself,
# since it depends on service name and namespace that are out of this kustomize package.
# It should be run by config/default
resources:
%s

patches:
# [WEBHOOK] To enable webhook, uncomment all the sections with [WEBHOOK] prefix.
# patches here are for enabling the conversion webhook for each CRD
%s

# [CERTMANAGER] To enable cert-manager, uncomment all the sections with [CERTMANAGER] prefix.
# patches here are for enabling the CA injection for each CRD
%s

# [WEBHOOK] To enable webhook, uncomment the following section
# the following config is for teaching kustomize how to do kustomization for CRDs.

#configurations:
#- kustomizeconfig.yaml
`
