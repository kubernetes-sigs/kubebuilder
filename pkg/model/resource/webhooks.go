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
}

// Validate checks that the Webhooks is valid.
func (webhooks Webhooks) Validate() error {
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

// Copy returns a deep copy of the API that can be safely modified without affecting the original.
func (webhooks Webhooks) Copy() Webhooks {
	// Deep copy the Spoke slice
	var spokeCopy []string
	if len(webhooks.Spoke) > 0 {
		spokeCopy = make([]string, len(webhooks.Spoke))
		copy(spokeCopy, webhooks.Spoke)
	} else {
		spokeCopy = nil
	}

	return Webhooks{
		WebhookVersion: webhooks.WebhookVersion,
		Defaulting:     webhooks.Defaulting,
		Validation:     webhooks.Validation,
		Conversion:     webhooks.Conversion,
		Spoke:          spokeCopy,
	}
}

// Update combines fields of the webhooks of two resources.
func (webhooks *Webhooks) Update(other *Webhooks) error {
	// If other is nil, nothing to merge
	if other == nil {
		return nil
	}

	// Update the version.
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

	return nil
}

// IsEmpty returns if the Webhooks' fields all contain zero-values.
func (webhooks Webhooks) IsEmpty() bool {
	return webhooks.WebhookVersion == "" &&
		!webhooks.Defaulting && !webhooks.Validation &&
		!webhooks.Conversion && len(webhooks.Spoke) == 0
}

// AddSpoke adds a new spoke version to the Webhooks configuration.
func (webhooks *Webhooks) AddSpoke(version string) {
	// Ensure the version is not already present
	for _, v := range webhooks.Spoke {
		if v == version {
			return
		}
	}
	webhooks.Spoke = append(webhooks.Spoke, version)
}
