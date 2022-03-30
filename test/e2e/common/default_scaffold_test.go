/*
Copyright 2022 The Kubernetes Authors.

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

package common

import (
	//nolint:golint
	//nolint:revive
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v3/test/e2e/utils"
)

var _ = Describe("kubebuilder", func() {
	Context("with default plugin and project", func() {
		var (
			kbc *utils.TestContext
		)

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(util.KubebuilderBinName)
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())
		})

		AfterEach(func() {
			By("removing working dir")
			if err := os.RemoveAll(kbc.Dir); err != nil {
				Expect(err).NotTo(HaveOccurred())
			}
		})

		// To ensure that the default scaffold will be done with the default
		// plugin and project file which currently is go/v3
		// It ensures that we do not add breaking changes when we add new plugins
		It("should scaffold a project with the defaults", func() {
			const defaultLayoutPlugin = "layout:\n- go.kubebuilder.io/v3"
			const defaultProjectVersion = "version: \"3\""

			By("initializing a project without inform the plugin")
			err := kbc.Init(
				"--domain", kbc.Domain,
				"--fetch-deps=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("checking plugin and version used to do the scaffold")
			found, err := util.FoundInFile(filepath.Join(kbc.Dir, "PROJECT"), defaultLayoutPlugin)
			Expect(err).NotTo(HaveOccurred())
			Expect(found).Should(Equal(true))

			By("checking project schema version used to do the scaffold")
			found, err = util.FoundInFile(filepath.Join(kbc.Dir, "PROJECT"), defaultProjectVersion)
			Expect(err).NotTo(HaveOccurred())
			Expect(found).Should(Equal(true))
		})
	})
})
