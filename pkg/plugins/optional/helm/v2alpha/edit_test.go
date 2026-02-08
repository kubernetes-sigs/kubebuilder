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
	"os"
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

	Context("removeV1AlphaPluginEntry", func() {
		It("should remove v1-alpha plugin entry if it exists", func() {
			// Add v1-alpha plugin entry to config
			err := cfg.EncodePluginConfig(v1AlphaPluginKey, map[string]any{})
			Expect(err).NotTo(HaveOccurred())

			// Verify it exists
			var v1AlphaCfg map[string]any
			err = cfg.DecodePluginConfig(v1AlphaPluginKey, &v1AlphaCfg)
			Expect(err).NotTo(HaveOccurred())

			// Remove it
			editCmd.removeV1AlphaPluginEntry()

			// Verify it was removed from in-memory config
			err = cfg.DecodePluginConfig(v1AlphaPluginKey, &v1AlphaCfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("plugin key"))
		})

		It("should not error when v1-alpha entry does not exist", func() {
			// Should not panic or error
			editCmd.removeV1AlphaPluginEntry()
		})

		It("should not error when plugins map is nil", func() {
			// Create a fresh config without any plugins
			memFs := afero.NewMemMapFs()
			freshFs := machinery.Filesystem{FS: memFs}
			store := yaml.New(freshFs)

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

			freshCfg := store.Config()
			freshEditCmd := &editSubcommand{config: freshCfg}

			// Should not panic or error
			freshEditCmd.removeV1AlphaPluginEntry()
		})

		It("should persist v1-alpha removal when config is saved", func() {
			// Create a store to test actual persistence
			memFs := afero.NewMemMapFs()
			testFs := machinery.Filesystem{FS: memFs}
			store := yaml.New(testFs)

			// Create PROJECT file with v1-alpha plugin entry
			projectContent := `domain: example.com
layout:
- go.kubebuilder.io/v4
plugins:
  helm.kubebuilder.io/v1-alpha: {}
projectName: test-project
repo: example.com/test-project
version: "3"
`
			err := afero.WriteFile(memFs, "PROJECT", []byte(projectContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			err = store.LoadFrom("PROJECT")
			Expect(err).NotTo(HaveOccurred())

			testCfg := store.Config()
			testEditCmd := &editSubcommand{config: testCfg}

			// Verify v1-alpha entry exists before removal
			var v1AlphaCfg map[string]any
			err = testCfg.DecodePluginConfig(v1AlphaPluginKey, &v1AlphaCfg)
			Expect(err).NotTo(HaveOccurred())

			// Remove v1-alpha entry
			testEditCmd.removeV1AlphaPluginEntry()

			// Save config to disk
			err = store.Save()
			Expect(err).NotTo(HaveOccurred())

			// Reload config from disk
			err = store.LoadFrom("PROJECT")
			Expect(err).NotTo(HaveOccurred())

			reloadedCfg := store.Config()

			// Verify v1-alpha entry is gone after reload
			err = reloadedCfg.DecodePluginConfig(v1AlphaPluginKey, &v1AlphaCfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("plugin key"))
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

	Context("addHelmMakefileTargets", func() {
		var tmpDir string

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "helm-makefile-test-*")
			Expect(err).NotTo(HaveOccurred())

			// Change to temp directory
			err = os.Chdir(tmpDir)
			Expect(err).NotTo(HaveOccurred())

			editCmd.outputDir = DefaultOutputDir
		})

		AfterEach(func() {
			// Clean up temp directory
			if tmpDir != "" {
				_ = os.RemoveAll(tmpDir)
			}
		})

		It("should add Helm targets to Makefile when it exists", func() {
			// Create a basic Makefile
			makefileContent := `IMG ?= controller:latest

##@ Development

.PHONY: build
build: ## Build manager binary.
	go build -o bin/manager cmd/main.go
`
			err := os.WriteFile("Makefile", []byte(makefileContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			err = editCmd.addHelmMakefileTargets("test-project-system")
			Expect(err).NotTo(HaveOccurred())

			// Verify Helm targets were added
			content, err := os.ReadFile("Makefile")
			Expect(err).NotTo(HaveOccurred())

			contentStr := string(content)
			Expect(contentStr).To(ContainSubstring("##@ Helm Deployment"))
			Expect(contentStr).To(ContainSubstring("## Helm binary to use for deploying the chart"))
			Expect(contentStr).To(ContainSubstring("HELM ?= helm"))
			Expect(contentStr).To(ContainSubstring("## Namespace to deploy the Helm release"))
			Expect(contentStr).To(ContainSubstring("HELM_NAMESPACE ?= test-project-system"))
			Expect(contentStr).To(ContainSubstring("## Name of the Helm release"))
			Expect(contentStr).To(ContainSubstring("HELM_RELEASE ?= test-project"))
			Expect(contentStr).To(ContainSubstring("## Path to the Helm chart directory"))
			Expect(contentStr).To(ContainSubstring("HELM_CHART_DIR ?= dist/chart"))
			Expect(contentStr).To(ContainSubstring("## Additional arguments to pass to helm commands"))
			Expect(contentStr).To(ContainSubstring("HELM_EXTRA_ARGS ?="))
			Expect(contentStr).To(ContainSubstring(".PHONY: helm-deploy"))
			Expect(contentStr).To(ContainSubstring(
				"helm-deploy: ## Deploy manager to the K8s cluster via Helm. Specify an image with IMG."))
			Expect(contentStr).To(ContainSubstring("--set manager.image.repository=$${IMG%:*}"))
			Expect(contentStr).To(ContainSubstring("--set manager.image.tag=$${IMG##*:}"))
			Expect(contentStr).To(ContainSubstring(".PHONY: helm-uninstall"))
			Expect(contentStr).To(ContainSubstring(
				"helm-uninstall: ## Uninstall the Helm release from the K8s cluster."))
			Expect(contentStr).To(ContainSubstring(".PHONY: helm-status"))
			Expect(contentStr).To(ContainSubstring("helm-status: ## Show Helm release status."))
			Expect(contentStr).To(ContainSubstring(".PHONY: helm-history"))
			Expect(contentStr).To(ContainSubstring("helm-history: ## Show Helm release history."))
			Expect(contentStr).To(ContainSubstring(".PHONY: helm-rollback"))
			Expect(contentStr).To(ContainSubstring("helm-rollback: ## Rollback to previous Helm release."))
		})

		It("should not duplicate Helm targets if already present", func() {
			// Create a Makefile that already has Helm targets (exact match to template)
			makefileContent := `IMG ?= controller:latest

.PHONY: build
build: ## Build manager binary.
	go build -o bin/manager cmd/main.go

##@ Helm Deployment

## Helm binary to use for deploying the chart
HELM ?= helm
## Namespace to deploy the Helm release
HELM_NAMESPACE ?= test-project-system
## Name of the Helm release
HELM_RELEASE ?= test-project
## Path to the Helm chart directory
HELM_CHART_DIR ?= dist/chart
## Additional arguments to pass to helm commands
HELM_EXTRA_ARGS ?=

.PHONY: helm-deploy
helm-deploy: ## Deploy manager to the K8s cluster via Helm. Specify an image with IMG.
	$(HELM) upgrade --install $(HELM_RELEASE) $(HELM_CHART_DIR) \
		--namespace $(HELM_NAMESPACE) \
		--create-namespace \
		--set manager.image.repository=$${IMG%:*} \
		--set manager.image.tag=$${IMG##*:} \
		--wait \
		--timeout 5m \
		$(HELM_EXTRA_ARGS)

.PHONY: helm-uninstall
helm-uninstall: ## Uninstall the Helm release from the K8s cluster.
	$(HELM) uninstall $(HELM_RELEASE) --namespace $(HELM_NAMESPACE)

.PHONY: helm-status
helm-status: ## Show Helm release status.
	$(HELM) status $(HELM_RELEASE) --namespace $(HELM_NAMESPACE)

.PHONY: helm-history
helm-history: ## Show Helm release history.
	$(HELM) history $(HELM_RELEASE) --namespace $(HELM_NAMESPACE)

.PHONY: helm-rollback
helm-rollback: ## Rollback to previous Helm release.
	$(HELM) rollback $(HELM_RELEASE) --namespace $(HELM_NAMESPACE)
`
			err := os.WriteFile("Makefile", []byte(makefileContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			err = editCmd.addHelmMakefileTargets("test-project-system")
			Expect(err).NotTo(HaveOccurred())

			// Verify targets were not duplicated
			content, err := os.ReadFile("Makefile")
			Expect(err).NotTo(HaveOccurred())

			// Count occurrences of helm-deploy target
			contentStr := string(content)
			helmDeployCount := 0
			for i := 0; i < len(contentStr)-len("helm-deploy:"); i++ {
				if contentStr[i:i+len("helm-deploy:")] == "helm-deploy:" {
					helmDeployCount++
				}
			}
			Expect(helmDeployCount).To(Equal(1)) // Should only appear once
		})

		It("should return error when Makefile does not exist", func() {
			err := editCmd.addHelmMakefileTargets("test-project-system")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("makefile not found"))
		})
	})
})
