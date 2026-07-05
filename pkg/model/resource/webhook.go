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

package resource

import (
	"fmt"
	"slices"

	"k8s.io/apimachinery/pkg/util/validation"
)

// Webhook contains information about a scaffolded webhook.
// A Webhook can be either:
//   - GVK-tied: attached to a specific Resource (MultiGVK is false, uses parent GVK)
//   - Multi-GVK/standalone: intercepts multiple resource types (MultiGVK is true)
type Webhook struct {
	// WebhookVersion holds the {Validating,Mutating}WebhookConfiguration API version used for the resource.
	WebhookVersion string `json:"webhookVersion,omitempty"`

	// Defaulting specifies if a defaulting/mutating webhook is scaffolded.
	Defaulting bool `json:"defaulting,omitempty"`

	// Validation specifies if a validation webhook is scaffolded.
	Validation bool `json:"validation,omitempty"`

	// Conversion specifies if a conversion webhook is scaffolded (only for GVK-tied webhooks).
	Conversion bool `json:"conversion,omitempty"`

	// Spoke versions for conversion (only for GVK-tied webhooks).
	Spoke []string `json:"spoke,omitempty"`

	// DefaultingPath holds the custom path for the defaulting/mutating webhook.
	// This path is used in the +kubebuilder:webhook marker annotation.
	DefaultingPath string `json:"defaultingPath,omitempty"`

	// ValidationPath holds the custom path for the validation webhook.
	// This path is used in the +kubebuilder:webhook marker annotation.
	ValidationPath string `json:"validationPath,omitempty"`

	// MultiGVK specifies whether this is a standalone multi-GVK webhook (true)
	// or a GVK-tied webhook attached to a specific Resource (false).
	MultiGVK bool `json:"multiGVK,omitempty"`

	// Name is a unique identifier for multi-GVK (standalone) webhooks.
	// Empty for GVK-tied webhooks; when set, the webhook is standalone.
	// Deprecated: Use MultiGVK instead. Will be removed in a future version.
	Name string `json:"name,omitempty"`

	// Groups is the list of API groups the webhook intercepts (only for multi-GVK).
	// Use "" for the core group.
	Groups []string `json:"groups,omitempty"`

	// Kinds is the list of resource kinds the webhook intercepts (only for multi-GVK).
	Kinds []string `json:"resources,omitempty"`

	// Versions is the list of API versions the webhook intercepts (only for multi-GVK),
	// or "*" for all.
	Versions []string `json:"versions,omitempty"`
}

// IsMultiGVK returns true if this is a standalone multi-GVK webhook.
// Checks the explicit MultiGVK field, falling back to Name != "" for backward
// compatibility with existing PROJECT files.
func (w Webhook) IsMultiGVK() bool {
	return w.MultiGVK || w.Name != ""
}

// Validate checks that the Webhook is structurally valid.
func (w Webhook) Validate() error {
	// Validate name for multi-GVK webhooks
	if w.Name != "" {
		if errs := validation.IsDNS1035Label(w.Name); len(errs) != 0 {
			return fmt.Errorf("invalid webhook name %q: %s", w.Name, errs)
		}
	}

	// Validate the Webhook version
	if w.WebhookVersion == "" {
		return fmt.Errorf("webhook version is required")
	}
	if err := validateAPIVersion(w.WebhookVersion); err != nil {
		return fmt.Errorf("invalid webhook version: %w", err)
	}

	// Validate that Spoke versions are unique
	seen := map[string]bool{}
	for _, version := range w.Spoke {
		if seen[version] {
			return fmt.Errorf("duplicate spoke version: %s", version)
		}
		seen[version] = true
	}

	return nil
}

// Copy returns a deep copy of the Webhook that can be safely modified without affecting the original.
func (w Webhook) Copy() Webhook {
	var spokeCopy []string
	if len(w.Spoke) > 0 {
		spokeCopy = make([]string, len(w.Spoke))
		copy(spokeCopy, w.Spoke)
	}

	var groupsCopy []string
	if len(w.Groups) > 0 {
		groupsCopy = make([]string, len(w.Groups))
		copy(groupsCopy, w.Groups)
	}

	var kindsCopy []string
	if len(w.Kinds) > 0 {
		kindsCopy = make([]string, len(w.Kinds))
		copy(kindsCopy, w.Kinds)
	}

	var versionsCopy []string
	if len(w.Versions) > 0 {
		versionsCopy = make([]string, len(w.Versions))
		copy(versionsCopy, w.Versions)
	}

	return Webhook{
		WebhookVersion: w.WebhookVersion,
		Defaulting:     w.Defaulting,
		Validation:     w.Validation,
		Conversion:     w.Conversion,
		Spoke:          spokeCopy,
		DefaultingPath: w.DefaultingPath,
		ValidationPath: w.ValidationPath,
		MultiGVK:       w.MultiGVK,
		Name:           w.Name,
		Groups:         groupsCopy,
		Kinds:          kindsCopy,
		Versions:       versionsCopy,
	}
}

// Update combines fields of two webhooks favoring the receiver's values.
func (w *Webhook) Update(other *Webhook) error {
	if other == nil {
		return nil
	}

	// If other is a standalone multi-GVK webhook, copy it only if self is empty
	if other.IsMultiGVK() {
		if !w.IsMultiGVK() && w.IsEmpty() {
			*w = other.Copy()
		}
		return nil
	}

	// Other is GVK-tied, merge fields
	if other.WebhookVersion != "" {
		if w.WebhookVersion == "" {
			w.WebhookVersion = other.WebhookVersion
		} else if w.WebhookVersion != other.WebhookVersion {
			return fmt.Errorf("webhook versions do not match")
		}
	}

	w.Defaulting = w.Defaulting || other.Defaulting
	w.Validation = w.Validation || other.Validation
	w.Conversion = w.Conversion || other.Conversion

	// Merge spoke versions without duplicates
	for _, spoke := range other.Spoke {
		if !slices.Contains(w.Spoke, spoke) {
			w.Spoke = append(w.Spoke, spoke)
		}
	}

	if other.DefaultingPath != "" {
		w.DefaultingPath = other.DefaultingPath
	}
	if other.ValidationPath != "" {
		w.ValidationPath = other.ValidationPath
	}

	return nil
}

// IsEmpty returns if the Webhook's fields all contain zero-values.
func (w Webhook) IsEmpty() bool {
	return w.WebhookVersion == "" &&
		!w.Defaulting && !w.Validation &&
		!w.Conversion && len(w.Spoke) == 0 &&
		w.DefaultingPath == "" && w.ValidationPath == "" &&
		!w.MultiGVK && w.Name == "" &&
		len(w.Groups) == 0 && len(w.Kinds) == 0 && len(w.Versions) == 0
}

// AddSpoke adds a new spoke version to the Webhook configuration.
func (w *Webhook) AddSpoke(version string) {
	if slices.Contains(w.Spoke, version) {
		return
	}
	w.Spoke = append(w.Spoke, version)
}
