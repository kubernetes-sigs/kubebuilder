// Copyright 2026 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build integration

package v4

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("Delete Comprehensive Integration Tests", func() {
	var kbc *utils.TestContext

	BeforeEach(func() {
		var err error
		kbc, err = utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(kbc.Prepare()).To(Succeed())

		err = kbc.Init("--plugins", "go/v4", "--domain", kbc.Domain, "--repo", kbc.Domain, "--skip-go-version-check")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		kbc.Destroy()
	})

	It("should test all critical delete scenarios efficiently", func() {
		By("Scenario 1: API deletion protection and granular webhook deletion")
		err := kbc.CreateAPI("--group", "test", "--version", "v1", "--kind", "Protected",
			"--resource", "--controller", "--make=false")
		Expect(err).NotTo(HaveOccurred())

		err = kbc.CreateWebhook("--group", "test", "--version", "v1", "--kind", "Protected",
			"--defaulting", "--programmatic-validation", "--make=false")
		Expect(err).NotTo(HaveOccurred())

		err = kbc.DeleteAPI("--group", "test", "--version", "v1", "--kind", "Protected",
			"--skip-confirmation")
		Expect(err).To(HaveOccurred(), "API deletion should be blocked")
		Expect(err.Error()).To(ContainSubstring("webhooks are configured"))

		projectBefore, _ := os.ReadFile(filepath.Join(kbc.Dir, "PROJECT"))
		Expect(string(projectBefore)).To(ContainSubstring("defaulting: true"))
		Expect(string(projectBefore)).To(ContainSubstring("validation: true"))

		err = kbc.DeleteWebhook("--group", "test", "--version", "v1", "--kind", "Protected",
			"--defaulting", "--skip-confirmation")
		Expect(err).NotTo(HaveOccurred())

		projectAfter, _ := os.ReadFile(filepath.Join(kbc.Dir, "PROJECT"))
		Expect(string(projectAfter)).NotTo(ContainSubstring("defaulting: true"))
		Expect(string(projectAfter)).To(ContainSubstring("validation: true"))
		Expect(kbc.HasFile("internal/webhook/v1/protected_webhook.go")).To(BeTrue(),
			"files remain when validation exists")

		err = kbc.DeleteWebhook("--group", "test", "--version", "v1", "--kind", "Protected",
			"--programmatic-validation", "--skip-confirmation")
		Expect(err).NotTo(HaveOccurred())

		Expect(kbc.HasFile("internal/webhook/v1/protected_webhook.go")).To(BeFalse(),
			"files deleted when all types removed")
		Expect(kbc.HasFile("config/certmanager/kustomization.yaml")).To(BeFalse(),
			"certmanager deleted with last webhook")

		err = kbc.DeleteAPI("--group", "test", "--version", "v1", "--kind", "Protected",
			"--skip-confirmation")
		Expect(err).NotTo(HaveOccurred())
	})

	It("should validate error conditions", func() {
		By("creating API for error testing")
		err := kbc.DeleteAPI("--group", "none", "--version", "v1", "--kind", "Missing",
			"--skip-confirmation")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("does not exist"))

		err = kbc.CreateAPI("--group", "test", "--version", "v1", "--kind", "NoWebhook",
			"--resource", "--controller", "--make=false")
		Expect(err).NotTo(HaveOccurred())

		err = kbc.DeleteWebhook("--group", "test", "--version", "v1", "--kind", "NoWebhook",
			"--skip-confirmation")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("does not have any webhooks"))

		err = kbc.CreateWebhook("--group", "test", "--version", "v1", "--kind", "NoWebhook",
			"--defaulting", "--make=false")
		Expect(err).NotTo(HaveOccurred())

		err = kbc.DeleteWebhook("--group", "test", "--version", "v1", "--kind", "NoWebhook",
			"--programmatic-validation", "--skip-confirmation")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("does not have a validation webhook"))
	})
})
