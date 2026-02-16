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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
)

var _ = Describe("Helm Makefile Targets", func() {
	var (
		tmpDir       string
		makefilePath string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "helm-makefile-test-*")
		Expect(err).NotTo(HaveOccurred())
		makefilePath = filepath.Join(tmpDir, "Makefile")

		// Create a basic Makefile with some existing content
		initialContent := `# Basic Makefile
.PHONY: all
all: build

.PHONY: build
build:
	go build -o bin/manager cmd/main.go

##@ Deployment
.PHONY: deploy
deploy:
	kubectl apply -k config/default
`
		err = os.WriteFile(makefilePath, []byte(initialContent), 0o644)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
	})

	It("should add Helm deployment targets without duplication", func() {
		// Get the Helm targets
		helmTargets := getHelmMakefileTargets("test-project", "test-system", "dist")

		By("appending Helm targets for the first time")
		err := util.AppendCodeIfNotExist(makefilePath, helmTargets)
		Expect(err).NotTo(HaveOccurred())

		// Read the Makefile
		content, err := os.ReadFile(makefilePath)
		Expect(err).NotTo(HaveOccurred())
		makefileContent := string(content)

		By("verifying Helm section was added")
		Expect(makefileContent).To(ContainSubstring("##@ Helm Deployment"))
		Expect(makefileContent).To(ContainSubstring("HELM ?= helm"))
		Expect(makefileContent).To(ContainSubstring(".PHONY: install-helm"))
		Expect(makefileContent).To(ContainSubstring("install-helm: ## Install the latest version of Helm."))
		Expect(makefileContent).To(ContainSubstring(".PHONY: helm-deploy"))
		Expect(makefileContent).To(ContainSubstring("helm-deploy: install-helm ##"))
		Expect(makefileContent).To(ContainSubstring(".PHONY: helm-uninstall"))

		// Count occurrences
		helmSectionCount := strings.Count(makefileContent, "##@ Helm Deployment")
		helmDeployCount := strings.Count(makefileContent, ".PHONY: helm-deploy")
		helmInstallCount := strings.Count(makefileContent, "install-helm: ## Install the latest version of Helm.")
		Expect(helmSectionCount).To(Equal(1), "Helm section should appear exactly once")
		Expect(helmDeployCount).To(Equal(1), "helm-deploy should appear exactly once")
		Expect(helmInstallCount).To(Equal(1), "install-helm target should appear exactly once")

		By("appending Helm targets again (should not duplicate)")
		err = util.AppendCodeIfNotExist(makefilePath, helmTargets)
		Expect(err).NotTo(HaveOccurred())

		// Read the Makefile again
		content2, err := os.ReadFile(makefilePath)
		Expect(err).NotTo(HaveOccurred())
		makefileContent2 := string(content2)

		By("verifying no duplication occurred")
		helmSectionCount2 := strings.Count(makefileContent2, "##@ Helm Deployment")
		helmDeployCount2 := strings.Count(makefileContent2, ".PHONY: helm-deploy")
		helmInstallCount2 := strings.Count(makefileContent2, "install-helm: ## Install the latest version of Helm.")
		Expect(helmSectionCount2).To(Equal(1), "Helm section should still appear exactly once")
		Expect(helmDeployCount2).To(Equal(1), "helm-deploy should still appear exactly once")
		Expect(helmInstallCount2).To(Equal(1), "install-helm target should still appear exactly once")

		// Verify content is identical (no duplication)
		Expect(makefileContent2).To(Equal(makefileContent), "Makefile should be unchanged after second append")
	})

	It("should generate correct Helm targets template", func() {
		helmTargets := getHelmMakefileTargets("my-project", "my-project-system", "dist")

		By("verifying template contains expected sections")
		Expect(helmTargets).To(ContainSubstring("##@ Helm Deployment"))
		Expect(helmTargets).To(ContainSubstring("HELM_NAMESPACE ?= my-project-system"))
		Expect(helmTargets).To(ContainSubstring("HELM_RELEASE ?= my-project"))
		Expect(helmTargets).To(ContainSubstring("HELM_CHART_DIR ?= dist/chart"))
		Expect(helmTargets).To(ContainSubstring("install-helm: ## Install the latest version of Helm."))
		Expect(helmTargets).To(ContainSubstring("helm-deploy: install-helm ##"))
		Expect(helmTargets).To(ContainSubstring("helm-uninstall:"))
		Expect(helmTargets).To(ContainSubstring("helm-status:"))
		Expect(helmTargets).To(ContainSubstring("helm-history:"))
		Expect(helmTargets).To(ContainSubstring("helm-rollback:"))
	})

	It("should handle custom output directory", func() {
		helmTargets := getHelmMakefileTargets("test-project", "test-system", "custom-charts")

		By("verifying custom directory is used")
		Expect(helmTargets).To(ContainSubstring("HELM_CHART_DIR ?= custom-charts/chart"))
	})
})
