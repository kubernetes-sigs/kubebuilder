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

package configgen

import (
	"fmt"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &ConfigGen{}

// ConfigGen scaffolds a KubebuilderConfigGen machinery.
type ConfigGen struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
	machinery.DomainMixin
	machinery.MultiGroupMixin
	machinery.ComponentConfigMixin

	// Manager image tag.
	Image string
}

// SetTemplateDefaults implements machinery.Template
func (f *ConfigGen) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = "kubebuilderconfiggen.yaml"
	}

	if f.Image == "" {
		f.Image = fmt.Sprintf("%s/%s:v0.1.0", f.Domain, f.ProjectName)
	}

	f.TemplateBody = fmt.Sprintf(configGenTransformerTemplate,
		machinery.NewMarkerFor(f.Path, crdName),
	)

	f.IfExistsAction = machinery.Error

	return nil
}

var _ machinery.Inserter = &ConfigGenUpdater{}

// ConfigGenUpdater updates this scaffold with a resource.
type ConfigGenUpdater struct { //nolint:golint
	machinery.ResourceMixin

	path string
}

// GetPath implements machinery.Builder
func (f *ConfigGenUpdater) GetPath() string {
	if f.path == "" {
		f.path = "kubebuilderconfiggen.yaml"
	}
	return f.path
}

// GetIfExistsAction implements machinery.Builder
func (f *ConfigGenUpdater) GetIfExistsAction() machinery.IfExistsAction {
	return machinery.Error
}

const (
	crdName = "crdName"
)

const (
	whCRDConvCodeFragment = `  #    %s.%s: false
`
)

// GetMarkers implements machinery.Inserter
func (f *ConfigGenUpdater) GetMarkers() []machinery.Marker {
	return []machinery.Marker{
		machinery.NewMarkerFor(f.GetPath(), crdName),
	}
}

// GetCodeFragments implements machinery.Inserter
func (f *ConfigGenUpdater) GetCodeFragments() machinery.CodeFragmentsMap {
	fragments := make(machinery.CodeFragmentsMap)

	// Only update if resource is set, which is not the case at init.
	if f.Resource != nil {
		// TODO(estroz): read pluralized name from type marker.
		val := fmt.Sprintf(whCRDConvCodeFragment, f.Resource.Plural, f.Resource.QualifiedGroup())
		fragments[machinery.NewMarkerFor(f.GetPath(), crdName)] = []string{val}
	}

	return fragments
}

const configGenTransformerTemplate = `apiVersion: kubebuilder.sigs.k8s.io/v1alpha1
kind: KubebuilderConfigGen

metadata:
  # name of the project.  used in various resource names.
  # required
  name: {{ .ProjectName }}-controller-manager

  # namespace for the project
  # optional -- defaults to "${metadata.name}-system"
  namespace: {{ .ProjectName }}-system

spec:
  # configure how CRDs are generated
  crds:
    # path to go module source directory provided to controller-gen libraries
    # optional -- defaults to '.'
    sourceDirectory: ./api{{ if .MultiGroup}}s{{ end }}/...

  # configure how the controller-manager is generated
  controllerManager:
    # image to run
    image: {{ .Image }}

    # if set, use component config for the controller-manager
    # optional -- defaults to "enable: false"
    componentConfig:
      # use component config
      enable: {{ if .ComponentConfig }}true{{ else }}false{{end}}

      # path to component config to put into a ConfigMap
      configFilepath: controller_manager_config.yaml

    # configure how metrics are exposed
    # uncomment to expose metrics configuration
    # optional -- defaults to not generating metrics configuration
    #metrics:
    #  # disable the auth proxy required for scraping metrics
    #  disable: false
    #
    #  # generate prometheus ServiceMonitor resource
    #  enableServiceMonitor: true

  # configure how webhooks are generated
  # uncomment to expose webhook configuration
  # optional -- defaults to not generating webhook configuration
  #webhooks:
  #  # enable will cause webhook config to be generated
  #  enable: true
  #
  #  # configures crds which use conversion webhooks
  #  enableConversion:
  #    # key is the name of the CRD. For example:
  #    # bars.example.my.domain: false
  #    %[1]s
  #
  #  # configures where to get the certificate used for webhooks
  #  # discriminated union
  #  certificateSource:
  #    # type of certificate source
  #    # one of ["certManager", "dev", "manual"] -- defaults to "manual"
  #    # certManager: certmanager is used to manage certificates -- requires CertManager to be installed
  #    # dev: certificate is generated and wired into resources
  #    # manual: no certificate is generated or wired into resources
  #    type: "dev"
  #
  #    # options for a dev certificate -- requires "dev" as the type
  #    devCertificate:
  #      duration: 1h
`
