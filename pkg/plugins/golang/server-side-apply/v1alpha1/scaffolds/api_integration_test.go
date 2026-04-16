/*
Copyright 2026 The Kubernetes Authors.

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

//go:build integration

package scaffolds

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("Server-Side Apply Plugin Scaffolding", func() {
	var kbc *utils.TestContext

	BeforeEach(func() {
		var err error
		kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(kbc.Prepare()).To(Succeed())
	})

	AfterEach(func() {
		kbc.Destroy()
	})

	Context("when scaffolding a project with server-side-apply plugin", func() {
		It("should generate applyconfiguration support and controller scaffolds", func() {
			By("initializing a multigroup project")
			err := kbc.Init(
				"--plugins", "go/v4",
				"--domain", kbc.Domain,
			)
			Expect(err).NotTo(HaveOccurred())

			err = kbc.Edit("--multigroup")
			Expect(err).NotTo(HaveOccurred())

			By("creating an API with server-side-apply plugin")
			err = kbc.CreateAPI(
				"--group", "apps",
				"--version", "v1",
				"--kind", "Application",
				"--plugins", "ssa/v1-alpha",
				"--controller=true",
				"--resource=true",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying Makefile includes applyconfiguration generation")
			makefile, err := os.ReadFile(filepath.Join(kbc.Dir, "Makefile"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(makefile)).To(ContainSubstring("applyconfiguration"),
				"Makefile should include applyconfiguration generation")

			By("verifying groupversion_info.go has applyconfiguration generation marker")
			groupversion, err := os.ReadFile(filepath.Join(kbc.Dir, "api", "apps", "v1", "groupversion_info.go"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(groupversion)).To(ContainSubstring("+kubebuilder:ac:generate=true"),
				"groupversion_info.go should have ac:generate marker")

			By("verifying controller includes Server-Side Apply guidance")
			controller, err := os.ReadFile(filepath.Join(kbc.Dir, "internal", "controller", "apps", "application_controller.go"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(controller)).To(ContainSubstring("Server-Side Apply"),
				"controller should include Server-Side Apply comments")

			By("generating code and verifying applyconfiguration packages are created")
			err = kbc.Make("generate")
			Expect(err).NotTo(HaveOccurred())

			applyConfigDir := filepath.Join(kbc.Dir, "api", "apps", "v1", "applyconfiguration")
			_, err = os.Stat(applyConfigDir)
			Expect(err).NotTo(HaveOccurred(), "applyconfiguration directory should exist after generation")

			By("verifying the scaffolded code passes tests")
			err = kbc.Make("test")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when mixing server-side-apply and standard APIs", func() {
		It("should only generate applyconfigurations for SSA-enabled APIs", func() {
			By("initializing a multigroup project")
			err := kbc.Init(
				"--plugins", "go/v4",
				"--domain", kbc.Domain,
			)
			Expect(err).NotTo(HaveOccurred())

			err = kbc.Edit("--multigroup")
			Expect(err).NotTo(HaveOccurred())

			By("creating a standard API without SSA plugin")
			err = kbc.CreateAPI(
				"--group", "core",
				"--version", "v1",
				"--kind", "Config",
				"--controller=true",
				"--resource=true",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating an API with server-side-apply plugin")
			err = kbc.CreateAPI(
				"--group", "apps",
				"--version", "v1",
				"--kind", "App",
				"--plugins", "ssa/v1-alpha",
				"--controller=true",
				"--resource=true",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying Makefile includes applyconfiguration generation")
			makefile, err := os.ReadFile(filepath.Join(kbc.Dir, "Makefile"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(makefile)).To(ContainSubstring("applyconfiguration"),
				"Makefile should include applyconfiguration generation")

			By("generating code")
			err = kbc.Make("generate")
			Expect(err).NotTo(HaveOccurred())

			By("verifying applyconfiguration was generated only for SSA-enabled API")
			_, err = os.Stat(filepath.Join(kbc.Dir, "api", "apps", "v1", "applyconfiguration"))
			Expect(err).NotTo(HaveOccurred(), "SSA API should have applyconfiguration directory")

			_, err = os.Stat(filepath.Join(kbc.Dir, "api", "core", "v1", "applyconfiguration"))
			Expect(err).To(HaveOccurred(), "standard API should not have applyconfiguration directory")

			By("verifying the scaffolded code passes tests")
			err = kbc.Make("test")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
