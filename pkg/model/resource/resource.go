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

package resource

import (
	"fmt"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/model/config"
)

// Resource contains the information required to scaffold files for a resource.
type Resource struct {
	// Group is the API Group. Does not contain the domain.
	Group string `json:"group,omitempty"`

	// GroupPackageName is the API Group cleaned to be used as the package name.
	GroupPackageName string `json:"-"`

	// Version is the API version.
	Version string `json:"version,omitempty"`

	// Kind is the API Kind.
	Kind string `json:"kind,omitempty"`

	// Plural is the API Kind plural form.
	Plural string `json:"plural,omitempty"`

	// ImportAlias is a cleaned concatenation of Group and Version.
	ImportAlias string `json:"-"`

	// Package is the go package of the Resource.
	Package string `json:"package,omitempty"`

	// Domain is the Group + "." + Domain of the Resource.
	Domain string `json:"domain,omitempty"`

	// Namespaced is true if the resource is namespaced.
	Namespaced bool `json:"namespaced,omitempty"`
}

// GVK returns the group-version-kind information to check against tracked resources in the configuration file
func (r *Resource) GVK() config.GVK {
	return config.GVK{
		Group:   r.Group,
		Version: r.Version,
		Kind:    r.Kind,
	}
}

func wrapKey(key string) string {
	return fmt.Sprintf("%%[%s]", key)
}

func (r Resource) Replacer() *strings.Replacer {
	var replacements []string

	replacements = append(replacements, wrapKey("group"), r.Group)
	replacements = append(replacements, wrapKey("group-package-name"), r.GroupPackageName)
	replacements = append(replacements, wrapKey("version"), r.Version)
	replacements = append(replacements, wrapKey("kind"), strings.ToLower(r.Kind))
	replacements = append(replacements, wrapKey("plural"), strings.ToLower(r.Plural))

	return strings.NewReplacer(replacements...)
}
