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
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("Feature Gates Discovery", func() {
	var kbc *utils.TestContext

	BeforeEach(func() {
		var err error
		kbc, err = utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(kbc.Prepare()).To(Succeed())
	})

	AfterEach(func() {
		By("cleaning up test context")
		kbc.Destroy()
	})

	Context("when parsing testdata project", func() {
		It("should NOT discover feature gate markers from captain_types.go by default", func() {
			By("parsing the testdata project API types")
			parser := machinery.NewFeatureGateMarkerParser()
			markers, err := parser.ParseFile("../../../testdata/project-v4/api/v1/captain_types.go")
			Expect(err).NotTo(HaveOccurred())
			
			By("verifying no feature gates are found in clean testdata")
			// Testdata files are auto-generated and should not contain feature gate markers by default
			Expect(markers).To(BeEmpty(), "Clean testdata should not contain feature gate markers")
		})
	})

	Context("when parsing files with commented fields", func() {
		It("should ignore commented feature gate markers", func() {
			By("creating a temporary file with commented feature gate markers")
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

			By("writing temporary file")
			tmpFile, err := os.CreateTemp("", "featuregate_test_*.go")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tempContent)
			Expect(err).NotTo(HaveOccurred())
			tmpFile.Close()

			By("parsing the temporary file")
			parser := machinery.NewFeatureGateMarkerParser()
			markers, err := parser.ParseFile(tmpFile.Name())
			Expect(err).NotTo(HaveOccurred())

			By("extracting feature gate names")
			featureGates := machinery.ExtractFeatureGates(markers)

			By("verifying only active feature gates are discovered")
			Expect(featureGates).To(ContainElement("active-feature"), "Should discover active feature gate")
			Expect(featureGates).NotTo(ContainElement("commented-feature"), "Should NOT discover commented feature gate")

			GinkgoWriter.Printf("Discovered feature gates: %v\n", featureGates)
		})
	})

	Context("when parsing feature gate strings", func() {
		It("should handle various feature gate parsing scenarios", func() {
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
				By("testing scenario: " + tc.name)
				result, err := machinery.ParseFeatureGates(tc.input)

				if tc.hasError {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(Equal(machinery.FeatureGates(tc.expected)))
				}
			}
		})
	})

	Context("when scaffolding a project with feature gates", func() {
		It("should generate feature gates file correctly", func() {
			By("initializing a project")
			err := kbc.Init(
				"--plugins", "go/v4",
				"--project-version", "3",
				"--domain", kbc.Domain,
			)
			Expect(err).NotTo(HaveOccurred(), "Failed to initialize project")

			By("creating API with feature gate markers")
			err = kbc.CreateAPI(
				"--group", kbc.Group,
				"--version", kbc.Version,
				"--kind", kbc.Kind,
				"--resource", "--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred(), "Failed to create API definition")

			By("adding feature gate markers to the API types")
			apiTypesFile := filepath.Join(kbc.Dir, "api", kbc.Version, strings.ToLower(kbc.Kind)+"_types.go")

			// Add a feature gate marker to the spec
			err = util.InsertCode(
				apiTypesFile,
				"// +optional\n\tFoo *string `json:\"foo,omitempty\"`",
				"\n\n\t// +feature-gate experimental-feature",
			)
			Expect(err).NotTo(HaveOccurred(), "Failed to add feature gate marker")

			// Add a field after the marker
			err = util.InsertCode(
				apiTypesFile,
				"// +feature-gate experimental-feature",
				"\n\tExperimentalField *string `json:\"experimentalField,omitempty\"`",
			)
			Expect(err).NotTo(HaveOccurred(), "Failed to add experimental field")

			By("regenerating the project to discover feature gates")
			err = kbc.CreateAPI(
				"--group", kbc.Group,
				"--version", kbc.Version,
				"--kind", kbc.Kind,
				"--resource", "--controller",
				"--make=false",
				"--force",
			)
			Expect(err).NotTo(HaveOccurred(), "Failed to regenerate project")

			By("verifying feature gates file was generated")
			featureGatesFile := filepath.Join(kbc.Dir, "internal", "featuregates", "featuregates.go")
			Expect(featureGatesFile).To(BeAnExistingFile())

			By("verifying feature gates file contains the discovered feature gate")
			hasContent, err := util.HasFileContentWith(featureGatesFile, "experimental-feature")
			Expect(err).NotTo(HaveOccurred())
			Expect(hasContent).To(BeTrue(), "Feature gates file should contain the discovered feature gate")
		})
	})
})
