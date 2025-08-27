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

package api

import (
	log "log/slog"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Spoke{}

// Spoke scaffolds the file that defines spoke version conversion
type Spoke struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	Force        bool
	SpokeVersion string
}

// SetTemplateDefaults implements file.Template
func (f *Spoke) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup && f.Resource.Group != "" {
			// Use SpokeVersion for dynamic file path generation
			f.Path = filepath.Join("api", f.Resource.Group, f.SpokeVersion, "%[kind]_conversion.go")
		} else {
			f.Path = filepath.Join("api", f.SpokeVersion, "%[kind]_conversion.go")
		}
	}

	// Replace placeholders in the path
	f.Path = f.Resource.Replacer().Replace(f.Path)
	log.Info("Creating spoke conversion file", "path", f.Path)

	f.TemplateBody = spokeTemplate

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.SkipFile
	}

	return nil
}

//nolint:lll
const spokeTemplate = `{{ .Boilerplate }}

package {{ .SpokeVersion }}

import (
	"log"

    "sigs.k8s.io/controller-runtime/pkg/conversion"
    {{ .Resource.ImportAlias }} "{{ .Resource.Path }}"
)

// ConvertTo converts this {{ .Resource.Kind }} ({{ .SpokeVersion }}) to the Hub version ({{ .Resource.Version }}).
func (src *{{ .Resource.Kind }}) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*{{ .Resource.ImportAlias }}.{{ .Resource.Kind }})
	log.Printf("ConvertTo: Converting {{ .Resource.Kind }} from Spoke version {{ .SpokeVersion }} to Hub version {{ .Resource.Version }};" +
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)
	
	// TODO(user): Implement conversion logic from {{ .SpokeVersion }} to {{ .Resource.Version }}
	// Example: Copying Spec fields
	// dst.Spec.Size = src.Spec.Replicas

	// Copy ObjectMeta to preserve name, namespace, labels, etc.
	dst.ObjectMeta = src.ObjectMeta

	return nil
}

// ConvertFrom converts the Hub version ({{ .Resource.Version }}) to this {{ .Resource.Kind }} ({{ .SpokeVersion }}).
func (dst *{{ .Resource.Kind }}) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*{{ .Resource.ImportAlias }}.{{ .Resource.Kind }})
	log.Printf("ConvertFrom: Converting {{ .Resource.Kind }} from Hub version {{ .Resource.Version }} to Spoke version {{ .SpokeVersion }};" +
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// TODO(user): Implement conversion logic from {{ .Resource.Version }} to {{ .SpokeVersion }}
	// Example: Copying Spec fields
	// dst.Spec.Replicas = src.Spec.Size

	// Copy ObjectMeta to preserve name, namespace, labels, etc.
	dst.ObjectMeta = src.ObjectMeta

	return nil
}
`
