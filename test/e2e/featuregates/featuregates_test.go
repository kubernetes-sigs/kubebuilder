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

package featuregates

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

func TestFeatureGates(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Feature Gates Unit Tests Suite")
}

var _ = Describe("Feature Gate Parsing", func() {
	It("should parse valid boolean feature gate values", func() {
		gates, err := machinery.ParseFeatureGates("test-gate=true,another-gate=false")
		Expect(err).NotTo(HaveOccurred())
		Expect(gates).To(HaveKeyWithValue("test-gate", true))
		Expect(gates).To(HaveKeyWithValue("another-gate", false))
	})

	It("should reject invalid feature gate values", func() {
		_, err := machinery.ParseFeatureGates("test-gate=maybe")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("must be 'true' or 'false'"))
	})

	It("should handle empty input", func() {
		gates, err := machinery.ParseFeatureGates("")
		Expect(err).NotTo(HaveOccurred())
		Expect(gates).To(BeEmpty())
	})

	It("should reject malformed input", func() {
		// Test various malformed inputs
		invalidInputs := []string{
			"InvalidFormat",      // CamelCase not allowed
			"invalid_underscore", // Underscores not allowed
			"invalid.dot",        // Dots not allowed
			"invalid space",      // Spaces not allowed
			"=true",              // Empty name
			"feature=",           // Empty value
			"feature=maybe",      // Invalid boolean value
		}

		for _, input := range invalidInputs {
			_, err := machinery.ParseFeatureGates(input)
			Expect(err).To(HaveOccurred(), "Expected error for input: %s", input)
		}
	})
})

var _ = Describe("Feature Gate Marker Discovery", func() {
	var parser *machinery.FeatureGateMarkerParser

	BeforeEach(func() {
		parser = machinery.NewFeatureGateMarkerParser()
	})

	It("should discover feature gate markers in Go source files", func() {
		// Create a temporary directory and file for testing
		tempDir := GinkgoT().TempDir()
		testFile := tempDir + "/test_types.go"

		source := `package v1

type TestSpec struct {
	// Regular field without feature gate
	Name string ` + "`json:\"name\"`" + `

	// +feature-gate experimental-field
	// +optional
	ExperimentalField *string ` + "`json:\"experimentalField,omitempty\"`" + `

	// +feature-gate another-experimental
	// This field requires the another-experimental gate to be enabled
	// +optional  
	AnotherField *int ` + "`json:\"anotherField,omitempty\"`" + `
}
`
		err := os.WriteFile(testFile, []byte(source), 0644)
		Expect(err).NotTo(HaveOccurred())

		markers, err := parser.ParseFile(testFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(markers).To(HaveLen(2))

		gates := machinery.ExtractFeatureGates(markers)
		Expect(gates).To(ContainElements("experimental-field", "another-experimental"))
	})

	It("should handle source code without feature gates", func() {
		// Create a temporary directory and file for testing
		tempDir := GinkgoT().TempDir()
		testFile := tempDir + "/test_types.go"

		source := `package v1

type TestSpec struct {
	Name string ` + "`json:\"name\"`" + `
	Count int ` + "`json:\"count\"`" + `
}
`
		err := os.WriteFile(testFile, []byte(source), 0644)
		Expect(err).NotTo(HaveOccurred())

		markers, err := parser.ParseFile(testFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(markers).To(BeEmpty())

		gates := machinery.ExtractFeatureGates(markers)
		Expect(gates).To(BeEmpty())
	})

	It("should ignore commented out feature gate markers", func() {
		// Create a temporary directory and file for testing
		tempDir := GinkgoT().TempDir()
		testFile := tempDir + "/test_types.go"

		source := `package v1

type TestSpec struct {
	// This is a comment about feature gates but not a marker
	// +feature-gate experimental-field
	ActiveField string ` + "`json:\"activeField\"`" + `

	// // +feature-gate commented-out-gate
	// This feature gate marker is commented out
	InactiveField string ` + "`json:\"inactiveField\"`" + `
}
`
		err := os.WriteFile(testFile, []byte(source), 0644)
		Expect(err).NotTo(HaveOccurred())

		markers, err := parser.ParseFile(testFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(markers).To(HaveLen(1))

		gates := machinery.ExtractFeatureGates(markers)
		Expect(gates).To(ContainElement("experimental-field"))
		Expect(gates).NotTo(ContainElement("commented-out-gate"))
	})
})

var _ = Describe("Feature Gate Validation", func() {
	It("should validate feature gate names", func() {
		validNames := []string{
			"valid-name",
			"another-valid-name",
			"alpha-feature",
			"beta-api",
			"feature1",
			"a",
			"experimental-feature-v2",
		}

		for _, name := range validNames {
			gates, err := machinery.ParseFeatureGates(name + "=true")
			Expect(err).NotTo(HaveOccurred(), "Expected %s to be valid", name)
			Expect(gates).To(HaveKeyWithValue(name, true))
		}
	})

	It("should reject invalid feature gate names", func() {
		invalidNames := []string{
			"InvalidCamelCase",   // CamelCase not allowed
			"invalid_underscore", // Underscores not allowed
			"invalid.dot",        // Dots not allowed
			"invalid space",      // Spaces not allowed
			"UPPERCASE",          // Uppercase not allowed
			"feature-",           // Cannot end with hyphen
			"-feature",           // Cannot start with hyphen
			"feature--name",      // Double hyphens not allowed
			"123-feature",        // Cannot start with number
		}

		for _, name := range invalidNames {
			_, err := machinery.ParseFeatureGates(name + "=true")
			Expect(err).To(HaveOccurred(), "Expected error for name: %s", name)
			Expect(err.Error()).To(ContainSubstring("invalid feature gate name"))
		}
	})

	It("should reject truly invalid formats", func() {
		invalidInputs := []string{
			"=true",                                // Empty name
			"feature=maybe",                        // Invalid boolean value
			"feature=",                             // Empty value
			"valid-feature=TRUE,invalid_name=true", // Mix of valid and invalid
		}

		for _, input := range invalidInputs {
			_, err := machinery.ParseFeatureGates(input)
			Expect(err).To(HaveOccurred(), "Expected error for input: %s", input)
		}
	})
})
