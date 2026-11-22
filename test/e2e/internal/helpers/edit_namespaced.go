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

package helpers

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	"github.com/spf13/afero"

	yamlstore "sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	_ "sigs.k8s.io/kubebuilder/v4/pkg/config/v3" // register the PROJECT version 3 config
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

// EditPreservesNamespacedScope drives the real CLI to verify that `kubebuilder edit`
func EditPreservesNamespacedScope(kbc *utils.TestContext) {
	initingNamespacedProject(kbc)
	expectProjectScope(kbc, false)
	expectNamespacedRBAC(kbc)

	By("running `edit --multigroup` without --namespaced")
	Expect(kbc.Edit("--multigroup")).To(Succeed())
	expectProjectScope(kbc, true)
	expectNamespacedRBAC(kbc)

	By("running `edit --license apache2 --owner` without --namespaced or --multigroup")
	Expect(kbc.Edit("--license", "apache2", "--owner", "Edit Scope Test Owner")).To(Succeed())
	expectProjectScope(kbc, true)
	expectNamespacedRBAC(kbc)
	expectBoilerplateOwner(kbc, "Edit Scope Test Owner")

	By("running `alpha generate` on a project with both multigroup and namespaced enabled")
	Expect(kbc.Regenerate()).To(Succeed())
	expectProjectScope(kbc, true)
	expectNamespacedRBAC(kbc)
	expectBoilerplateOwner(kbc, "Edit Scope Test Owner")

	By("checking the regenerated manager watches a namespace")
	manager, err := os.ReadFile(filepath.Join(kbc.Dir, "config", "manager", "manager.yaml"))
	Expect(err).NotTo(HaveOccurred())
	Expect(string(manager)).To(ContainSubstring("WATCH_NAMESPACE"))
}

// expectProjectScope loads the PROJECT file from disk and checks it is namespaced with the given multigroup setting.
func expectProjectScope(kbc *utils.TestContext, multigroup bool) {
	By("loading the saved PROJECT file")
	store := yamlstore.New(machinery.Filesystem{FS: afero.NewBasePathFs(afero.NewOsFs(), kbc.Dir)})
	ExpectWithOffset(1, store.Load()).To(Succeed())
	ExpectWithOffset(1, store.Config().IsMultiGroup()).To(Equal(multigroup), "unexpected multigroup in PROJECT")
	ExpectWithOffset(1, store.Config().IsNamespaced()).To(BeTrue(), "namespaced must stay enabled in PROJECT")
}

// expectNamespacedRBAC checks the manager RBAC is a Role/RoleBinding, not a ClusterRole/ClusterRoleBinding.
func expectNamespacedRBAC(kbc *utils.TestContext) {
	By("checking config/rbac uses a Role and RoleBinding")
	role, err := os.ReadFile(filepath.Join(kbc.Dir, "config", "rbac", "role.yaml"))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, string(role)).To(MatchRegexp(`(?m)^kind: Role$`), "role.yaml must contain a Role")
	ExpectWithOffset(1, string(role)).NotTo(ContainSubstring("kind: ClusterRole"))

	binding, err := os.ReadFile(filepath.Join(kbc.Dir, "config", "rbac", "role_binding.yaml"))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, string(binding)).To(MatchRegexp(`(?m)^kind: RoleBinding$`),
		"role_binding.yaml must contain a RoleBinding")
	ExpectWithOffset(1, string(binding)).NotTo(ContainSubstring("kind: ClusterRoleBinding"))
	ExpectWithOffset(1, string(binding)).To(MatchRegexp(`(?m)^\s+kind: Role$`),
		"role_binding.yaml must reference a Role")
}

// expectBoilerplateOwner checks the license header was regenerated with the given owner.
func expectBoilerplateOwner(kbc *utils.TestContext, owner string) {
	By("checking hack/boilerplate.go.txt has the apache2 header")
	header, err := os.ReadFile(filepath.Join(kbc.Dir, "hack", "boilerplate.go.txt"))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, string(header)).To(ContainSubstring("Copyright YEAR " + owner + "."))
	ExpectWithOffset(1, string(header)).To(ContainSubstring("Licensed under the Apache License, Version 2.0"))
}
