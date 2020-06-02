/*
Copyright 2018 The Kubernetes Authors.

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
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var (
	defaultKustomizePath = filepath.Join("config", "default", "kustomization.yaml")
)

var _ file.Template = &Kustomize{}

// Kustomize scaffolds the Kustomization file for the default overlay
type Kustomize struct {
	file.TemplateMixin

	// Prefix to use for name prefix customization
	Prefix string
}

// SetTemplateDefaults implements input.Template
func (f *Kustomize) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = defaultKustomizePath
	}

	f.TemplateBody = fmt.Sprintf(kustomizeTemplate,
		file.NewMarkerFor(f.Path, basesMarker),
	)

	f.IfExistsAction = file.Error

	if f.Prefix == "" {
		// use directory name as prefix
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		f.Prefix = strings.ToLower(filepath.Base(dir))
	}

	return nil
}

var _ file.Inserter = &KustomizeUpdater{}

type KustomizeUpdater struct { //nolint:maligned
	// Flags to indicate which parts need to be included when updating the file
	HasResource, HasController bool
}

// GetPath implements Builder
func (*KustomizeUpdater) GetPath() string {
	return defaultKustomizePath
}

// GetIfExistsAction implements Builder
func (*KustomizeUpdater) GetIfExistsAction() file.IfExistsAction {
	return file.Overwrite
}

const (
	basesMarker = "bases"
)

// GetMarkers implements file.Inserter
func (f *KustomizeUpdater) GetMarkers() []file.Marker {
	return []file.Marker{
		file.NewMarkerFor(defaultKustomizePath, basesMarker),
	}
}

// GetCodeFragments implements file.Inserter
func (f *KustomizeUpdater) GetCodeFragments() file.CodeFragmentsMap {
	var fragment file.CodeFragments

	if f.HasResource {
		fragment = append(fragment, "- ../crd\n")
	}

	if f.HasController {
		fragment = append(fragment, "- ../rbac\n")
	}

	fragments := make(file.CodeFragmentsMap, 1)
	if len(fragment) > 0 {
		fragments[file.NewMarkerFor(defaultKustomizePath, basesMarker)] = fragment
	}

	return fragments
}

const kustomizeTemplate = `# Adds namespace to all resources.
namespace: {{ .Prefix }}-system

# Value of this field is prepended to the
# names of all resources, e.g. a deployment named
# "wordpress" becomes "alices-wordpress".
# Note that it should also match with the prefix (text before '-') of the namespace
# field above.
namePrefix: {{ .Prefix }}-

# Labels to add to all resources and selectors.
#commonLabels:
#  someName: someValue

bases:
- ../manager
%s
# [WEBHOOK] To enable webhook, uncomment all the sections with [WEBHOOK] prefix including the one in
# crd/kustomization.yaml
#- ../webhook
# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER'. 'WEBHOOK' components are required.
#- ../certmanager
# [PROMETHEUS] To enable prometheus monitor, uncomment all sections with 'PROMETHEUS'.
#- ../prometheus

patchesStrategicMerge:
  # Protect the /metrics endpoint by putting it behind auth.
  # If you want your controller-manager to expose the /metrics
  # endpoint w/o any authn/z, please comment the following line.
- manager_auth_proxy_patch.yaml

# [WEBHOOK] To enable webhook, uncomment all the sections with [WEBHOOK] prefix including the one in
# crd/kustomization.yaml
#- manager_webhook_patch.yaml

# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER'.
# Uncomment 'CERTMANAGER' sections in crd/kustomization.yaml to enable the CA injection in the admission webhooks.
# 'CERTMANAGER' needs to be enabled to use ca injection
#- webhookcainjection_patch.yaml

# the following config is for teaching kustomize how to do var substitution
vars:
# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER' prefix.
#- name: CERTIFICATE_NAMESPACE # namespace of the certificate CR
#  objref:
#    kind: Certificate
#    group: cert-manager.io
#    version: v1alpha2
#    name: serving-cert # this name should match the one in certificate.yaml
#  fieldref:
#    fieldpath: metadata.namespace
#- name: CERTIFICATE_NAME
#  objref:
#    kind: Certificate
#    group: cert-manager.io
#    version: v1alpha2
#    name: serving-cert # this name should match the one in certificate.yaml
#- name: SERVICE_NAMESPACE # namespace of the service
#  objref:
#    kind: Service
#    version: v1
#    name: webhook-service
#  fieldref:
#    fieldpath: metadata.namespace
#- name: SERVICE_NAME
#  objref:
#    kind: Service
#    version: v1
#    name: webhook-service
`
