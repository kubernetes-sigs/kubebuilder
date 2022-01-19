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

// API contains information about scaffolded APIs
type API struct {
	// CRDVersion holds the CustomResourceDefinition API version used for the resource.
	CRDVersion string `json:"crdVersion,omitempty"`

	// Namespaced is true if the API is namespaced.
	Namespaced bool `json:"namespaced,omitempty"`
}

// Validate checks that the API is valid.
func (api API) Validate() error {
	// Validate the CRD version
	if err := validateAPIVersion(api.CRDVersion); err != nil {
		return fmt.Errorf("invalid CRD version: %w", err)
	}

	return nil
}

// Copy returns a deep copy of the API that can be safely modified without affecting the original.
func (api API) Copy() API {
	// As this function doesn't use a pointer receiver, api is already a shallow copy.
	// Any field that is a pointer, slice or map needs to be deep copied.
	return api
}

// Update combines fields of the APIs of two resources.
func (api *API) Update(other *API) error {
	// If other is nil, nothing to merge
	if other == nil {
		return nil
	}

	// Update the version.
	if other.CRDVersion != "" {
		if api.CRDVersion == "" {
			api.CRDVersion = other.CRDVersion
		} else if api.CRDVersion != other.CRDVersion {
			return fmt.Errorf("CRD versions do not match")
		}
	}

	// Update the namespace.
	api.Namespaced = api.Namespaced || other.Namespaced

	return nil
}

// IsEmpty returns if the API's fields all contain zero-values.
func (api API) IsEmpty() bool {
	return api.CRDVersion == "" && !api.Namespaced
}
