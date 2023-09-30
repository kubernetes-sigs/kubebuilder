/*
Copyright 2023 The Kubernetes Authors.

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

package externalplugin

import (
	"path/filepath"

	pluginutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	//nolint:golint
	//nolint:revive
	//nolint:golint
	//nolint:revive
	"sigs.k8s.io/kubebuilder/v3/test/e2e/utils"
)

var _ = Describe("kubebuilder", func() {
	Context("plugin sampleexternalplugin/v1", func() {
		var (
			kbc *utils.TestContext
		)

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred(), "Prepare NewTestContext should return no error.")
			Expect(kbc.Prepare()).To(Succeed())
		})

		AfterEach(func() {
			kbc.Destroy()
		})

		It("should generate a runnable project with sample external plugin", func() {
			GenerateProject(kbc)
		})

	})
})

// GenerateProject implements a sampleexternalplugin/v1 external plugin project defined by a TestContext.
func GenerateProject(kbc *utils.TestContext) {
	var err error

	By("initializing a project")
	err = kbc.Init(
		"--plugins", "sampleexternalplugin/v1",
		"--domain", "sample.domain.com",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	var initFileContentsTmpl = "A simple text file created with the `init` subcommand\nDOMAIN: sample.domain.com"
	initFileContainsExpr, err := pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "initFile.txt"), initFileContentsTmpl)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Check initFile.txt should return no error.")
	ExpectWithOffset(1, initFileContainsExpr).To(BeTrue(), "The init file does not contain the expected expression.")

	By("creating API definition")
	err = kbc.CreateAPI(
		"--plugins", "sampleexternalplugin/v1",
		"--number=2",
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	var apiFileContentsTmpl = "A simple text file created with the `create api` subcommand\nNUMBER: 2"
	apiFileContainsExpr, err := pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "apiFile.txt"), apiFileContentsTmpl)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Check apiFile.txt should return no error.")
	ExpectWithOffset(1, apiFileContainsExpr).To(BeTrue(), "The api file does not contain the expected expression.")

	By("scaffolding webhook")
	err = kbc.CreateWebhook(
		"--plugins", "sampleexternalplugin/v1",
		"--hooked",
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	var webhookFileContentsTmpl = "A simple text file created with the `create webhook` subcommand\nHOOKED!"
	webhookFileContainsExpr, err := pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "webhookFile.txt"), webhookFileContentsTmpl)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Check webhookFile.txt should return no error.")
	ExpectWithOffset(1, webhookFileContainsExpr).To(BeTrue(), "The webhook file does not contain the expected expression.")
}
