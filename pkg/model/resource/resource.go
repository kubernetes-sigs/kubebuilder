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

	"k8s.io/apimachinery/pkg/util/validation"
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

	// Controller specifies if a controller has been scaffolded (legacy, deprecated).
	// Deprecated: Use Controllers for multiple controller support.
	Controller bool `json:"controller,omitempty"`

	// Controllers holds named controllers for this resource.
	Controllers *Controllers `json:"controllers,omitempty"`

	// Webhooks holds the information related to the associated webhooks.
	Webhooks *Webhooks `json:"webhooks,omitempty"`

	// External specifies if the resource is defined externally.
	External bool `json:"external,omitempty"`

	// Module specifies the Go module path for external API dependencies.
	// Can optionally include @version to pin the dependency (e.g., "github.com/org/repo@v1.2.3").
	// This is only used when External is true.
	Module string `json:"module,omitempty"`

	// Core specifies if the resource is from Kubernetes API.
	Core bool `json:"core,omitempty"`
}

// Validate checks that the Resource is valid.
func (r Resource) Validate() error {
	// Validate the GVK
	if err := r.GVK.Validate(); err != nil {
		return err
	}

	// Validate the Plural
	// NOTE: IsDNS1035Label returns a slice of strings instead of an error, so no wrapping
	if errors := validation.IsDNS1035Label(r.Plural); len(errors) != 0 {
		return fmt.Errorf("invalid Plural: %#v", errors)
	}

	// TODO: validate the path

	// Validate the API
	if r.API != nil && !r.API.IsEmpty() {
		if err := r.API.Validate(); err != nil {
			return fmt.Errorf("invalid API: %w", err)
		}
	}

	// Validate the Webhooks
	if r.Webhooks != nil && !r.Webhooks.IsEmpty() {
		if err := r.Webhooks.Validate(); err != nil {
			return fmt.Errorf("invalid Webhooks: %w", err)
		}
	}

	// Validate the Controllers
	if r.Controllers != nil && !r.Controllers.IsEmpty() {
		if err := r.Controllers.Validate(); err != nil {
			return fmt.Errorf("invalid Controllers: %w", err)
		}
	}

	return nil
}

// Normalize handles the edge case where both controller: true and controllers: are set.
// This can only occur if someone manually edits the PROJECT file.
// The new controllers array format takes precedence, and the legacy flag is cleared.
func (r *Resource) Normalize() {
	if r == nil {
		return
	}

	// If both controller: true and controllers: are set (manual edit),
	// keep the explicit controllers array and clear the legacy flag
	if r.Controller && r.Controllers != nil && !r.Controllers.IsEmpty() {
		r.Controller = false
	}
}

// PackageName returns a name valid to be used for Go packages.
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

// HasController returns true if the resource has at least one associated controller.
// It checks both the legacy Controller bool field and the new Controllers field.
func (r Resource) HasController() bool {
	// Check legacy field first for backward compatibility
	if r.Controller {
		return true
	}
	// Check new Controllers field
	if r.Controllers != nil && !r.Controllers.IsEmpty() {
		return true
	}
	return false
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

// IsExternal returns true if the resource was scaffold as external.
func (r Resource) IsExternal() bool {
	return r.External
}

// IsRegularPlural returns true if the plural is the regular plural form for the kind.
func (r Resource) IsRegularPlural() bool {
	return r.Plural == RegularPlural(r.Kind)
}

// GetControllerNames returns the names of all controllers for this resource.
// For resources using the new Controllers field, it returns the actual controller names.
// For resources using the legacy Controller bool field, it returns a default name (lowercase kind).
// Returns nil if the resource has no controllers.
func (r Resource) GetControllerNames() []string {
	// New format: return explicit controller names
	if r.Controllers != nil && !r.Controllers.IsEmpty() {
		return r.Controllers.GetControllerNames()
	}

	// Legacy format: generate default name from kind
	if r.Controller {
		return []string{strings.ToLower(r.Kind)}
	}

	return nil
}

// Copy returns a deep copy of the Resource that can be safely modified without affecting the original.
func (r Resource) Copy() Resource {
	// As this function doesn't use a pointer receiver, r is already a shallow copy.
	// Any field that is a pointer, slice or map needs to be deep copied.
	if r.API != nil {
		api := r.API.Copy()
		r.API = &api
	}
	if r.Controllers != nil {
		controllers := r.Controllers.Copy()
		r.Controllers = &controllers
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
		return fmt.Errorf("cannot update a nil resource")
	}

	// Make sure we are not merging resources for different GVKs.
	if !r.IsEqualTo(other.GVK) {
		return fmt.Errorf("cannot update a resource (GVK %+v) with another with non-matching GVK %+v", r.GVK, other.GVK)
	}

	if r.Plural != other.Plural {
		return fmt.Errorf("cannot update resource (Plural %q) with another with non-matching Plural %q",
			r.Plural, other.Plural)
	}

	if other.Path != "" && r.Path != other.Path {
		if r.Path == "" {
			r.Path = other.Path
		} else {
			return fmt.Errorf("cannot update resource (Path %q) with another with non-matching Path %q", r.Path, other.Path)
		}
	}

	// Update API.
	if r.API == nil && other.API != nil {
		r.API = &API{}
	}
	if err := r.API.Update(other.API); err != nil {
		return err
	}

	// Update controllers
	if other.Controllers != nil && !other.Controllers.IsEmpty() {
		// Migrate legacy controller: true to the new controllers array format
		if r.Controller {
			if r.Controllers == nil {
				r.Controllers = &Controllers{}
			}
			// Add a default controller with a kind-based name
			defaultName := strings.ToLower(r.Kind)
			if !r.Controllers.HasController(defaultName) {
				_ = r.Controllers.AddController(defaultName)
			}
		}

		// Initialize controllers array if not yet created
		if r.Controllers == nil {
			r.Controllers = &Controllers{}
		}

		// Merge controllers from the other resource
		if err := r.Controllers.Update(other.Controllers); err != nil {
			return err
		}

		// Clear the legacy flag now that we're using the new format
		r.Controller = false
	} else {
		// Only update the legacy field if not using the new format
		r.Controller = r.Controller || other.Controller
	}

	// Update Webhooks.
	if r.Webhooks == nil && other.Webhooks != nil {
		r.Webhooks = &Webhooks{}
	}

	return r.Webhooks.Update(other.Webhooks)
}

func wrapKey(key string) string {
	return fmt.Sprintf("%%[%s]", key)
}

// Replacer returns a strings.Replacer that replaces resource keywords with values.
func (r Resource) Replacer() *strings.Replacer {
	replacements := make([]string, 0, 10)

	replacements = append(replacements, wrapKey("group"), r.Group)
	replacements = append(replacements, wrapKey("version"), r.Version)
	replacements = append(replacements, wrapKey("kind"), strings.ToLower(r.Kind))
	replacements = append(replacements, wrapKey("plural"), strings.ToLower(r.Plural))
	replacements = append(replacements, wrapKey("package-name"), r.PackageName())

	return strings.NewReplacer(replacements...)
}
