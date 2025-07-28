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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFeatureGateMarkerParser_ParseFile(t *testing.T) {
	parser := NewFeatureGateMarkerParser()

	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	content := `package test

import "fmt"

// MyStruct represents a test struct
type MyStruct struct {
	// Regular field
	Name string ` + "`" + `json:"name"` + "`" + `

	// Feature gated field
	// +feature-gate experimental-feature
	ExperimentalField string ` + "`" + `json:"experimentalField,omitempty"` + "`" + `

	// Another feature gated field
	// +feature-gate another-feature
	AnotherField int ` + "`" + `json:"anotherField,omitempty"` + "`" + `

	// Regular comment
	// This is not a feature gate marker
	RegularField bool ` + "`" + `json:"regularField"` + "`" + `
}

// +feature-gate function-feature
func (m *MyStruct) ExperimentalMethod() {
	fmt.Println("This is experimental")
}
`

	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NoError(t, err)

	markers, err := parser.ParseFile(testFile)
	assert.NoError(t, err)
	assert.Len(t, markers, 3)

	// Check that we found the expected feature gates
	expectedGates := map[string]bool{
		"experimental-feature": false,
		"another-feature":      false,
		"function-feature":     false,
	}

	for _, marker := range markers {
		assert.Contains(t, expectedGates, marker.GateName)
		assert.Equal(t, testFile, marker.File)
		assert.Greater(t, marker.Line, 0)
	}
}

func TestExtractFeatureGates(t *testing.T) {
	markers := []FeatureGateMarker{
		{GateName: "feature1", Line: 1, File: "test1.go"},
		{GateName: "feature2", Line: 2, File: "test1.go"},
		{GateName: "feature1", Line: 3, File: "test2.go"}, // Duplicate
		{GateName: "feature3", Line: 4, File: "test2.go"},
	}

	gates := ExtractFeatureGates(markers)
	assert.ElementsMatch(t, []string{"feature1", "feature2", "feature3"}, gates)
}

func TestValidateFeatureGates(t *testing.T) {
	requiredGates := []string{"feature1", "feature2", "feature3"}

	// All required gates enabled
	enabledGates := FeatureGates{
		"feature1": true,
		"feature2": true,
		"feature3": true,
	}

	missing := ValidateFeatureGates(requiredGates, enabledGates)
	assert.Empty(t, missing)

	// Some gates disabled
	enabledGates = FeatureGates{
		"feature1": true,
		"feature2": false,
		"feature3": true,
	}

	missing = ValidateFeatureGates(requiredGates, enabledGates)
	assert.ElementsMatch(t, []string{"feature2"}, missing)

	// Missing gates
	enabledGates = FeatureGates{
		"feature1": true,
		// feature2 missing
		"feature3": true,
	}

	missing = ValidateFeatureGates(requiredGates, enabledGates)
	assert.ElementsMatch(t, []string{"feature2"}, missing)
}

func TestFeatureGateMarkerParser_CommentedMarkers(t *testing.T) {
	parser := NewFeatureGateMarkerParser()

	// Create a temporary file with commented feature gate markers
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	content := `package test

// Example of a feature-gated field:
// Bar is an experimental field that requires the "experimental-bar" feature gate to be enabled
// TODO: When controller-tools supports feature gates (issue #1238), use:
// +kubebuilder:feature-gate=experimental-bar
// +feature-gate experimental-bar
// +optional
// Bar *string ` + "`" + `json:"bar,omitempty"` + "`" + `

type TestSpec struct {
	// foo is an example field
	// +optional
	Foo *string ` + "`" + `json:"foo,omitempty"` + "`" + `
}
`

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Parse the directory
	markers, err := parser.ParseDirectory(tempDir)
	if err != nil {
		t.Fatalf("Failed to parse directory: %v", err)
	}

	// Should not find any feature gate markers since they're commented out
	if len(markers) > 0 {
		t.Errorf("Expected no feature gate markers, but found %d: %v", len(markers), markers)
	}
}

func TestFeatureGateMarkerParser_ActiveMarkers(t *testing.T) {
	parser := NewFeatureGateMarkerParser()

	// Create a temporary file with active feature gate markers
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	content := `package test

type TestSpec struct {
	// foo is an example field
	// +optional
	Foo *string ` + "`" + `json:"foo,omitempty"` + "`" + `

	// Bar is an experimental field that requires the "experimental-bar" feature gate to be enabled
	// +feature-gate experimental-bar
	// +optional
	Bar *string ` + "`" + `json:"bar,omitempty"` + "`" + `
}
`

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Parse the directory
	markers, err := parser.ParseDirectory(tempDir)
	if err != nil {
		t.Fatalf("Failed to parse directory: %v", err)
	}

	// Should find the active feature gate marker
	if len(markers) != 1 {
		t.Errorf("Expected 1 feature gate marker, but found %d: %v", len(markers), markers)
	}

	if markers[0].GateName != "experimental-bar" {
		t.Errorf("Expected gate name 'experimental-bar', but got '%s'", markers[0].GateName)
	}
}
