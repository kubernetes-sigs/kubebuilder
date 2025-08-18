/*
Copyright 2025 The Kubernetes Authors.

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

package machinery

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// featureGateNameRegex validates feature gate names
// Names should follow Kubernetes conventions: start with lowercase letter, followed by lowercase alphanumeric with hyphens
var featureGateNameRegex = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// FeatureGate represents a feature gate with its name and enabled state
type FeatureGate struct {
	Name    string
	Enabled bool
}

// FeatureGates represents a collection of feature gates
type FeatureGates map[string]bool

// validateFeatureGateName validates that a feature gate name follows conventions
func validateFeatureGateName(name string) error {
	if name == "" {
		return fmt.Errorf("empty feature gate name")
	}

	if !featureGateNameRegex.MatchString(name) {
		return fmt.Errorf("invalid feature gate name '%s': must be lowercase alphanumeric with hyphens (e.g., 'experimental-feature')", name)
	}

	return nil
}

// ParseFeatureGates parses a comma-separated string of feature gates
// Format: "feature1=true,feature2=false,feature3=true"
// Feature gate names must be lowercase alphanumeric with hyphens
// Values must be 'true' or 'false' (case insensitive)
func ParseFeatureGates(featureGates string) (FeatureGates, error) {
	if featureGates == "" {
		return make(FeatureGates), nil
	}

	gates := make(FeatureGates)
	parts := strings.Split(featureGates, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("empty feature gate name in input")
		}

		// Parse the feature gate name and value
		var name string
		var enabled bool = true // Default to enabled

		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) != 2 {
				return nil, fmt.Errorf("invalid feature gate format: %s", part)
			}
			name = strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			if value == "" {
				return nil, fmt.Errorf("empty value for feature gate '%s'", name)
			}

			switch strings.ToLower(value) {
			case "true":
				enabled = true
			case "false":
				enabled = false
			default:
				return nil, fmt.Errorf("invalid feature gate value for %s: %s (must be 'true' or 'false')", name, value)
			}
		} else {
			// No '=' found, treat entire part as name with default value true
			name = part
		}

		// Validate the feature gate name
		if err := validateFeatureGateName(name); err != nil {
			return nil, err
		}

		gates[name] = enabled
	}

	return gates, nil
}

// IsEnabled checks if a specific feature gate is enabled
func (fg FeatureGates) IsEnabled(name string) bool {
	enabled, exists := fg[name]
	return exists && enabled
}

// GetEnabledGates returns a list of enabled feature gate names
func (fg FeatureGates) GetEnabledGates() []string {
	var enabled []string
	for name, isEnabled := range fg {
		if isEnabled {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

// GetDisabledGates returns a list of disabled feature gate names
func (fg FeatureGates) GetDisabledGates() []string {
	var disabled []string
	for name, enabled := range fg {
		if !enabled {
			disabled = append(disabled, name)
		}
	}
	return disabled
}

// String returns a string representation of the feature gates
func (fg FeatureGates) String() string {
	if len(fg) == 0 {
		return ""
	}

	// Get feature gate names and sort them for consistent output
	names := make([]string, 0, len(fg))
	for name := range fg {
		names = append(names, name)
	}
	sort.Strings(names)

	var parts []string
	for _, name := range names {
		enabled := fg[name]
		if enabled {
			parts = append(parts, name)
		} else {
			parts = append(parts, fmt.Sprintf("%s=false", name))
		}
	}
	return strings.Join(parts, ",")
}
