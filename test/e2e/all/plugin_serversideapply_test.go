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
	"sigs.k8s.io/kubebuilder/v4/test/e2e/helpers"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

// Test specs for server-side-apply plugin
var _ = Describe("kubebuilder", func() {
	Context("server-side-apply plugin", func() {
		var kbc *utils.TestContext

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())
		})

		AfterEach(func() {
			By("removing restricted namespace label")
			_ = kbc.RemoveNamespaceLabelToEnforceRestricted()

			By("undeploy the project")
			_ = kbc.Make("undeploy")

			By("uninstalling the project")
			_ = kbc.Make("uninstall")

			By("removing controller image and working dir")
			kbc.Destroy()
		})

		It("should generate a runnable project with server-side-apply plugin", func() {
			generateServerSideApply(kbc)
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         false,
				HasMetrics:         true,
				HasNetworkPolicies: false,
				InstallMethod:      helpers.InstallMethodKustomize,
			})
		})

		It("should generate a runnable project with server-side-apply plugin using Helm", func() {
			generateServerSideApply(kbc)
			helpers.Run(kbc, helpers.RunOptions{
				HasWebhook:         false,
				HasMetrics:         true,
				HasNetworkPolicies: false,
				InstallMethod:      helpers.InstallMethodHelm,
			})
		})
	})
})

// generateServerSideApply implements a go/v4 plugin project and scaffolds an API using server-side-apply plugin
func generateServerSideApply(kbc *utils.TestContext) {
	By("initializing a project")
	err := kbc.Init(
		"--plugins", "go/v4",
		"--project-version", "3",
		"--domain", kbc.Domain,
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to initialize project")

	By("creating API with server-side-apply plugin")
	err = kbc.CreateAPI(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--plugins", "server-side-apply/v1-alpha",
		"--controller=true",
		"--resource=true",
		"--make=false",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to create API with server-side-apply plugin")

	By("running make generate to create apply configurations")
	err = kbc.Make("generate")
	Expect(err).NotTo(HaveOccurred(), "Failed to run make generate")

	By("verifying apply configuration files were generated")
	applyConfigPath := filepath.Join(kbc.Dir, "pkg", "applyconfiguration")
	_, err = os.Stat(applyConfigPath)
	Expect(err).NotTo(HaveOccurred(), "Apply configuration directory was not created")

	By("running make manifests")
	err = kbc.Make("manifests")
	Expect(err).NotTo(HaveOccurred(), "Failed to run make manifests")
}
