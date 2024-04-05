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

package alphagenerate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	pluginutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	// nolint:revive
	. "github.com/onsi/ginkgo/v2"
	// nolint:revive
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v3/test/e2e/utils"
)

var _ = Describe("kubebuilder", func() {
	Context("alpha generate ", func() {
		var (
			kbc *utils.TestContext
		)

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())
		})

		AfterEach(func() {
			kbc.Destroy()
		})

		It("should regenerate the project with success", func() {
			ReGenerateProject(kbc)
		})

	})
})

// ReGenerateProject implements a project that is regenerated by kubebuilder.
func ReGenerateProject(kbc *utils.TestContext) {
	var err error

	By("initializing a project")
	err = kbc.Init(
		"--plugins", "go/v4",
		"--project-version", "3",
		"--domain", kbc.Domain,
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("regenerating the project")
	err = kbc.Regenerate(
		"--input-dir", kbc.Dir,
		"--output-dir", filepath.Join(kbc.Dir, "testdir"),
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("checking if the project file was generated with the expected layout")
	var layout = `layout:
- go.kubebuilder.io/v4
`
	fileContainsExpr, err := pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "testdir", "PROJECT"), layout)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, fileContainsExpr).To(BeTrue())

	By("checking if the project file was generated with the expected domain")
	var domain = fmt.Sprintf("domain: %s", kbc.Domain)
	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "testdir", "PROJECT"), domain)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, fileContainsExpr).To(BeTrue())

	By("checking if the project file was generated with the expected version")
	var version = `version: "3"`
	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "testdir", "PROJECT"), version)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, fileContainsExpr).To(BeTrue())

	By("editing a project with multigroup=true")
	err = kbc.Edit(
		"--multigroup=true",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("create APIs with resource and controller")
	err = kbc.CreateAPI(
		"--group", "crew",
		"--version", "v1",
		"--kind", "Captain",
		"--namespaced",
		"--resource",
		"--controller",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("create Webhooks with conversion and validating webhook")
	err = kbc.CreateWebhook(
		"--group", "crew",
		"--version", "v1",
		"--kind", "Captain",
		"--programmatic-validation",
		"--conversion",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("create APIs with deploy-image plugin")
	err = kbc.CreateAPI(
		"--group", "crew",
		"--version", "v1",
		"--kind", "Memcached",
		"--image=memcached:1.6.15-alpine",
		"--image-container-command=memcached,-m=64,modern,-v",
		"--image-container-port=11211",
		"--run-as-user=1001",
		"--plugins=\"deploy-image/v1-alpha\"",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("Enable grafana plugin to an existing project")
	err = kbc.Edit(
		"--plugins", "grafana.kubebuilder.io/v1-alpha",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("Edit the grafana config file")
	grafanaConfig, err := os.OpenFile(filepath.Join(kbc.Dir, "grafana/custom-metrics/config.yaml"),
		os.O_APPEND|os.O_WRONLY, 0644)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	newLine := "test_new_line"
	_, err = io.WriteString(grafanaConfig, newLine)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	err = grafanaConfig.Close()
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("regenerating the project at another output directory")
	err = kbc.Regenerate(
		"--input-dir", kbc.Dir,
		"--output-dir", filepath.Join(kbc.Dir, "testdir2"),
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("checking if the project file was generated with the expected multigroup flag")
	var multiGroup = `multigroup: true`
	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "testdir2", "PROJECT"), multiGroup)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, fileContainsExpr).To(BeTrue())

	By("checking if the project file was generated with the expected group")
	var APIGroup = "group: crew"
	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "testdir2", "PROJECT"), APIGroup)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, fileContainsExpr).To(BeTrue())

	By("checking if the project file was generated with the expected kind")
	var APIKind = "kind: Captain"
	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "testdir2", "PROJECT"), APIKind)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, fileContainsExpr).To(BeTrue())

	By("checking if the project file was generated with the expected version")
	var APIVersion = "version: v1"
	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "testdir2", "PROJECT"), APIVersion)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, fileContainsExpr).To(BeTrue())

	By("checking if the project file was generated with the expected namespaced")
	var namespaced = "namespaced: true"
	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "testdir2", "PROJECT"), namespaced)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, fileContainsExpr).To(BeTrue())

	By("checking if the project file was generated with the expected controller")
	var controller = "controller: true"
	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "testdir2", "PROJECT"), controller)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, fileContainsExpr).To(BeTrue())

	By("checking if the project file was generated with the expected webhook")
	var webhook = `webhooks:
    conversion: true
    validation: true`
	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "testdir2", "PROJECT"), webhook)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, fileContainsExpr).To(BeTrue())

	By("checking if the project file was generated with the expected deploy-image plugin fields")
	var deployImagePlugin = "deploy-image.go.kubebuilder.io/v1-alpha"
	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "testdir2", "PROJECT"), deployImagePlugin)
	Expect(err).NotTo(HaveOccurred())
	Expect(fileContainsExpr).To(BeTrue())
	var deployImagePluginFields = `kind: Memcached
      options:
        containerCommand: memcached,-m=64,modern,-v
        containerPort: "11211"
        image: memcached:1.6.15-alpine
        runAsUser: "1001"`
	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "testdir2", "PROJECT"), deployImagePluginFields)
	Expect(err).NotTo(HaveOccurred())
	Expect(fileContainsExpr).To(BeTrue())

	By("checking if the project file was generated with the expected grafana plugin fields")
	var grafanaPlugin = "grafana.kubebuilder.io/v1-alpha"
	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "testdir2", "PROJECT"), grafanaPlugin)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, fileContainsExpr).To(BeTrue())

	By("checking if the generated grafana config file has the same content as the old one")
	grafanaConfigPath := filepath.Join(kbc.Dir, "grafana/custom-metrics/config.yaml")
	generatedGrafanaConfigPath := filepath.Join(kbc.Dir, "testdir2", "grafana/custom-metrics/config.yaml")
	Expect(grafanaConfigPath).Should(BeARegularFile())
	Expect(generatedGrafanaConfigPath).Should(BeARegularFile())
	bytesBefore, err := os.ReadFile(grafanaConfigPath)
	Expect(err).NotTo(HaveOccurred())
	bytesAfter, err := os.ReadFile(generatedGrafanaConfigPath)
	Expect(err).NotTo(HaveOccurred())
	Expect(bytesBefore).Should(Equal(bytesAfter))
}
