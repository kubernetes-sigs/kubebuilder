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

package v4

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

func TestFeatureGateDiscoveryInTestdata(t *testing.T) {
	// Test feature gate discovery in our testdata project
	parser := machinery.NewFeatureGateMarkerParser()

	// Parse the testdata project API types
	markers, err := parser.ParseFile("../../../testdata/project-v4/api/v1/captain_types.go")
	assert.NoError(t, err)
	assert.NotEmpty(t, markers, "Should discover feature gate markers")

	// Extract feature gate names
	featureGates := machinery.ExtractFeatureGates(markers)

	// Verify we found the expected feature gates
	expectedGates := []string{
		"experimental-bar",
	}

	for _, expectedGate := range expectedGates {
		assert.Contains(t, featureGates, expectedGate,
			"Should discover feature gate: %s", expectedGate)
	}

	t.Logf("Discovered feature gates: %v", featureGates)
}

func TestFeatureGateDiscoveryIgnoresCommentedFields(t *testing.T) {
	// Test that commented fields with feature gate markers are not discovered
	parser := machinery.NewFeatureGateMarkerParser()

	// Create a temporary file with commented feature gate markers
	tempContent := `package test

type TestStruct struct {
	// This field is commented out and should not be discovered
	// +feature-gate commented-feature
	// Field *string ` + "`" + `json:"field,omitempty"` + "`" + `

	// This field is active and should be discovered
	// +feature-gate active-feature
	ActiveField *string ` + "`" + `json:"activeField,omitempty"` + "`" + `
}
`

	// Write to temporary file
	tmpFile, err := os.CreateTemp("", "featuregate_test_*.go")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(tempContent)
	assert.NoError(t, err)
	tmpFile.Close()

	// Parse the temporary file
	markers, err := parser.ParseFile(tmpFile.Name())
	assert.NoError(t, err)

	// Extract feature gate names
	featureGates := machinery.ExtractFeatureGates(markers)

	// Should only discover the active feature gate, not the commented one
	assert.Contains(t, featureGates, "active-feature", "Should discover active feature gate")
	assert.NotContains(t, featureGates, "commented-feature", "Should NOT discover commented feature gate")

	t.Logf("Discovered feature gates: %v", featureGates)
}

func TestFeatureGateParsing(t *testing.T) {
	// Test various feature gate parsing scenarios
	testCases := []struct {
		name     string
		input    string
		expected map[string]bool
		hasError bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: map[string]bool{},
			hasError: false,
		},
		{
			name:     "single feature gate",
			input:    "experimental-bar",
			expected: map[string]bool{"experimental-bar": true},
			hasError: false,
		},
		{
			name:  "multiple feature gates",
			input: "experimental-bar,advanced-features",
			expected: map[string]bool{
				"experimental-bar":  true,
				"advanced-features": true,
			},
			hasError: false,
		},
		{
			name:  "mixed enabled and disabled",
			input: "experimental-bar=true,advanced-features=false",
			expected: map[string]bool{
				"experimental-bar":  true,
				"advanced-features": false,
			},
			hasError: false,
		},
		{
			name:     "invalid format",
			input:    "experimental-bar=invalid",
			expected: nil,
			hasError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := machinery.ParseFeatureGates(tc.input)

			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, machinery.FeatureGates(tc.expected), result)
			}
		})
	}
}
