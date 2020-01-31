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
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v2/internal"
)

const (
	kustomizeResourceScaffoldMarker         = "# +kubebuilder:scaffold:crdkustomizeresource"
	kustomizeWebhookPatchScaffoldMarker     = "# +kubebuilder:scaffold:crdkustomizewebhookpatch"
	kustomizeCAInjectionPatchScaffoldMarker = "# +kubebuilder:scaffold:crdkustomizecainjectionpatch"
)

var _ file.Template = &Kustomization{}

// Kustomization scaffolds the kustomization file in manager folder.
type Kustomization struct {
	file.Input

	// Resource is the Resource to make the EnableWebhookPatch for
	Resource *resource.Resource
}

// GetInput implements input.Template
func (f *Kustomization) GetInput() (file.Input, error) {
	if f.Path == "" {
		f.Path = filepath.Join("config", "crd", "kustomization.yaml")
	}
	f.TemplateBody = kustomizationTemplate
	return f.Input, nil
}

func (f *Kustomization) Update() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "crd", "kustomization.yaml")
	}

	// TODO(directxman12): not technically valid if something changes from the default
	// (we'd need to parse the markers)

	kustomizeResourceCodeFragment := fmt.Sprintf("- bases/%s_%s.yaml\n", f.Resource.Domain, f.Resource.Plural)
	kustomizeWebhookPatchCodeFragment := fmt.Sprintf("#- patches/webhook_in_%s.yaml\n", f.Resource.Plural)
	kustomizeCAInjectionPatchCodeFragment := fmt.Sprintf("#- patches/cainjection_in_%s.yaml\n", f.Resource.Plural)

	return internal.InsertStringsInFile(f.Path,
		map[string][]string{
			kustomizeResourceScaffoldMarker:         {kustomizeResourceCodeFragment},
			kustomizeWebhookPatchScaffoldMarker:     {kustomizeWebhookPatchCodeFragment},
			kustomizeCAInjectionPatchScaffoldMarker: {kustomizeCAInjectionPatchCodeFragment},
		})
}

var kustomizationTemplate = fmt.Sprintf(`# This kustomization.yaml is not intended to be run by itself,
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
`, kustomizeResourceScaffoldMarker, kustomizeWebhookPatchScaffoldMarker, kustomizeCAInjectionPatchScaffoldMarker)
