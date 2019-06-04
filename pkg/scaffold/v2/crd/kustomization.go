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

package v2

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/markbates/inflect"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v2/internal"
)

const (
	kustomizeResourceScaffoldMarker = "# +kubebuilder:scaffold:kustomizeresource"
	kustomizePatchScaffoldMarker    = "# +kubebuilder:scaffold:kustomizepatch"
)

var _ input.File = &Kustomization{}

// Kustomization scaffolds the kustomization file in manager folder.
type Kustomization struct {
	input.Input

	// Resource is the Resource to make the EnableWebhookPatch for
	Resource *resource.Resource
}

// GetInput implements input.File
func (c *Kustomization) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = filepath.Join("config", "crd", "kustomization.yaml")
	}
	c.TemplateBody = kustomizationTemplate
	c.Input.IfExistsAction = input.Error
	return c.Input, nil
}

func (c *Kustomization) Update() error {
	if c.Path == "" {
		c.Path = filepath.Join("config", "crd", "kustomization.yaml")
	}

	// TODO(directxman12): not technically valid if something changes from the default
	// (we'd need to parse the markers)
	rs := inflect.NewDefaultRuleset()
	plural := rs.Pluralize(strings.ToLower(c.Resource.Kind))

	kustomizeResourceCodeFragment := fmt.Sprintf("- bases/%s.%s_%s.yaml\n", c.Resource.Group, c.Domain, plural)
	kustomizePatchCodeFragment := fmt.Sprintf("#- patches/webhook_in_%s.yaml\n", plural)

	return internal.InsertStringsInFile(c.Path,
		map[string][]string{
			kustomizeResourceScaffoldMarker: []string{kustomizeResourceCodeFragment},
			kustomizePatchScaffoldMarker:    []string{kustomizePatchCodeFragment},
		})
}

var kustomizationTemplate = fmt.Sprintf(`# This kustomization.yaml is not intended to be run by itself,
# since it depends on service name and namespace that are out of this kustomize package.
# It should be run by config/default
resources:
%s

patches:
# patches here are for enabling the conversion webhook for each CRD
%s

# the following config is for teaching kustomize how to do kustomization for CRDs.
configurations:
- kustomizeconfig.yaml
`, kustomizeResourceScaffoldMarker, kustomizePatchScaffoldMarker)
