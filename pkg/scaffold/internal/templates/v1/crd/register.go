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

package crd

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &Register{}

// Register scaffolds the pkg/apis/group/version/register.go file
type Register struct {
	file.TemplateMixin
	file.RepositoryMixin
	file.BoilerplateMixin
	file.ResourceMixin
}

// SetTemplateDefaults implements input.Template
func (f *Register) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("pkg", "apis", "%[group-package-name]", "%[version]", "register.go")
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.TemplateBody = registerTemplate

	return nil
}

//nolint:lll
const registerTemplate = `{{ .Boilerplate }}

// NOTE: Boilerplate only. Ignore this file.

// Package {{ .Resource.Version }} contains API Schema definitions for the {{ .Resource.Group }} {{ .Resource.Version }} API group
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package,register
// +k8s:conversion-gen={{ .Repo }}/pkg/apis/{{ .Resource.GroupPackageName }}
// +k8s:defaulter-gen=TypeMeta
// +groupName={{ .Resource.Domain }}
package {{ .Resource.Version }}

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "{{ .Resource.Domain }}", Version: "{{ .Resource.Version }}"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	// AddToScheme is required by pkg/client/...
	AddToScheme = SchemeBuilder.AddToScheme
)

// Resource is required by pkg/client/listers/...
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}
`
