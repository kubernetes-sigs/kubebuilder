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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFeatureGates(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    FeatureGates
		expectError bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: FeatureGates{},
		},
		{
			name:     "single feature gate",
			input:    "feature1",
			expected: FeatureGates{"feature1": true},
		},
		{
			name:     "single feature gate with explicit true",
			input:    "feature1=true",
			expected: FeatureGates{"feature1": true},
		},
		{
			name:     "single feature gate with explicit false",
			input:    "feature1=false",
			expected: FeatureGates{"feature1": false},
		},
		{
			name:     "multiple feature gates",
			input:    "feature1,feature2",
			expected: FeatureGates{"feature1": true, "feature2": true},
		},
		{
			name:     "mixed enabled and disabled",
			input:    "feature1=true,feature2=false,feature3",
			expected: FeatureGates{"feature1": true, "feature2": false, "feature3": true},
		},
		{
			name:     "with whitespace",
			input:    " feature1 , feature2=false , feature3 ",
			expected: FeatureGates{"feature1": true, "feature2": false, "feature3": true},
		},
		{
			name:        "invalid format",
			input:       "feature1=invalid",
			expectError: true,
		},
		{
			name:        "empty feature name",
			input:       "=true",
			expectError: true,
		},
		{
			name:        "empty feature name with comma",
			input:       ",feature1",
			expectError: true,
		},
		{
			name:        "empty feature name with comma and space",
			input:       " ,feature1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseFeatureGates(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFeatureGates_IsEnabled(t *testing.T) {
	gates := FeatureGates{
		"enabled":  true,
		"disabled": false,
	}

	assert.True(t, gates.IsEnabled("enabled"))
	assert.False(t, gates.IsEnabled("disabled"))
	assert.False(t, gates.IsEnabled("nonexistent"))
}

func TestFeatureGates_GetEnabledGates(t *testing.T) {
	gates := FeatureGates{
		"enabled1":  true,
		"enabled2":  true,
		"disabled1": false,
		"disabled2": false,
	}

	enabled := gates.GetEnabledGates()
	assert.ElementsMatch(t, []string{"enabled1", "enabled2"}, enabled)
}

func TestFeatureGates_GetDisabledGates(t *testing.T) {
	gates := FeatureGates{
		"enabled1":  true,
		"enabled2":  true,
		"disabled1": false,
		"disabled2": false,
	}

	disabled := gates.GetDisabledGates()
	assert.ElementsMatch(t, []string{"disabled1", "disabled2"}, disabled)
}

func TestFeatureGates_String(t *testing.T) {
	tests := []struct {
		name     string
		gates    FeatureGates
		expected string
	}{
		{
			name:     "empty gates",
			gates:    FeatureGates{},
			expected: "",
		},
		{
			name:     "only enabled gates",
			gates:    FeatureGates{"feature1": true, "feature2": true},
			expected: "feature1,feature2",
		},
		{
			name:     "mixed gates",
			gates:    FeatureGates{"feature1": true, "feature2": false},
			expected: "feature1,feature2=false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.gates.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
