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

var _ = Describe("Edit Scaffolding Integration Test", func() {
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

	It("should handle scope transitions comprehensively", func() {
		roleFile := filepath.Join(kbc.Dir, "config", "rbac", "role.yaml")
		roleBindingFile := filepath.Join(kbc.Dir, "config", "rbac", "role_binding.yaml")
		managerFile := filepath.Join(kbc.Dir, "config", "manager", "manager.yaml")
		projectFile := filepath.Join(kbc.Dir, "PROJECT")

		// ========== Part 1: Cluster-scoped → Namespaced (without --force) ==========
		By("initializing a cluster-scoped project")
		err := kbc.Init(
			"--plugins", "go/v4",
			"--project-version", "3",
			"--domain", kbc.Domain,
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying initial state is cluster-scoped")
		content, err := os.ReadFile(roleFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("kind: ClusterRole"))

		By("enabling namespaced layout without --force")
		err = kbc.Edit("--namespaced")
		Expect(err).NotTo(HaveOccurred())

		By("verifying PROJECT file was updated")
		content, err = os.ReadFile(projectFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(strings.Split(string(content), "resources:")[0]).To(ContainSubstring("namespaced: true"))

		By("verifying RBAC was changed to namespace-scoped")
		content, err = os.ReadFile(roleFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("kind: Role"))
		Expect(string(content)).NotTo(ContainSubstring("kind: ClusterRole"))

		content, err = os.ReadFile(roleBindingFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("kind: RoleBinding"))

		By("verifying manager.yaml was NOT updated (no --force)")
		content, err = os.ReadFile(managerFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).NotTo(ContainSubstring("WATCH_NAMESPACE"),
			"manager.yaml should not be updated without --force flag")

		// ========== Part 2: Revert to cluster, then switch with --force ==========
		By("reverting to cluster-scoped first")
		err = kbc.Edit("--namespaced=false")
		Expect(err).NotTo(HaveOccurred())

		By("re-enabling namespaced layout with --force to update manager.yaml")
		err = kbc.Edit("--namespaced", "--force")
		Expect(err).NotTo(HaveOccurred())

		By("verifying manager.yaml was updated with WATCH_NAMESPACE")
		content, err = os.ReadFile(managerFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("- name: WATCH_NAMESPACE"))
		Expect(string(content)).To(ContainSubstring("fieldRef:"))
		Expect(string(content)).To(ContainSubstring("fieldPath: metadata.namespace"))

		// ========== Part 3: Create API and verify namespace RBAC markers ==========
		By("creating an API in namespaced mode")
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
		expectedNamespace := "e2e-" + kbc.TestSuffix + "-system"
		Expect(string(content)).To(ContainSubstring("namespace="+expectedNamespace),
			"Controller RBAC markers should include namespace parameter")

		By("verifying CRD admin/editor/viewer roles are namespace-scoped")
		adminRoleFile := filepath.Join(kbc.Dir, "config", "rbac",
			strings.ToLower(kbc.Kind)+"_admin_role.yaml")
		editorRoleFile := filepath.Join(kbc.Dir, "config", "rbac",
			strings.ToLower(kbc.Kind)+"_editor_role.yaml")
		viewerRoleFile := filepath.Join(kbc.Dir, "config", "rbac",
			strings.ToLower(kbc.Kind)+"_viewer_role.yaml")

		content, err = os.ReadFile(adminRoleFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("kind: Role"))

		content, err = os.ReadFile(editorRoleFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("kind: Role"))

		content, err = os.ReadFile(viewerRoleFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("kind: Role"))

		// ========== Part 4: Switch to cluster-scoped and verify all roles updated ==========
		By("switching to cluster-scoped with --force")
		err = kbc.Edit("--namespaced=false", "--force")
		Expect(err).NotTo(HaveOccurred())

		By("verifying PROJECT file was updated")
		content, err = os.ReadFile(projectFile)
		Expect(err).NotTo(HaveOccurred())
		projectContent := string(content)
		beforeResources := strings.Split(projectContent, "resources:")[0]
		if strings.Contains(beforeResources, "namespaced:") {
			Expect(beforeResources).To(ContainSubstring("namespaced: false"))
		}

		By("verifying manager RBAC was changed to cluster-scoped")
		content, err = os.ReadFile(roleFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("kind: ClusterRole"))

		content, err = os.ReadFile(roleBindingFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("kind: ClusterRoleBinding"))

		By("verifying manager.yaml was updated (WATCH_NAMESPACE removed)")
		content, err = os.ReadFile(managerFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).NotTo(ContainSubstring("WATCH_NAMESPACE"))

		By("verifying CRD roles were updated to cluster-scoped")
		content, err = os.ReadFile(adminRoleFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("kind: ClusterRole"))

		content, err = os.ReadFile(editorRoleFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("kind: ClusterRole"))

		content, err = os.ReadFile(viewerRoleFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("kind: ClusterRole"))

		// ========== Part 5: Switch back to namespaced and verify again ==========
		By("switching back to namespace-scoped with --force")
		err = kbc.Edit("--namespaced", "--force")
		Expect(err).NotTo(HaveOccurred())

		By("verifying all roles reverted to namespace-scoped")
		content, err = os.ReadFile(roleFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("kind: Role"))

		content, err = os.ReadFile(adminRoleFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("kind: Role"))

		content, err = os.ReadFile(editorRoleFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("kind: Role"))

		content, err = os.ReadFile(viewerRoleFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("kind: Role"))

		By("verifying manager.yaml has WATCH_NAMESPACE again")
		content, err = os.ReadFile(managerFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("WATCH_NAMESPACE"))
	})
})
