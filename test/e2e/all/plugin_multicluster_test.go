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

package all

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

// Test specs for multicluster-runtime/v1-alpha plugin
var _ = Describe("kubebuilder", func() {
	Context("plugin multicluster-runtime/v1-alpha", func() {
		var kbc *utils.TestContext

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())
		})

		AfterEach(func() {
			By("removing controller image and working dir")
			kbc.Destroy()
		})

		It("should scaffold and build a project with the kubeconfig provider", func() {
			By("initialising a go/v4 + multicluster-runtime/v1-alpha project")
			Expect(kbc.Init(
				"--plugins", "go/v4,multicluster-runtime/v1-alpha",
				"--domain", kbc.Domain,
			)).To(Succeed())

			By("creating a multicluster-aware API and controller")
			Expect(kbc.CreateAPI(
				"--plugins", "go/v4,multicluster-runtime/v1-alpha",
				"--group", kbc.Group,
				"--version", kbc.Version,
				"--kind", kbc.Kind,
				"--controller",
				"--resource",
				"--make=false",
			)).To(Succeed())

			By("verifying cmd/main.go uses mcmanager.New")
			mainPath := filepath.Join(kbc.Dir, "cmd", "main.go")
			mainBytes, err := os.ReadFile(mainPath) //nolint:gosec
			Expect(err).NotTo(HaveOccurred())
			Expect(string(mainBytes)).To(ContainSubstring("mcmanager.New"))
			Expect(string(mainBytes)).To(ContainSubstring("sigs.k8s.io/multicluster-runtime"))

			By("verifying the controller uses mcreconcile.Request")
			controllerGlob := filepath.Join(kbc.Dir, "internal", "controller", "*.go")
			matches, err := filepath.Glob(controllerGlob)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).NotTo(BeEmpty())
			controllerBytes, err := os.ReadFile(matches[0]) //nolint:gosec
			Expect(err).NotTo(HaveOccurred())
			Expect(string(controllerBytes)).To(ContainSubstring("mcreconcile.Request"))
			Expect(string(controllerBytes)).To(ContainSubstring("mcbuilder.ControllerManagedBy"))

			By("verifying the project builds without errors")
			Expect(kbc.Make("build")).To(Succeed())
		})

		It("should scaffold and build a project with the namespace provider", func() {
			By("initialising a go/v4 + multicluster-runtime/v1-alpha project with namespace provider")
			Expect(kbc.Init(
				"--plugins", "go/v4,multicluster-runtime/v1-alpha",
				"--domain", kbc.Domain,
				"--provider", "namespace",
			)).To(Succeed())

			By("verifying cmd/main.go uses the namespace provider")
			mainPath := filepath.Join(kbc.Dir, "cmd", "main.go")
			mainBytes, err := os.ReadFile(mainPath) //nolint:gosec
			Expect(err).NotTo(HaveOccurred())
			Expect(string(mainBytes)).To(ContainSubstring("nsprovider"))

			By("verifying the project builds without errors")
			Expect(kbc.Make("build")).To(Succeed())
		})

		It("should allow switching providers with kubebuilder edit", func() {
			By("initialising a project with the kubeconfig provider")
			Expect(kbc.Init(
				"--plugins", "go/v4,multicluster-runtime/v1-alpha",
				"--domain", kbc.Domain,
				"--provider", "kubeconfig",
			)).To(Succeed())

			By("switching to the namespace provider via kubebuilder edit")
			Expect(kbc.Edit(
				"--plugins", "multicluster-runtime/v1-alpha",
				"--provider", "namespace",
			)).To(Succeed())

			By("verifying cmd/main.go now uses the namespace provider")
			mainPath := filepath.Join(kbc.Dir, "cmd", "main.go")
			mainBytes, err := os.ReadFile(mainPath) //nolint:gosec
			Expect(err).NotTo(HaveOccurred())
			Expect(string(mainBytes)).To(ContainSubstring("nsprovider"))
			Expect(string(mainBytes)).NotTo(ContainSubstring("kubeconfigprovider"))

			By("verifying the project still builds after switching providers")
			Expect(kbc.Make("build")).To(Succeed())
		})

		It("should scaffold and build a project with the cluster-api provider", func() {
			By("initialising a go/v4 + multicluster-runtime/v1-alpha project with cluster-api provider")
			Expect(kbc.Init(
				"--plugins", "go/v4,multicluster-runtime/v1-alpha",
				"--domain", kbc.Domain,
				"--provider", "cluster-api",
			)).To(Succeed())

			By("verifying cmd/main.go uses the cluster-api provider")
			mainPath := filepath.Join(kbc.Dir, "cmd", "main.go")
			mainBytes, err := os.ReadFile(mainPath) //nolint:gosec
			Expect(err).NotTo(HaveOccurred())
			Expect(string(mainBytes)).To(ContainSubstring("capiprovider"))

			By("verifying the project builds without errors")
			Expect(kbc.Make("build")).To(Succeed())
		})

		It("should scaffold and build a project with the file provider", func() {
			By("initialising a go/v4 + multicluster-runtime/v1-alpha project with file provider")
			Expect(kbc.Init(
				"--plugins", "go/v4,multicluster-runtime/v1-alpha",
				"--domain", kbc.Domain,
				"--provider", "file",
			)).To(Succeed())

			By("verifying cmd/main.go uses the file provider")
			mainPath := filepath.Join(kbc.Dir, "cmd", "main.go")
			mainBytes, err := os.ReadFile(mainPath) //nolint:gosec
			Expect(err).NotTo(HaveOccurred())
			Expect(string(mainBytes)).To(ContainSubstring("fileprovider"))

			By("verifying the project builds without errors")
			Expect(kbc.Make("build")).To(Succeed())
		})
	})
})
