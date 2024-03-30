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
	"regexp"
	"strings"

	"sigs.k8s.io/kubebuilder/v3/pkg/internal/validation"
)

const (
	versionPattern = "^v\\d+(?:alpha\\d+|beta\\d+)?$"

	groupRequired   = "group cannot be empty if the domain is empty"
	versionRequired = "version cannot be empty"
	kindRequired    = "kind cannot be empty"
)

var (
	versionRegex = regexp.MustCompile(versionPattern)
)

// GVK stores the Group - Version - Kind triplet that uniquely identifies a resource.
// In kubebuilder, the k8s fully qualified group is stored as Group and Domain to improve UX.
type GVK struct {
	Group   string `json:"group,omitempty"`
	Domain  string `json:"domain,omitempty"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
}

// Validate checks that the GVK is valid.
func (gvk GVK) Validate() error {
	// Check if the qualified group has a valid DNS1123 subdomain value
	if gvk.QualifiedGroup() == "" {
		return fmt.Errorf(groupRequired)
	}
	if err := validation.IsDNS1123Subdomain(gvk.QualifiedGroup()); err != nil {
		// NOTE: IsDNS1123Subdomain returns a slice of strings instead of an error, so no wrapping
		return fmt.Errorf("either Group or Domain is invalid: %s", err)
	}

	// Check if the version follows the valid pattern
	if gvk.Version == "" {
		return fmt.Errorf(versionRequired)
	}
	if !versionRegex.MatchString(gvk.Version) {
		return fmt.Errorf("Version must match %s (was %s)", versionPattern, gvk.Version)
	}

	// Check if kind has a valid DNS1035 label value
	if gvk.Kind == "" {
		return fmt.Errorf(kindRequired)
	}
	if errors := validation.IsDNS1035Label(strings.ToLower(gvk.Kind)); len(errors) != 0 {
		// NOTE: IsDNS1035Label returns a slice of strings instead of an error, so no wrapping
		return fmt.Errorf("invalid Kind: %#v", errors)
	}

	// Require kind to start with an uppercase character
	// NOTE: previous validation already fails for empty strings, gvk.Kind[0] will not panic
	if string(gvk.Kind[0]) == strings.ToLower(string(gvk.Kind[0])) {
		return fmt.Errorf("invalid Kind: must start with an uppercase character")
	}

	return nil
}

// QualifiedGroup returns the fully qualified group name with the available information.
func (gvk GVK) QualifiedGroup() string {
	switch "" {
	case gvk.Domain:
		return gvk.Group
	case gvk.Group:
		return gvk.Domain
	default:
		return fmt.Sprintf("%s.%s", gvk.Group, gvk.Domain)
	}
}

// IsEqualTo compares two GVK objects.
func (gvk GVK) IsEqualTo(other GVK) bool {
	return gvk.Group == other.Group &&
		gvk.Domain == other.Domain &&
		gvk.Version == other.Version &&
		gvk.Kind == other.Kind
}
