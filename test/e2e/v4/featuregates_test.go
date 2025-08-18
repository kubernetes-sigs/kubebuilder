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

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("Feature Gates", func() {
	Context("project with feature gates", func() {
		var (
			kbc *utils.TestContext
		)

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc).NotTo(BeNil())

			kbc.Domain = "example.com"
			kbc.Group = "crew"
			kbc.Version = "v1"
			kbc.Kind = "Captain"

			By("preparing the test environment")
			Expect(kbc.Prepare()).To(Succeed())
		})

		AfterEach(func() {
			By("destroying directory")
			kbc.Destroy()
		})

		It("should scaffold a project with feature gates", func() {
			By("initializing a project")
			err := kbc.Init(
				"--domain", kbc.Domain,
				"--with-feature-gates",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating API with feature gates")
			err = kbc.CreateAPI(
				"--group", kbc.Group,
				"--version", kbc.Version,
				"--kind", kbc.Kind,
				"--resource", "--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying feature gates file was generated")
			featureGatesFile := filepath.Join(kbc.Dir, "internal", "featuregates", "featuregates.go")
			Expect(featureGatesFile).To(BeAnExistingFile())

			By("verifying main.go has feature gates flag")
			mainFile := filepath.Join(kbc.Dir, "cmd", "main.go")
			Expect(mainFile).To(BeAnExistingFile())
			mainContent, err := os.ReadFile(mainFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(mainContent)).To(ContainSubstring("--feature-gates"))
			Expect(string(mainContent)).To(ContainSubstring("featuregates"))

			By("verifying API types have feature gate example")
			typesFile := filepath.Join(kbc.Dir, "api", kbc.Version, strings.ToLower(kbc.Kind)+"_types.go")
			Expect(typesFile).To(BeAnExistingFile())
			typesContent, err := os.ReadFile(typesFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(typesContent)).To(ContainSubstring("+feature-gate"))
		})

		It("should NOT scaffold feature gates by default", func() {
			By("initializing a project without feature gates flag")
			err := kbc.Init(
				"--domain", kbc.Domain,
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating API without feature gates flag")
			err = kbc.CreateAPI(
				"--group", kbc.Group,
				"--version", kbc.Version,
				"--kind", kbc.Kind,
				"--resource", "--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying feature gates file was NOT generated")
			featureGatesFile := filepath.Join(kbc.Dir, "internal", "featuregates", "featuregates.go")
			Expect(featureGatesFile).NotTo(BeAnExistingFile())

			By("verifying main.go does NOT have feature gates flag")
			mainFile := filepath.Join(kbc.Dir, "cmd", "main.go")
			Expect(mainFile).To(BeAnExistingFile())
			mainContent, err := os.ReadFile(mainFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(mainContent)).NotTo(ContainSubstring("--feature-gates"))
			Expect(string(mainContent)).NotTo(ContainSubstring("featuregates"))

			By("verifying API types do NOT have feature gate example")
			typesFile := filepath.Join(kbc.Dir, "api", kbc.Version, strings.ToLower(kbc.Kind)+"_types.go")
			Expect(typesFile).To(BeAnExistingFile())
			typesContent, err := os.ReadFile(typesFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(typesContent)).NotTo(ContainSubstring("+feature-gate"))
		})

		It("should discover feature gates from API types", func() {
			By("initializing a project")
			err := kbc.Init(
				"--domain", kbc.Domain,
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating API")
			err = kbc.CreateAPI(
				"--group", kbc.Group,
				"--version", kbc.Version,
				"--kind", kbc.Kind,
				"--resource", "--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("adding custom feature gates to the API types")
			typesFile := filepath.Join(kbc.Dir, "api", kbc.Version, strings.ToLower(kbc.Kind)+"_types.go")

			// Add a new field with feature gate using the utility function
			newField := `
	// Experimental field that requires the "experimental-field" feature gate
	// +feature-gate experimental-field
	// +optional
	ExperimentalField *string ` + "`" + `json:"experimentalField,omitempty"` + "`"

			err = pluginutil.InsertCode(
				typesFile,
				"Foo *string `json:\"foo,omitempty\"`",
				newField,
			)
			Expect(err).NotTo(HaveOccurred())

			By("regenerating the project to discover new feature gates")
			err = kbc.CreateAPI(
				"--group", kbc.Group,
				"--version", kbc.Version,
				"--kind", kbc.Kind,
				"--resource", "--controller",
				"--force",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying the new feature gate was discovered")
			featureGatesFile := filepath.Join(kbc.Dir, "internal", "featuregates", "featuregates.go")
			Expect(featureGatesFile).To(BeAnExistingFile())
			featureGatesContent, err := os.ReadFile(featureGatesFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(featureGatesContent)).To(ContainSubstring("experimental-field"))
		})

		It("should build project with feature gates", func() {
			By("initializing a project with feature gates")
			err := kbc.Init(
				"--domain", kbc.Domain,
				"--with-feature-gates",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating API")
			err = kbc.CreateAPI(
				"--group", kbc.Group,
				"--version", kbc.Version,
				"--kind", kbc.Kind,
				"--resource", "--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("building the project - this should succeed with feature gates")
			err = kbc.Make("build")
			Expect(err).NotTo(HaveOccurred())

			By("verifying the binary was created")
			binaryPath := filepath.Join(kbc.Dir, "bin", "manager")
			Expect(binaryPath).To(BeAnExistingFile())
		})

		It("should generate manifests with feature gates", func() {
			By("initializing a project with feature gates")
			err := kbc.Init(
				"--domain", kbc.Domain,
				"--with-feature-gates",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating API")
			err = kbc.CreateAPI(
				"--group", kbc.Group,
				"--version", kbc.Version,
				"--kind", kbc.Kind,
				"--resource", "--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("generating manifests")
			err = kbc.Make("manifests")
			Expect(err).NotTo(HaveOccurred())

			By("verifying CRDs were generated")
			crdFile := filepath.Join(kbc.Dir, "config", "crd", "bases",
				strings.ToLower(kbc.Group)+"."+kbc.Domain+"_captains.yaml")
			Expect(crdFile).To(BeAnExistingFile())
		})
	})
})
