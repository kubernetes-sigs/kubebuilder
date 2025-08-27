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

package v2alpha

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

var _ = Describe("editSubcommand", func() {
	var (
		editCmd *editSubcommand
		cfg     config.Config
		fs      machinery.Filesystem
	)

	BeforeEach(func() {
		// Create test config
		memFs := afero.NewMemMapFs()
		fs = machinery.Filesystem{FS: memFs}
		store := yaml.New(fs)

		// Create a basic PROJECT file
		projectContent := `domain: example.com
layout:
- go.kubebuilder.io/v4
projectName: test-project
repo: example.com/test-project
version: "3"
`
		err := afero.WriteFile(memFs, "PROJECT", []byte(projectContent), 0o644)
		Expect(err).NotTo(HaveOccurred())

		err = store.LoadFrom("PROJECT")
		Expect(err).NotTo(HaveOccurred())

		cfg = store.Config()

		// Create edit subcommand
		editCmd = &editSubcommand{
			config: cfg,
		}
	})

	Context("UpdateMetadata", func() {
		It("should set correct metadata", func() {
			cliMeta := plugin.CLIMetadata{CommandName: "kubebuilder"}
			meta := plugin.SubcommandMetadata{}
			editCmd.UpdateMetadata(cliMeta, &meta)

			Expect(meta.Description).To(ContainSubstring("Generate a Helm chart"))
			Expect(meta.Description).To(ContainSubstring("kustomize"))
			Expect(meta.Examples).NotTo(BeEmpty())
		})
	})

	Context("BindFlags", func() {
		It("should bind flags correctly", func() {
			flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
			editCmd.BindFlags(flagSet)

			// Check that flags were added
			manifestsFlag := flagSet.Lookup("manifests")
			Expect(manifestsFlag).NotTo(BeNil())
			Expect(manifestsFlag.DefValue).To(Equal(DefaultManifestsFile))

			outputFlag := flagSet.Lookup("output-dir")
			Expect(outputFlag).NotTo(BeNil())
			Expect(outputFlag.DefValue).To(Equal(DefaultOutputDir))

			forceFlag := flagSet.Lookup("force")
			Expect(forceFlag).NotTo(BeNil())
		})
	})

	Context("InjectConfig", func() {
		It("should inject config correctly", func() {
			err := editCmd.InjectConfig(cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(editCmd.config).To(Equal(cfg))
		})
	})

	Context("hasWebhooksWith", func() {
		It("should return false for config without webhooks", func() {
			result := hasWebhooksWith(cfg)
			Expect(result).To(BeFalse())
		})
	})

	Context("PostScaffold", func() {
		BeforeEach(func() {
			// Create the directory structure
			err := fs.FS.MkdirAll(".github/workflows", 0o755)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not modify workflow file when no webhooks present", func() {
			// Create test workflow file
			workflowContent := `name: Test Chart
on:
  push:
    branches: [main]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

#      - name: Install cert-manager via Helm
#        run: |
#          helm repo add jetstack https://charts.jetstack.io
#          helm repo update
#          helm install cert-manager jetstack/cert-manager \
#            --namespace cert-manager --create-namespace --set crds.enabled=true
#
#      - name: Wait for cert-manager to be ready
#        run: |
#          kubectl wait --namespace cert-manager --for=condition=available \
#            --timeout=300s deployment/cert-manager
#          kubectl wait --namespace cert-manager --for=condition=available \
#            --timeout=300s deployment/cert-manager-cainjector
#          kubectl wait --namespace cert-manager --for=condition=available \
#            --timeout=300s deployment/cert-manager-webhook
`
			workflowPath := filepath.Join(".github", "workflows", "test-chart.yml")
			err := afero.WriteFile(fs.FS, workflowPath, []byte(workflowContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			err = editCmd.PostScaffold()
			Expect(err).NotTo(HaveOccurred())

			// Content should remain unchanged
			content, err := afero.ReadFile(fs.FS, workflowPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("#      - name: Install cert-manager via Helm"))
		})

		It("should handle missing workflow file gracefully", func() {
			editCmd.config = cfg
			err := editCmd.PostScaffold()
			Expect(err).NotTo(HaveOccurred()) // Should not error even if file doesn't exist
		})
	})
})
