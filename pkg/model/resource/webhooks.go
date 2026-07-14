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
	"strings"

	"github.com/gobuffalo/flect"
	"k8s.io/apimachinery/pkg/util/validation"
)

// Webhooks contains information about scaffolded webhooks
type Webhooks struct {
	// WebhookVersion holds the {Validating,Mutating}WebhookConfiguration API version used for the resource.
	WebhookVersion string `json:"webhookVersion,omitempty"`

	// Defaulting specifies if a defaulting webhook is associated to the resource.
	Defaulting bool `json:"defaulting,omitempty"`

	// Validation specifies if a validation webhook is associated to the resource.
	Validation bool `json:"validation,omitempty"`

	// Conversion specifies if a conversion webhook is associated to the resource.
	Conversion bool `json:"conversion,omitempty"`

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
func (webhooks Webhooks) IsMultiGVK() bool {
	return webhooks.MultiGVK || webhooks.Name != ""
}

// Validate checks that the Webhooks is valid.
func (webhooks Webhooks) Validate() error {
	// Validate name for multi-GVK webhooks
	if webhooks.Name != "" {
		if errs := validation.IsDNS1035Label(webhooks.Name); len(errs) != 0 {
			return fmt.Errorf("invalid webhook name %q: %s", webhooks.Name, errs)
		}
	}

	// Validate the Webhook version
	if err := validateAPIVersion(webhooks.WebhookVersion); err != nil {
		return fmt.Errorf("invalid Webhook version: %w", err)
	}

	// Validate that Spoke versions are unique
	seen := map[string]bool{}
	for _, version := range webhooks.Spoke {
		if seen[version] {
			return fmt.Errorf("duplicate spoke version: %s", version)
		}
		seen[version] = true
	}

	return nil
}

// Copy returns a deep copy of the Webhooks that can be safely modified without affecting the original.
func (webhooks Webhooks) Copy() Webhooks {
	// Deep copy the Spoke slice
	var spokeCopy []string
	if len(webhooks.Spoke) > 0 {
		spokeCopy = make([]string, len(webhooks.Spoke))
		copy(spokeCopy, webhooks.Spoke)
	}

	var groupsCopy []string
	if len(webhooks.Groups) > 0 {
		groupsCopy = make([]string, len(webhooks.Groups))
		copy(groupsCopy, webhooks.Groups)
	}

	var kindsCopy []string
	if len(webhooks.Kinds) > 0 {
		kindsCopy = make([]string, len(webhooks.Kinds))
		copy(kindsCopy, webhooks.Kinds)
	}

	var versionsCopy []string
	if len(webhooks.Versions) > 0 {
		versionsCopy = make([]string, len(webhooks.Versions))
		copy(versionsCopy, webhooks.Versions)
	}

	return Webhooks{
		WebhookVersion: webhooks.WebhookVersion,
		Defaulting:     webhooks.Defaulting,
		Validation:     webhooks.Validation,
		Conversion:     webhooks.Conversion,
		Spoke:          spokeCopy,
		DefaultingPath: webhooks.DefaultingPath,
		ValidationPath: webhooks.ValidationPath,
		MultiGVK:       webhooks.MultiGVK,
		Name:           webhooks.Name,
		Groups:         groupsCopy,
		Kinds:          kindsCopy,
		Versions:       versionsCopy,
	}
}

// Update combines fields of the webhooks of two resources.
func (webhooks *Webhooks) Update(other *Webhooks) error {
	// If other is nil, nothing to merge
	if other == nil {
		return nil
	}

	// If other is a standalone multi-GVK webhook, copy it only if self is empty
	if other.IsMultiGVK() {
		if !webhooks.IsMultiGVK() && webhooks.IsEmpty() {
			*webhooks = other.Copy()
		}
		return nil
	}

	// Other is GVK-tied, merge fields
	if other.WebhookVersion != "" {
		if webhooks.WebhookVersion == "" {
			webhooks.WebhookVersion = other.WebhookVersion
		} else if webhooks.WebhookVersion != other.WebhookVersion {
			return fmt.Errorf("webhook versions do not match")
		}
	}

	// Update defaulting.
	webhooks.Defaulting = webhooks.Defaulting || other.Defaulting

	// Update validation.
	webhooks.Validation = webhooks.Validation || other.Validation

	// Update conversion.
	webhooks.Conversion = webhooks.Conversion || other.Conversion

	// Update Spoke (merge without duplicates)
	if len(other.Spoke) > 0 {
		existingSpokes := make(map[string]struct{})
		for _, spoke := range webhooks.Spoke {
			existingSpokes[spoke] = struct{}{}
		}
		for _, spoke := range other.Spoke {
			if _, exists := existingSpokes[spoke]; !exists {
				webhooks.Spoke = append(webhooks.Spoke, spoke)
			}
		}
	}

	// Update custom paths (other takes precedence if not empty)
	if other.DefaultingPath != "" {
		webhooks.DefaultingPath = other.DefaultingPath
	}
	if other.ValidationPath != "" {
		webhooks.ValidationPath = other.ValidationPath
	}

	return nil
}

// IsEmpty returns if the Webhooks' fields all contain zero-values.
func (webhooks Webhooks) IsEmpty() bool {
	return webhooks.WebhookVersion == "" &&
		!webhooks.Defaulting && !webhooks.Validation &&
		!webhooks.Conversion && len(webhooks.Spoke) == 0 &&
		webhooks.DefaultingPath == "" && webhooks.ValidationPath == "" &&
		!webhooks.MultiGVK && webhooks.Name == "" &&
		len(webhooks.Groups) == 0 && len(webhooks.Kinds) == 0 && len(webhooks.Versions) == 0
}

// AddSpoke adds a new spoke version to the Webhooks configuration.
func (webhooks *Webhooks) AddSpoke(version string) {
	// Ensure the version is not already present
	if slices.Contains(webhooks.Spoke, version) {
		return
	}
	webhooks.Spoke = append(webhooks.Spoke, version)
}

// WebhookToGVKs converts a Webhooks' Groups/Kinds/Versions to a list of GVKs.
// Groups are full domain strings (e.g., "crew.testproject.org"),
// Kinds are plural resource names (e.g., "captains"),
// Versions are API versions (e.g., "v1").
func WebhookToGVKs(wh Webhooks) []GVK {
	if len(wh.Groups) == 0 || len(wh.Kinds) == 0 || len(wh.Versions) == 0 {
		return nil
	}

	// Filter out wildcard versions since they cannot be represented as concrete GVKs.
	concreteVersions := make([]string, 0, len(wh.Versions))
	for _, v := range wh.Versions {
		if v != "*" {
			concreteVersions = append(concreteVersions, v)
		}
	}
	if len(concreteVersions) == 0 {
		return nil
	}

	gvks := make([]GVK, 0, len(wh.Groups)*len(wh.Kinds)*len(concreteVersions))
	for _, groupDomain := range wh.Groups {
		group := ExtractGroupFromDomain(groupDomain)
		for _, pluralKind := range wh.Kinds {
			singularKind := flect.Singularize(strings.ToLower(pluralKind))
			// Capitalize first letter to match Kind convention
			if len(singularKind) > 0 {
				singularKind = strings.ToUpper(singularKind[:1]) + singularKind[1:]
			}
			for _, version := range concreteVersions {
				gvks = append(gvks, GVK{
					Group:   group,
					Version: version,
					Kind:    singularKind,
				})
			}
		}
	}
	return gvks
}

// ExtractGroupFromDomain extracts the short group name from a fully qualified domain.
// For example, "crew.testproject.org" returns "crew", "core" returns "core".
func ExtractGroupFromDomain(groupDomain string) string {
	if before, _, ok := strings.Cut(groupDomain, "."); ok {
		return before
	}
	return groupDomain
}
