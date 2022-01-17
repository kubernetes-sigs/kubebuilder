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
}

// Validate checks that the Webhooks is valid.
func (webhooks Webhooks) Validate() error {
	// Validate the Webhook version
	if err := validateAPIVersion(webhooks.WebhookVersion); err != nil {
		return fmt.Errorf("invalid Webhook version: %w", err)
	}

	return nil
}

// Copy returns a deep copy of the API that can be safely modified without affecting the original.
func (webhooks Webhooks) Copy() Webhooks {
	// As this function doesn't use a pointer receiver, webhooks is already a shallow copy.
	// Any field that is a pointer, slice or map needs to be deep copied.
	return webhooks
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

	return nil
}

// IsEmpty returns if the Webhooks' fields all contain zero-values.
func (webhooks Webhooks) IsEmpty() bool {
	return webhooks.WebhookVersion == "" && !webhooks.Defaulting && !webhooks.Validation && !webhooks.Conversion
}
