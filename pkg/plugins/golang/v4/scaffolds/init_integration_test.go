//go:build integration

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

package scaffolds

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("Init Scaffolding Integration Test", func() {
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

	Context("cluster-scoped init (default)", func() {
		It("should scaffold cluster-scoped configuration", func() {
			By("initializing a cluster-scoped project")
			err := kbc.Init(
				"--plugins", "go/v4",
				"--project-version", "3",
				"--domain", kbc.Domain,
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying PROJECT file does not have namespaced flag")
			projectFile := filepath.Join(kbc.Dir, "PROJECT")
			content, err := os.ReadFile(projectFile)
			Expect(err).NotTo(HaveOccurred())
			projectContent := string(content)
			// Check root level doesn't have namespaced: true
			beforeResources := strings.Split(projectContent, "resources:")[0]
			if strings.Contains(beforeResources, "namespaced:") {
				Expect(beforeResources).NotTo(ContainSubstring("namespaced: true"))
			}

			By("verifying RBAC is cluster-scoped")
			roleFile := filepath.Join(kbc.Dir, "config", "rbac", "role.yaml")
			content, err = os.ReadFile(roleFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("kind: ClusterRole"))

			roleBindingFile := filepath.Join(kbc.Dir, "config", "rbac", "role_binding.yaml")
			content, err = os.ReadFile(roleBindingFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("kind: ClusterRoleBinding"))

			By("verifying manager.yaml does NOT have WATCH_NAMESPACE")
			managerFile := filepath.Join(kbc.Dir, "config", "manager", "manager.yaml")
			content, err = os.ReadFile(managerFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).NotTo(ContainSubstring("WATCH_NAMESPACE"))

			By("verifying cmd/main.go does NOT have namespace helper functions")
			mainFile := filepath.Join(kbc.Dir, "cmd", "main.go")
			content, err = os.ReadFile(mainFile)
			Expect(err).NotTo(HaveOccurred())
			mainContent := string(content)
			Expect(mainContent).NotTo(ContainSubstring("func getWatchNamespace()"))
			Expect(mainContent).NotTo(ContainSubstring("func setupCacheNamespaces"))
		})
	})

	Context("namespace-scoped init (--namespaced)", func() {
		It("should scaffold namespace-scoped configuration", func() {
			By("initializing a namespace-scoped project")
			err := kbc.Init(
				"--plugins", "go/v4",
				"--project-version", "3",
				"--domain", kbc.Domain,
				"--namespaced",
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying PROJECT file has namespaced: true")
			projectFile := filepath.Join(kbc.Dir, "PROJECT")
			content, err := os.ReadFile(projectFile)
			Expect(err).NotTo(HaveOccurred())
			projectContent := string(content)
			// Check root level has namespaced: true
			beforeResources := strings.Split(projectContent, "resources:")[0]
			Expect(beforeResources).To(ContainSubstring("namespaced: true"))

			By("verifying RBAC is namespace-scoped")
			roleFile := filepath.Join(kbc.Dir, "config", "rbac", "role.yaml")
			content, err = os.ReadFile(roleFile)
			Expect(err).NotTo(HaveOccurred())
			roleContent := string(content)
			Expect(roleContent).To(ContainSubstring("kind: Role"))
			Expect(roleContent).NotTo(ContainSubstring("kind: ClusterRole"))

			roleBindingFile := filepath.Join(kbc.Dir, "config", "rbac", "role_binding.yaml")
			content, err = os.ReadFile(roleBindingFile)
			Expect(err).NotTo(HaveOccurred())
			bindingContent := string(content)
			Expect(bindingContent).To(ContainSubstring("kind: RoleBinding"))
			Expect(bindingContent).NotTo(ContainSubstring("kind: ClusterRoleBinding"))

			By("verifying manager.yaml has WATCH_NAMESPACE environment variable")
			managerFile := filepath.Join(kbc.Dir, "config", "manager", "manager.yaml")
			content, err = os.ReadFile(managerFile)
			Expect(err).NotTo(HaveOccurred())
			managerContent := string(content)
			Expect(managerContent).To(ContainSubstring("- name: WATCH_NAMESPACE"))
			Expect(managerContent).To(ContainSubstring("fieldRef:"))
			Expect(managerContent).To(ContainSubstring("fieldPath: metadata.namespace"))

			By("verifying cmd/main.go has getWatchNamespace helper function")
			mainFile := filepath.Join(kbc.Dir, "cmd", "main.go")
			content, err = os.ReadFile(mainFile)
			Expect(err).NotTo(HaveOccurred())
			mainContent := string(content)
			Expect(mainContent).To(ContainSubstring("func getWatchNamespace()"))
			Expect(mainContent).To(ContainSubstring("WATCH_NAMESPACE"))

			By("verifying cmd/main.go has setupCacheNamespaces helper function")
			Expect(mainContent).To(ContainSubstring("func setupCacheNamespaces"))
			Expect(mainContent).To(ContainSubstring("DefaultNamespaces"))
			Expect(mainContent).To(ContainSubstring("cache.Config"))

			By("verifying cmd/main.go uses helper functions in main()")
			Expect(mainContent).To(ContainSubstring("watchNamespace, err := getWatchNamespace()"))
			Expect(mainContent).To(ContainSubstring("mgrOptions.Cache = setupCacheNamespaces(watchNamespace)"))

			By("creating an API to verify controller scaffolding")
			err = kbc.CreateAPI(
				"--group", kbc.Group,
				"--version", kbc.Version,
				"--kind", kbc.Kind,
				"--resource",
				"--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying controller has namespace parameter in RBAC markers")
			controllerFile := filepath.Join(kbc.Dir, "internal", "controller",
				strings.ToLower(kbc.Kind)+"_controller.go")
			content, err = os.ReadFile(controllerFile)
			Expect(err).NotTo(HaveOccurred())
			controllerContent := string(content)

			expectedNamespace := "e2e-" + kbc.TestSuffix + "-system"
			Expect(controllerContent).To(ContainSubstring("namespace="+expectedNamespace),
				"Controller RBAC markers should include namespace parameter")

			By("verifying admin/editor/viewer roles are namespace-scoped")
			editorFile := filepath.Join(kbc.Dir, "config", "rbac",
				strings.ToLower(kbc.Kind)+"_editor_role.yaml")
			content, err = os.ReadFile(editorFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("kind: Role"))
			Expect(string(content)).NotTo(ContainSubstring("kind: ClusterRole"))
		})
	})
})
