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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

func TestFeatureGates(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Feature Gates Suite")
}

var _ = Describe("Feature Gates", func() {
	Context("project with feature gates", func() {
		var (
			ctx *utils.TestContext
		)

		BeforeEach(func() {
			var err error
			ctx, err = utils.NewTestContext("go")
			Expect(err).NotTo(HaveOccurred())
			Expect(ctx).NotTo(BeNil())

			// Set up test context
			ctx.Domain = "example.com"
			ctx.Group = "crew"
			ctx.Version = "v1"
			ctx.Kind = "Captain"
			ctx.Resources = "captains"

			// Prepare the test environment
			err = ctx.Prepare()
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			ctx.Destroy()
		})

		It("should scaffold a project with feature gates", func() {
			// Initialize the project
			err := ctx.Init()
			Expect(err).NotTo(HaveOccurred())

			// Create API with feature gates
			err = ctx.CreateAPI(
				"--group", ctx.Group,
				"--version", ctx.Version,
				"--kind", ctx.Kind,
				"--resource", "--controller",
			)
			Expect(err).NotTo(HaveOccurred())

			// Verify feature gates file was generated
			featureGatesFile := filepath.Join(ctx.Dir, "internal", "featuregates", "featuregates.go")
			Expect(featureGatesFile).To(BeAnExistingFile())

			// Verify main.go has feature gates flag
			mainFile := filepath.Join(ctx.Dir, "cmd", "main.go")
			Expect(mainFile).To(BeAnExistingFile())
			mainContent, err := os.ReadFile(mainFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(mainContent)).To(ContainSubstring("--feature-gates"))
			Expect(string(mainContent)).To(ContainSubstring("featuregates"))

			// Verify API types have feature gate example
			typesFile := filepath.Join(ctx.Dir, "api", ctx.Version, ctx.Kind+"_types.go")
			Expect(typesFile).To(BeAnExistingFile())
			typesContent, err := os.ReadFile(typesFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(typesContent)).To(ContainSubstring("+feature-gate"))
		})

		It("should discover feature gates from API types", func() {
			// Initialize the project
			err := ctx.Init()
			Expect(err).NotTo(HaveOccurred())

			// Create API
			err = ctx.CreateAPI(
				"--group", ctx.Group,
				"--version", ctx.Version,
				"--kind", ctx.Kind,
				"--resource", "--controller",
			)
			Expect(err).NotTo(HaveOccurred())

			// Add custom feature gates to the API types
			typesFile := filepath.Join(ctx.Dir, "api", ctx.Version, ctx.Kind+"_types.go")
			typesContent, err := os.ReadFile(typesFile)
			Expect(err).NotTo(HaveOccurred())

			// Add a new field with feature gate
			newField := `
	// Experimental field that requires the "experimental-field" feature gate
	// +feature-gate experimental-field
	// +optional
	ExperimentalField *string ` + "`" + `json:"experimentalField,omitempty"` + "`" + `
`
			// Insert the new field in the Spec struct
			updatedContent := strings.Replace(string(typesContent),
				"// Bar *string ` + \"`\" + `json:\"bar,omitempty\"` + \"`\" + `",
				"// Bar *string ` + \"`\" + `json:\"bar,omitempty\"` + \"`\" + `\n"+newField,
				1)

			err = os.WriteFile(typesFile, []byte(updatedContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			// Regenerate the project to discover new feature gates
			err = ctx.CreateAPI(
				"--group", ctx.Group,
				"--version", ctx.Version,
				"--kind", ctx.Kind,
				"--resource", "--controller",
				"--force",
			)
			Expect(err).NotTo(HaveOccurred())

			// Verify the new feature gate was discovered
			featureGatesFile := filepath.Join(ctx.Dir, "internal", "featuregates", "featuregates.go")
			Expect(featureGatesFile).To(BeAnExistingFile())
			featureGatesContent, err := os.ReadFile(featureGatesFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(featureGatesContent)).To(ContainSubstring("experimental-field"))
		})

		It("should build project with feature gates", func() {
			// Initialize the project
			err := ctx.Init()
			Expect(err).NotTo(HaveOccurred())

			// Create API
			err = ctx.CreateAPI(
				"--group", ctx.Group,
				"--version", ctx.Version,
				"--kind", ctx.Kind,
				"--resource", "--controller",
			)
			Expect(err).NotTo(HaveOccurred())

			// Build the project - this should succeed with feature gates
			err = ctx.Make("build")
			Expect(err).NotTo(HaveOccurred())

			// Verify the binary was created
			binaryPath := filepath.Join(ctx.Dir, "bin", "manager")
			Expect(binaryPath).To(BeAnExistingFile())
		})

		It("should generate manifests with feature gates", func() {
			// Initialize the project
			err := ctx.Init()
			Expect(err).NotTo(HaveOccurred())

			// Create API
			err = ctx.CreateAPI(
				"--group", ctx.Group,
				"--version", ctx.Version,
				"--kind", ctx.Kind,
				"--resource", "--controller",
			)
			Expect(err).NotTo(HaveOccurred())

			// Generate manifests
			err = ctx.Make("manifests")
			Expect(err).NotTo(HaveOccurred())

			// Verify CRDs were generated
			crdFile := filepath.Join(ctx.Dir, "config", "crd", "bases",
				strings.ToLower(ctx.Group)+"_"+ctx.Version+"_"+strings.ToLower(ctx.Kind)+".yaml")
			Expect(crdFile).To(BeAnExistingFile())
		})
	})
})
