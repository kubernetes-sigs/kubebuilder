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

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("sampleexternalplugin", func() {
	Context("plugin sampleexternalplugin/v1", func() {
		var (
			kbc *utils.TestContext
		)

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())

			By("installing the sampleexternalplugin")
			err = installPlugin()
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			By("removing working dir")
			kbc.Destroy()
		})

		It("should scaffold Prometheus instance with init", func() {
			By("initializing a project with go/v4 and sampleexternalplugin/v1")
			err := kbc.Init(
				"--plugins", "go/v4,sampleexternalplugin/v1",
				"--domain", kbc.Domain,
				"--prometheus-namespace", "monitoring",
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying Prometheus instance manifest was created")
			prometheusFile := filepath.Join(kbc.Dir, "config", "prometheus", "prometheus.yaml")
			Expect(prometheusFile).To(BeAnExistingFile())

			By("verifying prometheus manifest contains monitoring namespace")
			content, err := os.ReadFile(prometheusFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("namespace: monitoring"))
			Expect(string(content)).To(ContainSubstring("kind: Prometheus"))

			By("verifying kustomization.yaml was created")
			kustomizationFile := filepath.Join(kbc.Dir, "config", "prometheus", "kustomization.yaml")
			Expect(kustomizationFile).To(BeAnExistingFile())

			By("verifying kustomization contains namespace")
			content, err = os.ReadFile(kustomizationFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("namespace: monitoring"))
			Expect(string(content)).To(ContainSubstring("- prometheus.yaml"))

			By("verifying setup instructions were created")
			patchFile := filepath.Join(kbc.Dir, "config", "default", "kustomization_prometheus_patch.yaml")
			Expect(patchFile).To(BeAnExistingFile())
		})

		It("should scaffold Prometheus instance with edit", func() {
			By("initializing a project with go/v4 only")
			err := kbc.Init(
				"--plugins", "go/v4",
				"--domain", kbc.Domain,
			)
			Expect(err).NotTo(HaveOccurred())

			By("running edit command with sampleexternalplugin/v1")
			editCmd := exec.Command(kbc.BinaryName, "edit",
				"--plugins", "sampleexternalplugin/v1",
				"--prometheus-namespace", "observability",
			)
			editCmd.Dir = kbc.Dir
			output, err := kbc.Run(editCmd)
			Expect(err).NotTo(HaveOccurred(), "edit command failed: %s", string(output))

			By("verifying Prometheus instance manifest was created")
			prometheusFile := filepath.Join(kbc.Dir, "config", "prometheus", "prometheus.yaml")
			Expect(prometheusFile).To(BeAnExistingFile())

			By("verifying prometheus manifest contains observability namespace")
			content, err := os.ReadFile(prometheusFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("namespace: observability"))

			By("verifying kustomization contains observability namespace")
			kustomizationFile := filepath.Join(kbc.Dir, "config", "prometheus", "kustomization.yaml")
			content, err = os.ReadFile(kustomizationFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("namespace: observability"))
		})

		It("should validate namespace format", func() {
			By("attempting to initialize with invalid namespace (uppercase)")
			err := kbc.Init(
				"--plugins", "go/v4,sampleexternalplugin/v1",
				"--domain", kbc.Domain,
				"--prometheus-namespace", "Invalid-Namespace",
			)
			Expect(err).To(HaveOccurred())
		})

		It("should work with default namespace when flag not provided", func() {
			By("initializing without prometheus-namespace flag")
			err := kbc.Init(
				"--plugins", "go/v4,sampleexternalplugin/v1",
				"--domain", kbc.Domain,
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying default namespace (monitoring-system) was used")
			prometheusFile := filepath.Join(kbc.Dir, "config", "prometheus", "prometheus.yaml")
			content, err := os.ReadFile(prometheusFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("namespace: monitoring-system"))
		})
	})
})

// installPlugin builds and installs the sampleexternalplugin binary
func installPlugin() error {
	// Get the plugin directory (5 levels up from test/e2e)
	pluginDir, err := filepath.Abs("../../")
	if err != nil {
		return fmt.Errorf("failed to get plugin directory: %w", err)
	}

	// Run make install in the plugin directory
	cmd := exec.Command("make", "install")
	cmd.Dir = pluginDir
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install plugin: %w", err)
	}

	return nil
}
