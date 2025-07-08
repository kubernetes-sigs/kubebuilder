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

		It("should extend a runnable project with helm plugin", func() {
			initTheProject(kbc)
			generateProject(kbc, "")
		})

		It("should extend a runnable project with helm plugin and webhooks", func() {
			initTheProject(kbc)
			generateProject(kbc, "")
			extendProjectWithWebhooks(kbc)

			By("re-edit the project after creating webhooks")
			err := kbc.Edit(
				"--plugins", "helm.kubebuilder.io/v1-alpha", "--force",
			)
			Expect(err).NotTo(HaveOccurred(), "Failed to edit the project")

			fileContainsExpr, err := pluginutil.HasFileContentWith(
				filepath.Join(kbc.Dir, "dist", "chart", "values.yaml"),
				`webhook:
  enable: true`)
			Expect(err).NotTo(HaveOccurred(), "Failed to read values.yaml file")
			Expect(fileContainsExpr).To(BeTrue(), "Failed to get enabled webhook value from values.yaml file")
		})

		// Without --force, the webhooks should not be enabled in the values.yaml file.
		It("should extend a runnable project with helm plugin but not running with --force", func() {
			initTheProject(kbc)
			generateProject(kbc, "")
			extendProjectWithWebhooks(kbc)

			By("re-edit the project after creating webhooks without --force")
			err := kbc.Edit(
				"--plugins", "helm.kubebuilder.io/v1-alpha",
			)
			Expect(err).NotTo(HaveOccurred(), "Failed to edit the project")

			fileContainsExpr, err := pluginutil.HasFileContentWith(
				filepath.Join(kbc.Dir, "dist", "chart", "values.yaml"),
				`webhook:
  enable: true`)
			Expect(err).NotTo(HaveOccurred(), "Failed to read values.yaml file")
			Expect(fileContainsExpr).To(BeFalse(), "Failed to get enabled webhook value from values.yaml file")
		})

		It("should extend a runnable project with helm plugin and a custom helm directory", func() {
			initTheProject(kbc)
			generateProject(kbc, "helm-charts")
		})
	})
})

// generateProject implements a helm/v1(-alpha) plugin project defined by a TestContext.
func generateProject(kbc *utils.TestContext, directory string) {
	var err error

	editOptions := []string{
		"--plugins", "helm.kubebuilder.io/v1-alpha",
	}

	if directory != "" {
		editOptions = append(editOptions, "--output-dir", directory)
	} else {
		directory = "dist"
	}

	By("editing a project")
	err = kbc.Edit(
		editOptions...,
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to edit the project")

	fileContainsExpr, err := pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "PROJECT"),
		`helm.kubebuilder.io/v1-alpha:
    options:
      directory: `+directory)
	Expect(err).NotTo(HaveOccurred(), "Failed to read PROJECT file")
	Expect(fileContainsExpr).To(BeTrue(), "Failed to find helm plugin in PROJECT file")

	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "PROJECT"),
		"projectName: e2e-"+kbc.TestSuffix)
	Expect(err).NotTo(HaveOccurred(), "Failed to read PROJECT file")
	Expect(fileContainsExpr).To(BeTrue(), "Failed to find projectName in PROJECT file")

	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, directory, "chart", "Chart.yaml"),
		"name: e2e-"+kbc.TestSuffix)
	Expect(err).NotTo(HaveOccurred(), "Failed to read Chart.yaml file")
	Expect(fileContainsExpr).To(BeTrue(), "Failed to find name in Chart.yaml file")

	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, directory, "chart", "templates", "manager", "manager.yaml"),
		`metadata:
  name: e2e-`+kbc.TestSuffix)
	Expect(err).NotTo(HaveOccurred(), "Failed to read manager.yaml file")
	Expect(fileContainsExpr).To(BeTrue(), "Failed to find name in helm template manager.yaml file")
}

// extendProjectWithWebhooks is creating API and scaffolding webhooks in the project
func extendProjectWithWebhooks(kbc *utils.TestContext) {
	By("creating API definition")
	err := kbc.CreateAPI(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--namespaced",
		"--resource",
		"--controller",
		"--make=false",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to create API")

	By("scaffolding mutating and validating webhooks")
	err = kbc.CreateWebhook(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--defaulting",
		"--programmatic-validation",
		"--make=false",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to scaffolding mutating webhook")

	By("run make manifests")
	Expect(kbc.Make("manifests")).To(Succeed())
}

// initTheProject initializes a project with the go/v4 plugin and sets the domain.
func initTheProject(kbc *utils.TestContext) {
	By("initializing a project")
	err := kbc.Init(
		"--plugins", "go/v4",
		"--project-version", "3",
		"--domain", kbc.Domain,
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to initialize project")
}
