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
	// GVK contains the resource's Group-Version-Kind triplet.
	GVK `json:",inline"`

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

// HasAPI returns true if the resource has an associated API.
func (r Resource) HasAPI() bool {
	return r.API != nil && r.API.CRDVersion != ""
}

// HasController returns true if the resource has an associated controller.
func (r Resource) HasController() bool {
	return r.Controller
}

// HasDefaultingWebhook returns true if the resource has an associated defaulting webhook.
func (r Resource) HasDefaultingWebhook() bool {
	return r.Webhooks != nil && r.Webhooks.Defaulting
}

// HasValidationWebhook returns true if the resource has an associated validation webhook.
func (r Resource) HasValidationWebhook() bool {
	return r.Webhooks != nil && r.Webhooks.Validation
}

// HasConversionWebhook returns true if the resource has an associated conversion webhook.
func (r Resource) HasConversionWebhook() bool {
	return r.Webhooks != nil && r.Webhooks.Conversion
}

// Copy returns a deep copy of the Resource that can be safely modified without affecting the original.
func (r Resource) Copy() Resource {
	// As this function doesn't use a pointer receiver, r is already a shallow copy.
	// Any field that is a pointer, slice or map needs to be deep copied.
	if r.API != nil {
		api := r.API.Copy()
		r.API = &api
	}
	if r.Webhooks != nil {
		webhooks := r.Webhooks.Copy()
		r.Webhooks = &webhooks
	}
	return r
}

// Update combines fields of two resources that have matching GVK favoring the receiver's values.
func (r *Resource) Update(other Resource) error {
	// If self is nil, return an error
	if r == nil {
		return fmt.Errorf("unable to update a nil Resource")
	}

	// Make sure we are not merging resources for different GVKs.
	if !r.GVK.IsEqualTo(other.GVK) {
		return fmt.Errorf("unable to update a Resource with another with non-matching GVK")
	}

	// TODO: currently Plural & Path will always match. In the future, this may not be true (e.g. providing a
	//       --plural flag). In that case, we should yield an error in case of updating two resources with different
	//       values for these fields.

	// Update API.
	if r.API == nil && other.API != nil {
		r.API = &API{}
	}
	if err := r.API.Update(other.API); err != nil {
		return err
	}

	// Update controller.
	r.Controller = r.Controller || other.Controller

	// Update Webhooks.
	if r.Webhooks == nil && other.Webhooks != nil {
		r.Webhooks = &Webhooks{}
	}
	if err := r.Webhooks.Update(other.Webhooks); err != nil {
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
