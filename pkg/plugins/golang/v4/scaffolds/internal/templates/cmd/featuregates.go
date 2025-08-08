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

package cmd

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &FeatureGates{}

// FeatureGates scaffolds a feature gates package
type FeatureGates struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
	machinery.RepositoryMixin

	// Available feature gates discovered from the project
	AvailableGates []string
}

// SetTemplateDefaults implements machinery.Template
func (f *FeatureGates) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = "internal/featuregates/featuregates.go"
	}

	f.TemplateBody = featureGatesTemplate

	return nil
}

const featureGatesTemplate = `{{ .Boilerplate }}

package featuregates

import (
	"fmt"
	"strings"
)

// FeatureGates represents a map of feature gate names to their enabled state
type FeatureGates map[string]bool

// GetAvailableFeatureGates returns a list of available feature gates for this project
func GetAvailableFeatureGates() []string {
	return []string{
{{- range .AvailableGates }}
		"{{ . }}",
{{- end }}
	}
}

// GetFeatureGatesHelpText returns the help text for the feature-gates flag
func GetFeatureGatesHelpText() string {
	gates := GetAvailableFeatureGates()
	if len(gates) == 0 {
		return "No feature gates available"
	}
	return strings.Join(gates, ", ")
}

// ValidateFeatureGates validates that all specified feature gates are available
func ValidateFeatureGates(specifiedGates FeatureGates) error {
	availableGates := GetAvailableFeatureGates()
	availableMap := make(map[string]bool)

	for _, gate := range availableGates {
		availableMap[gate] = true
	}

	var invalidGates []string
	for gate := range specifiedGates {
		if !availableMap[gate] {
			invalidGates = append(invalidGates, gate)
		}
	}

	if len(invalidGates) > 0 {
		return fmt.Errorf("invalid feature gates: %s. Available gates: %s",
			strings.Join(invalidGates, ", "),
			strings.Join(availableGates, ", "))
	}

	return nil
}

// ParseFeatureGates parses the feature gates string into a map
func ParseFeatureGates(featureGates string) (FeatureGates, error) {
	gates := make(FeatureGates)
	
	if featureGates == "" {
		return gates, nil
	}
	
	parts := strings.Split(featureGates, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				gateName := strings.TrimSpace(kv[0])
				if gateName == "" {
					return nil, fmt.Errorf("empty feature gate name")
				}
				gates[gateName] = strings.TrimSpace(kv[1]) == "true"
			} else {
				return nil, fmt.Errorf("invalid feature gate format: %s", part)
			}
		} else {
			if part == "" {
				return nil, fmt.Errorf("empty feature gate name")
			}
			gates[part] = true
		}
	}
	
	return gates, nil
}

// IsEnabled checks if a specific feature gate is enabled
func (fg FeatureGates) IsEnabled(gateName string) bool {
	return fg[gateName]
}

// GetEnabledGates returns a list of enabled feature gates
func (fg FeatureGates) GetEnabledGates() []string {
	var enabled []string
	for gate, isEnabled := range fg {
		if isEnabled {
			enabled = append(enabled, gate)
		}
	}
	return enabled
}

// GetDisabledGates returns a list of disabled feature gates
func (fg FeatureGates) GetDisabledGates() []string {
	var disabled []string
	for gate, isEnabled := range fg {
		if !isEnabled {
			disabled = append(disabled, gate)
		}
	}
	return disabled
}

// String returns a string representation of the feature gates
func (fg FeatureGates) String() string {
	var parts []string
	for gate, enabled := range fg {
		if enabled {
			parts = append(parts, gate)
		} else {
			parts = append(parts, gate+"=false")
		}
	}
	return strings.Join(parts, ",")
}
`
