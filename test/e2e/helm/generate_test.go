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

package helm

import (
	"path/filepath"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"

	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("kubebuilder", func() {
	Context("plugin helm/v1-alpha", func() {
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

		It("should generate a runnable project with helm plugin", func() {
			GenerateProject(kbc)
		})
	})
})

// GenerateProject implements a helm/v1(-alpha) plugin project defined by a TestContext.
func GenerateProject(kbc *utils.TestContext) {
	var err error

	By("initializing a project")
	err = kbc.Init(
		"--plugins", "helm.kubebuilder.io/v1-alpha",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to initialize project")

	fileContainsExpr, err := pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "PROJECT"),
		`helm.kubebuilder.io/v1-alpha: {}`)
	Expect(err).NotTo(HaveOccurred(), "Failed to edit sum rate for custom metrics")
	Expect(fileContainsExpr).To(BeTrue(), "Failed to find helm plugin in PROJECT file")

	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "PROJECT"),
		"projectName: e2e-"+kbc.TestSuffix)
	Expect(err).NotTo(HaveOccurred(), "Failed to edit sum rate for custom metrics")
	Expect(fileContainsExpr).To(BeTrue(), "Failed to find projectName in PROJECT file")

	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "dist", "chart", "Chart.yaml"),
		"name: e2e-"+kbc.TestSuffix)
	Expect(err).NotTo(HaveOccurred(), "Failed to edit sum rate for custom metrics")
	Expect(fileContainsExpr).To(BeTrue(), "Failed to find name in Chart.yaml file")

	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "dist", "chart", "templates", "manager", "manager.yaml"),
		`metadata:
  name: e2e-`+kbc.TestSuffix)
	Expect(err).NotTo(HaveOccurred(), "Failed to edit sum rate for custom metrics")
	Expect(fileContainsExpr).To(BeTrue(), "Failed to find name in helm template manager.yaml file")

}
