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

	"sigs.k8s.io/kubebuilder/v2/pkg/model/config"
)

// Resource contains the information required to scaffold files for a resource.
// nolint: maligned
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

	// Endpoint is the API endpoint which is used import the go package of the Resource.
	Endpoint string `json:"endpoint,omitempty"`

	// QualifiedGroup is the Group + "." + QualifiedGroup of the Resource.
	QualifiedGroup string `json:"qualified-group,omitempty"`

	// Domain is the Group + "." + QualifiedGroup of the Resource.
	Domain string `json:"domain,omitempty"`

	// Controller holds true when the controller is scaffolded
	Controller bool `json:"controller,omitempty"`

	// Namespaced is true if the resource is namespaced.
	// todo: remove when the v2 is no longer supported
	Namespaced bool `json:"namespaced,omitempty"`

	// API holds the the api data that is scaffolded
	API config.API `json:"api,omitempty"`

	// Webhooks holds webhooks data that is scaffolded
	Webhooks config.Webhooks `json:"webhooks,omitempty"`
}

// Data returns the ResourceData information to check against tracked resources in the configuration file
func (r *Resource) Data() config.ResourceData {
	return config.ResourceData{
		Group:      r.Group,
		Version:    r.Version,
		Kind:       r.Kind,
		Domain:     r.Domain,
		Plural:     r.Plural,
		Endpoint:   r.Endpoint,
		Controller: r.Controller,
		API:        &r.API,
		Webhooks:   &r.Webhooks,
	}
}

func wrapKey(key string) string {
	return fmt.Sprintf("%%[%s]", key)
}

// Replacer returns a strings.Replacer that replaces resource keywords with values.
func (r Resource) Replacer() *strings.Replacer {
	var replacements []string

	replacements = append(replacements, wrapKey("group"), r.Group)
	replacements = append(replacements, wrapKey("group-package-name"), r.GroupPackageName)
	replacements = append(replacements, wrapKey("version"), r.Version)
	replacements = append(replacements, wrapKey("kind"), strings.ToLower(r.Kind))
	replacements = append(replacements, wrapKey("plural"), strings.ToLower(r.Plural))

	return strings.NewReplacer(replacements...)
}
