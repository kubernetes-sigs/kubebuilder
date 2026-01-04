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

//go:build e2e

package v4

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("kubebuilder", func() {
	Context("delete api and delete webhook", func() {
		var (
			kbc *utils.TestContext
		)

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())
		})

		AfterEach(func() {
			By("clean up API and webhook deletion test")
			kbc.Destroy()
		})

		It("should delete API and webhooks using delete commands with all scenarios", func() {
			By("initializing a project")
			err := kbc.Init(
				"--plugins", "go/v4",
				"--domain", kbc.Domain,
				"--skip-go-version-check",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating an API with resource and controller")
			err = kbc.CreateAPI(
				"--group", "crew",
				"--version", "v1",
				"--kind", "Captain",
				"--resource",
				"--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			// Verify API files created
			Expect(kbc.HasFile(filepath.Join("api", "v1", "captain_types.go"))).To(BeTrue())
			Expect(kbc.HasFile(filepath.Join("internal", "controller", "captain_controller.go"))).To(BeTrue())

			By("creating defaulting and validation webhooks")
			err = kbc.CreateWebhook(
				"--group", "crew",
				"--version", "v1",
				"--kind", "Captain",
				"--defaulting",
				"--programmatic-validation",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			// Verify webhook files created
			Expect(kbc.HasFile(filepath.Join("internal", "webhook", "v1", "captain_webhook.go"))).To(BeTrue())
			Expect(kbc.HasFile(filepath.Join("config", "certmanager", "kustomization.yaml"))).To(BeTrue())
			Expect(kbc.HasFile(filepath.Join("config", "webhook", "service.yaml"))).To(BeTrue())

			// Verify PROJECT has webhooks
			projectContent, err := os.ReadFile(filepath.Join(kbc.Dir, "PROJECT"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(projectContent)).To(ContainSubstring("defaulting: true"))
			Expect(string(projectContent)).To(ContainSubstring("validation: true"))

			By("trying to delete API (should fail - webhooks exist)")
			err = kbc.DeleteAPI(
				"--group", "crew",
				"--version", "v1",
				"--kind", "Captain",
				"--skip-confirmation",
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("webhooks are configured"))

			// API files should still exist
			Expect(kbc.HasFile(filepath.Join("api", "v1", "captain_types.go"))).To(BeTrue())

			By("deleting only the defaulting webhook")
			err = kbc.DeleteWebhook(
				"--group", "crew",
				"--version", "v1",
				"--kind", "Captain",
				"--defaulting",
				"--skip-confirmation",
			)
			Expect(err).NotTo(HaveOccurred())

			// Webhook files should still exist (validation remains)
			Expect(kbc.HasFile(filepath.Join("internal", "webhook", "v1", "captain_webhook.go"))).To(BeTrue())

			// PROJECT should have only validation
			projectContent, err = os.ReadFile(filepath.Join(kbc.Dir, "PROJECT"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(projectContent)).NotTo(ContainSubstring("defaulting: true"))
			Expect(string(projectContent)).To(ContainSubstring("validation: true"))

			By("deleting the validation webhook (last webhook)")
			err = kbc.DeleteWebhook(
				"--group", "crew",
				"--version", "v1",
				"--kind", "Captain",
				"--programmatic-validation",
				"--skip-confirmation",
			)
			Expect(err).NotTo(HaveOccurred())

			// Webhook files should now be deleted
			Expect(kbc.HasFile(filepath.Join("internal", "webhook", "v1", "captain_webhook.go"))).To(BeFalse())

			// Certmanager and webhook directories should be deleted
			Expect(kbc.HasFile(filepath.Join("config", "certmanager", "kustomization.yaml"))).To(BeFalse())
			Expect(kbc.HasFile(filepath.Join("config", "webhook", "service.yaml"))).To(BeFalse())

			// PROJECT should have no webhooks
			projectContent, err = os.ReadFile(filepath.Join(kbc.Dir, "PROJECT"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(projectContent)).NotTo(ContainSubstring("webhooks:"))

			By("deleting the API (should work now)")
			err = kbc.DeleteAPI(
				"--group", "crew",
				"--version", "v1",
				"--kind", "Captain",
				"--skip-confirmation",
			)
			Expect(err).NotTo(HaveOccurred())

			// API files should be deleted
			Expect(kbc.HasFile(filepath.Join("api", "v1", "captain_types.go"))).To(BeFalse())
			Expect(kbc.HasFile(filepath.Join("internal", "controller", "captain_controller.go"))).To(BeFalse())

			// PROJECT should have no Captain resource
			projectContent, err = os.ReadFile(filepath.Join(kbc.Dir, "PROJECT"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(projectContent)).NotTo(ContainSubstring("kind: Captain"))
		})

		It("should delete all webhook types at once when no flags specified", func() {
			By("initializing a project")
			err := kbc.Init(
				"--plugins", "go/v4",
				"--domain", kbc.Domain,
				"--skip-go-version-check",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating an API with webhook")
			err = kbc.CreateAPI(
				"--group", "crew",
				"--version", "v1",
				"--kind", "Sailor",
				"--resource",
				"--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			err = kbc.CreateWebhook(
				"--group", "crew",
				"--version", "v1",
				"--kind", "Sailor",
				"--defaulting",
				"--programmatic-validation",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("deleting all webhooks (no type flags)")
			err = kbc.DeleteWebhook(
				"--group", "crew",
				"--version", "v1",
				"--kind", "Sailor",
				"--skip-confirmation",
			)
			Expect(err).NotTo(HaveOccurred())

			// All webhook files should be deleted
			Expect(kbc.HasFile(filepath.Join("internal", "webhook", "v1", "sailor_webhook.go"))).To(BeFalse())

			// PROJECT should have no webhooks
			projectContent, err := os.ReadFile(filepath.Join(kbc.Dir, "PROJECT"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(projectContent)).NotTo(ContainSubstring("webhooks:"))
		})

		It("should handle multigroup layout correctly", func() {
			By("initializing a project with multigroup")
			err := kbc.Init(
				"--plugins", "go/v4",
				"--domain", kbc.Domain,
				"--skip-go-version-check",
			)
			Expect(err).NotTo(HaveOccurred())

			err = kbc.Edit(
				"--multigroup",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating an API in multigroup layout")
			err = kbc.CreateAPI(
				"--group", "crew",
				"--version", "v1",
				"--kind", "Bosun",
				"--resource",
				"--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			// Verify multigroup structure
			Expect(kbc.HasFile(filepath.Join("api", "crew", "v1", "bosun_types.go"))).To(BeTrue())

			By("creating a webhook in multigroup layout")
			err = kbc.CreateWebhook(
				"--group", "crew",
				"--version", "v1",
				"--kind", "Bosun",
				"--defaulting",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			Expect(kbc.HasFile(filepath.Join("internal", "webhook", "crew", "v1", "bosun_webhook.go"))).To(BeTrue())

			By("deleting the webhook in multigroup layout")
			err = kbc.DeleteWebhook(
				"--group", "crew",
				"--version", "v1",
				"--kind", "Bosun",
				"--skip-confirmation",
			)
			Expect(err).NotTo(HaveOccurred())

			Expect(kbc.HasFile(filepath.Join("internal", "webhook", "crew", "v1", "bosun_webhook.go"))).To(BeFalse())

			By("deleting the API in multigroup layout")
			err = kbc.DeleteAPI(
				"--group", "crew",
				"--version", "v1",
				"--kind", "Bosun",
				"--skip-confirmation",
			)
			Expect(err).NotTo(HaveOccurred())

			Expect(kbc.HasFile(filepath.Join("api", "crew", "v1", "bosun_types.go"))).To(BeFalse())
		})

		It("should handle multiple resources with webhooks correctly", func() {
			By("initializing a project")
			err := kbc.Init(
				"--plugins", "go/v4",
				"--domain", kbc.Domain,
				"--skip-go-version-check",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating two APIs with webhooks")
			err = kbc.CreateAPI(
				"--group", "crew",
				"--version", "v1",
				"--kind", "First",
				"--resource",
				"--controller=false",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			err = kbc.CreateWebhook(
				"--group", "crew",
				"--version", "v1",
				"--kind", "First",
				"--defaulting",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			err = kbc.CreateAPI(
				"--group", "crew",
				"--version", "v1",
				"--kind", "Second",
				"--resource",
				"--controller=false",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			err = kbc.CreateWebhook(
				"--group", "crew",
				"--version", "v1",
				"--kind", "Second",
				"--programmatic-validation",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("deleting First's webhook")
			err = kbc.DeleteWebhook(
				"--group", "crew",
				"--version", "v1",
				"--kind", "First",
				"--skip-confirmation",
			)
			Expect(err).NotTo(HaveOccurred())

			// certmanager should still exist (Second has webhook)
			Expect(kbc.HasFile(filepath.Join("config", "certmanager", "kustomization.yaml"))).To(BeTrue(),
				"certmanager should remain when other webhooks exist")

			By("deleting Second's webhook (last webhook)")
			err = kbc.DeleteWebhook(
				"--group", "crew",
				"--version", "v1",
				"--kind", "Second",
				"--skip-confirmation",
			)
			Expect(err).NotTo(HaveOccurred())

			// certmanager should now be deleted
			Expect(kbc.HasFile(filepath.Join("config", "certmanager", "kustomization.yaml"))).To(BeFalse(),
				"certmanager should be deleted with last webhook")

			// Verify kustomization.yaml has webhook lines commented
			kustomizeContent, err := os.ReadFile(filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(kustomizeContent)).NotTo(ContainSubstring("- ../webhook\n"),
				"webhook should be commented or removed")
		})
	})
})

