/*
Copyright 2024 The Kubernetes Authors.

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

package kdefault

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

const (
	caNamespace = "crdkustomizecainjectionns"
	caName      = "crdkustomizecainjectionname"
)

// KustomizationCAConversionUpdater appends CA injection targets for CRDs with --conversion
type KustomizationCAConversionUpdater struct {
	machinery.TemplateMixin
	machinery.ResourceMixin
}

// SetTemplateDefaults defines the file path and behavior for existing files
func (f *KustomizationCAConversionUpdater) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "default", "kustomization.yaml")
	}
	f.IfExistsAction = machinery.SkipFile // Only append to the existing file, don’t overwrite it
	return nil
}

// GetMarkers provides the markers where the CA injection targets will be appended
func (f *KustomizationCAConversionUpdater) GetMarkers() []machinery.Marker {
	return []machinery.Marker{
		machinery.NewMarkerFor(f.Path, caNamespace),
		machinery.NewMarkerFor(f.Path, caName),
	}
}

// GetCodeFragments appends CA injection targets for the CRD with --conversion as comments
func (f *KustomizationCAConversionUpdater) GetCodeFragments() machinery.CodeFragmentsMap {
	fragments := make(machinery.CodeFragmentsMap)

	// Obtain the formatted CRD name as Plural.Group.Domain (e.g., cronjobs.batch.tutorial.kubebuilder.io)
	crdName := fmt.Sprintf("%s.%s", f.Resource.Plural, f.Resource.QualifiedGroup())

	if !f.Resource.Webhooks.IsEmpty() && f.Resource.Webhooks.Conversion {
		// Commented CA injection configuration for the namespace part
		caInjectionNamespace := fmt.Sprintf(`#     - select:
#         kind: CustomResourceDefinition
#         name: %s
#       fieldPaths:
#         - .metadata.annotations.[cert-manager.io/inject-ca-from]
#       options:
#         delimiter: '/'
#         index: 0
#         create: true
`, crdName)

		// Commented CA injection configuration for the name part
		caInjectionName := fmt.Sprintf(`#     - select:
#         kind: CustomResourceDefinition
#         name: %s
#       fieldPaths:
#         - .metadata.annotations.[cert-manager.io/inject-ca-from]
#       options:
#         delimiter: '/'
#         index: 1
#         create: true
`, crdName)

		// Append to the correct markers to prevent duplication
		namespaceMarker := machinery.NewMarkerFor(f.Path, caNamespace)
		certificateMarker := machinery.NewMarkerFor(f.Path, caName)

		// Check if the fragments already exist before adding them
		if _, exists := fragments[namespaceMarker]; !exists {
			fragments[namespaceMarker] = []string{caInjectionNamespace}
		}
		if _, exists := fragments[certificateMarker]; !exists {
			fragments[certificateMarker] = []string{caInjectionName}
		}
	}

	return fragments
}
