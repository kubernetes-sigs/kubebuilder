/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www/apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package machinery

import (
	"fmt"
	"strings"
)

// FeatureGate represents a feature gate with its name and enabled state
type FeatureGate struct {
	Name    string
	Enabled bool
}

// FeatureGates represents a collection of feature gates
type FeatureGates map[string]bool

// ParseFeatureGates parses a comma-separated string of feature gates
// Format: "feature1=true,feature2=false,feature3"
// If no value is specified, the feature gate defaults to enabled
func ParseFeatureGates(featureGates string) (FeatureGates, error) {
	if featureGates == "" {
		return make(FeatureGates), nil
	}

	gates := make(FeatureGates)
	parts := strings.Split(featureGates, ",")

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			// If this is the first part and it's empty, it's an error
			if i == 0 {
				return nil, fmt.Errorf("empty feature gate name")
			}
			continue
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

			switch strings.ToLower(value) {
			case "true", "1", "yes", "on":
				enabled = true
			case "false", "0", "no", "off":
				enabled = false
			default:
				return nil, fmt.Errorf("invalid feature gate value for %s: %s", name, value)
			}
		} else {
			name = part
		}

		if name == "" {
			return nil, fmt.Errorf("empty feature gate name")
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

	var parts []string
	for name, enabled := range fg {
		if enabled {
			parts = append(parts, name)
		} else {
			parts = append(parts, fmt.Sprintf("%s=false", name))
		}
	}
	return strings.Join(parts, ",")
}
