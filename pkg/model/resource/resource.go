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
)

// Resource contains the information required to scaffold files for a resource.
type Resource struct {
	// Group is the resource's group. Does not contain the domain.
	Group string `json:"group,omitempty"`
	// Domain is the resource's domain.
	Domain string `json:"domain,omitempty"`
	// Version is the resource's version.
	Version string `json:"version,omitempty"`
	// Kind is the resource's kind.
	Kind string `json:"kind,omitempty"`

	// Plural is the resource's kind plural form.
	Plural string `json:"plural,omitempty"`

	// Path is the path to the go package where the types are defined.
	Path string `json:"path,omitempty"`

	// API holds the information related to the resource API.
	API *API `json:"api,omitempty"`

	// Controller specifies if a controller has been scaffolded.
	Controller bool `json:"controller,omitempty"`

	// Webhooks holds the information related to the associated webhooks.
	Webhooks *Webhooks `json:"webhooks,omitempty"`
}

// QualifiedGroup returns the fully qualified group name with the available information.
func (r Resource) QualifiedGroup() string {
	switch "" {
	case r.Domain:
		return r.Group
	case r.Group:
		return r.Domain
	default:
		return fmt.Sprintf("%s.%s", r.Group, r.Domain)
	}
}

// PackageName returns a name valid to be used por go packages.
func (r Resource) PackageName() string {
	if r.Group == "" {
		return safeImport(r.Domain)
	}

	return safeImport(r.Group)
}

// ImportAlias returns a identifier usable as an import alias for this resource.
func (r Resource) ImportAlias() string {
	if r.Group == "" {
		return safeImport(r.Domain + r.Version)
	}

	return safeImport(r.Group + r.Version)
}

// GVK returns the GVK identifier of a resource.
func (r Resource) GVK() GVK {
	return GVK{
		Group:   r.QualifiedGroup(),
		Version: r.Version,
		Kind:    r.Kind,
	}
}

// Update combines fields of two resources that have matching GVK favoring the receiver's values.
func (r *Resource) Update(other *Resource) error {
	// If other is nil, nothing to merge
	if other == nil {
		return nil
	}

	// If self is nil, set to other
	if r == nil {
		return fmt.Errorf("unable to update a nil Resource")
	}

	// Make sure we are not merging resources for different GVKs.
	if !r.GVK().IsEqualTo(other.GVK()) {
		return fmt.Errorf("unable to update a Resource with another with non-matching GVK")
	}

	// TODO: currently Plural & Path will always match. In the future, this may not be true (e.g. providing a
	//       --plural flag). In that case, we should yield an error in case of updating two resources with different
	//       values for these fields.

	// Update API.
	if r.API == nil && other.API != nil {
		r.API = &API{}
	}
	if err := r.API.update(other.API); err != nil {
		return err
	}

	// Update controller.
	r.Controller = r.Controller || other.Controller

	// Update Webhooks.
	if r.Webhooks == nil && other.Webhooks != nil {
		r.Webhooks = &Webhooks{}
	}
	if err := r.Webhooks.update(other.Webhooks); err != nil {
		return err
	}

	return nil
}

func wrapKey(key string) string {
	return fmt.Sprintf("%%[%s]", key)
}

// Replacer returns a strings.Replacer that replaces resource keywords with values.
func (r Resource) Replacer() *strings.Replacer {
	var replacements []string

	replacements = append(replacements, wrapKey("group"), r.Group)
	replacements = append(replacements, wrapKey("version"), r.Version)
	replacements = append(replacements, wrapKey("kind"), strings.ToLower(r.Kind))
	replacements = append(replacements, wrapKey("plural"), strings.ToLower(r.Plural))
	replacements = append(replacements, wrapKey("package-name"), r.PackageName())

	return strings.NewReplacer(replacements...)
}

// API holds the information related to the golang type definition and the CRD.
type API struct {
	// Version holds the CustomResourceDefinition API version used for the resource.
	Version string `json:"crdVersion,omitempty"`

	// Namespaced is true if the resource is namespaced.
	Namespaced bool `json:"namespaced,omitempty"`
}

// update combines fields of the APIs of two resources.
func (api *API) update(other *API) error {
	// If other is nil, nothing to merge
	if other == nil {
		return nil
	}

	// Update the version.
	if other.Version != "" {
		if api.Version == "" {
			api.Version = other.Version
		} else if api.Version != other.Version {
			return fmt.Errorf("CRD versions do not match")
		}
	}

	// Update the namespace.
	api.Namespaced = api.Namespaced || other.Namespaced

	return nil
}

// IsEmpty returns if the API's fields all contain zero-values.
func (api API) IsEmpty() bool {
	return api.Version == "" && !api.Namespaced
}

// Webhooks holds the information related to the associated webhooks.
type Webhooks struct {
	// Version holds the {Validating,Mutating}WebhookConfiguration API version used for the resource.
	Version string `json:"webhookVersion,omitempty"`

	// Defaulting specifies if a defaulting webhook is associated to the resource.
	Defaulting bool `json:"defaulting,omitempty"`

	// Validation specifies if a validation webhook is associated to the resource.
	Validation bool `json:"validating,omitempty"`

	// Conversion specifies if a conversion webhook is associated to the resource.
	Conversion bool `json:"conversion,omitempty"`
}

// update combines fields of the webhooks of two resources.
func (webhooks *Webhooks) update(other *Webhooks) error {
	// If other is nil, nothing to merge
	if other == nil {
		return nil
	}

	// Update the version.
	if other.Version != "" {
		if webhooks.Version == "" {
			webhooks.Version = other.Version
		} else if webhooks.Version != other.Version {
			return fmt.Errorf("webhook versions do not match")
		}
	}

	// Update defaulting.
	webhooks.Defaulting = webhooks.Defaulting || other.Defaulting

	// Update validation.
	webhooks.Validation = webhooks.Validation || other.Validation

	// Update conversion.
	webhooks.Conversion = webhooks.Conversion || other.Conversion

	return nil
}

// IsEmpty returns if the Webhooks' fields all contain zero-values.
func (webhooks Webhooks) IsEmpty() bool {
	return webhooks.Version == "" && !webhooks.Defaulting && !webhooks.Validation && !webhooks.Conversion
}
