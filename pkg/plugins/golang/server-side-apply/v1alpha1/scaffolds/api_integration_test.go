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

	It("should scaffold API with server-side-apply plugin correctly", func() {
		By("initializing a project")
		err := kbc.Init(
			"--plugins", "go/v4",
			"--domain", kbc.Domain,
		)
		Expect(err).NotTo(HaveOccurred())

		By("enabling multigroup")
		err = kbc.Edit("--multigroup")
		Expect(err).NotTo(HaveOccurred())

		By("creating API with server-side-apply plugin")
		err = kbc.CreateAPI(
			"--group", "apps",
			"--version", "v1",
			"--kind", "Application",
			"--plugins", "server-side-apply/v1-alpha",
			"--controller=true",
			"--resource=true",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying Makefile was updated")
		makefile, err := os.ReadFile(filepath.Join(kbc.Dir, "Makefile"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(makefile)).To(ContainSubstring("APPLYCONFIGURATION_PATHS"))
		Expect(string(makefile)).To(ContainSubstring("./api/apps/v1"))

		By("verifying groupversion_info.go has ac:generate marker")
		groupversion, err := os.ReadFile(filepath.Join(kbc.Dir, "api", "apps", "v1", "groupversion_info.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(groupversion)).To(ContainSubstring("+kubebuilder:ac:generate=true"))

		By("verifying controller has Server-Side Apply comments")
		controller, err := os.ReadFile(filepath.Join(kbc.Dir, "internal", "controller", "apps", "application_controller.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(controller)).To(ContainSubstring("Server-Side Apply"))

		By("running make generate")
		err = kbc.Make("generate")
		Expect(err).NotTo(HaveOccurred())

		By("verifying apply configurations were generated")
		applyConfigDir := filepath.Join(kbc.Dir, "api", "apps", "v1", "applyconfiguration")
		_, err = os.Stat(applyConfigDir)
		Expect(err).NotTo(HaveOccurred())

		By("running make test")
		err = kbc.Make("test")
		Expect(err).NotTo(HaveOccurred())
	})

	It("should handle multiple APIs with mixed plugins", func() {
		By("initializing a project")
		err := kbc.Init(
			"--plugins", "go/v4",
			"--domain", kbc.Domain,
		)
		Expect(err).NotTo(HaveOccurred())

		By("enabling multigroup")
		err = kbc.Edit("--multigroup")
		Expect(err).NotTo(HaveOccurred())

		By("creating traditional API")
		err = kbc.CreateAPI(
			"--group", "core",
			"--version", "v1",
			"--kind", "Config",
			"--controller=true",
			"--resource=true",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("creating API with server-side-apply plugin")
		err = kbc.CreateAPI(
			"--group", "apps",
			"--version", "v1",
			"--kind", "App",
			"--plugins", "server-side-apply/v1-alpha",
			"--controller=true",
			"--resource=true",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying Makefile only has plugin API")
		makefile, err := os.ReadFile(filepath.Join(kbc.Dir, "Makefile"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(makefile)).To(ContainSubstring("./api/apps/v1"))
		Expect(string(makefile)).NotTo(ContainSubstring("./api/core/v1"))

		By("running make generate")
		err = kbc.Make("generate")
		Expect(err).NotTo(HaveOccurred())

		By("verifying only plugin API has apply configs")
		_, err = os.Stat(filepath.Join(kbc.Dir, "api", "apps", "v1", "applyconfiguration"))
		Expect(err).NotTo(HaveOccurred())

		_, err = os.Stat(filepath.Join(kbc.Dir, "api", "core", "v1", "applyconfiguration"))
		Expect(err).To(HaveOccurred()) // Should NOT exist

		By("running make test")
		err = kbc.Make("test")
		Expect(err).NotTo(HaveOccurred())
	})
})
