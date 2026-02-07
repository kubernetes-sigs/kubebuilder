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

	It("should comprehensively test all delete scenarios with state verification", func() {
		By("Scenario 1: Error conditions")
		err := kbc.DeleteAPI("--group", "none", "--version", "v1", "--kind", "Missing", "-y")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("does not exist"))

		By("Scenario 2: API deletion protection when webhooks exist")
		err = kbc.CreateAPI("--group", "test", "--version", "v1", "--kind", "Protected",
			"--resource", "--controller", "--make=false")
		Expect(err).NotTo(HaveOccurred())

		err = kbc.CreateWebhook("--group", "test", "--version", "v1", "--kind", "Protected",
			"--defaulting", "--programmatic-validation", "--make=false")
		Expect(err).NotTo(HaveOccurred())

		err = kbc.DeleteAPI("--group", "test", "--version", "v1", "--kind", "Protected", "-y")
		Expect(err).To(HaveOccurred(), "API deletion should be blocked")
		Expect(err.Error()).To(ContainSubstring("webhooks are configured"))

		By("Scenario 3: Granular webhook deletion and PROJECT updates")
		projectPath := filepath.Join(kbc.Dir, "PROJECT")

		err = kbc.DeleteWebhook("--group", "test", "--version", "v1", "--kind", "Protected",
			"--defaulting", "-y")
		Expect(err).NotTo(HaveOccurred())

		projectContent, _ := os.ReadFile(projectPath)
		Expect(string(projectContent)).NotTo(ContainSubstring("defaulting: true"))
		Expect(string(projectContent)).To(ContainSubstring("validation: true"))
		Expect(kbc.HasFile("internal/webhook/v1/protected_webhook.go")).To(BeTrue(),
			"files remain when validation exists")

		err = kbc.DeleteWebhook("--group", "test", "--version", "v1", "--kind", "Protected",
			"--programmatic-validation", "-y")
		Expect(err).NotTo(HaveOccurred())

		Expect(kbc.HasFile("internal/webhook/v1/protected_webhook.go")).To(BeFalse(),
			"files deleted when all types removed")
		Expect(kbc.HasFile("config/certmanager/kustomization.yaml")).To(BeFalse(),
			"certmanager deleted with last webhook")

		err = kbc.DeleteAPI("--group", "test", "--version", "v1", "--kind", "Protected", "-y")
		Expect(err).NotTo(HaveOccurred())

		By("Scenario 4: Error when deleting non-existent webhook type")
		err = kbc.CreateAPI("--group", "test", "--version", "v1", "--kind", "NoWebhook",
			"--resource", "--controller", "--make=false")
		Expect(err).NotTo(HaveOccurred())

		err = kbc.DeleteWebhook("--group", "test", "--version", "v1", "--kind", "NoWebhook", "-y")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("does not have any webhooks"))

		err = kbc.CreateWebhook("--group", "test", "--version", "v1", "--kind", "NoWebhook",
			"--defaulting", "--make=false")
		Expect(err).NotTo(HaveOccurred())

		err = kbc.DeleteWebhook("--group", "test", "--version", "v1", "--kind", "NoWebhook",
			"--programmatic-validation", "-y")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("does not have a validation webhook"))
	})

	It("should verify all API deletion scenarios with exact state restoration", func() {
		By("Scenario 1: Simple API - before equals after delete")
		By("capturing initial state before creating API")
		mainPathBefore := filepath.Join(kbc.Dir, "cmd", "main.go")

		mainContentBefore, err := os.ReadFile(mainPathBefore)
		Expect(err).NotTo(HaveOccurred())

		By("verifying baseline is clean (no pre-existing scaffolded code)")
		Expect(string(mainContentBefore)).NotTo(ContainSubstring("crewv1 \""),
			"baseline main.go should not contain crew/v1 imports")
		Expect(string(mainContentBefore)).NotTo(ContainSubstring("SailorReconciler"),
			"baseline main.go should not contain SailorReconciler")

		By("creating an API with controller")
		err = kbc.CreateAPI(
			"--group", "crew",
			"--version", "v1",
			"--kind", "Sailor",
			"--resource",
			"--controller",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying code was added to main.go")
		mainContentAfterCreate, err := os.ReadFile(mainPathBefore)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainContentAfterCreate)).To(ContainSubstring("crewv1 \""),
			"import should be added")
		Expect(string(mainContentAfterCreate)).To(ContainSubstring("utilruntime.Must(crewv1.AddToScheme(scheme))"),
			"AddToScheme should be added")
		Expect(string(mainContentAfterCreate)).To(ContainSubstring("&controller.SailorReconciler{"),
			"controller setup should be added")

		By("verifying PROJECT file was updated")
		projectPath := filepath.Join(kbc.Dir, "PROJECT")
		projectContentAfterCreate, err := os.ReadFile(projectPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectContentAfterCreate)).To(ContainSubstring("kind: Sailor"),
			"resource should be in PROJECT")

		By("verifying files were created")
		Expect(kbc.HasFile("api/v1/sailor_types.go")).To(BeTrue())
		Expect(kbc.HasFile("internal/controller/sailor_controller.go")).To(BeTrue())
		Expect(kbc.HasFile("internal/controller/suite_test.go")).To(BeTrue())
		Expect(kbc.HasFile("config/samples/crew_v1_sailor.yaml")).To(BeTrue())
		Expect(kbc.HasFile("config/rbac/sailor_admin_role.yaml")).To(BeTrue())

		By("deleting the API")
		err = kbc.DeleteAPI(
			"--group", "crew",
			"--version", "v1",
			"--kind", "Sailor",
			"-y",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying code was REMOVED from main.go")
		mainContentAfterDelete, err := os.ReadFile(mainPathBefore)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainContentAfterDelete)).NotTo(ContainSubstring("crewv1 \""),
			"import should be removed")
		Expect(string(mainContentAfterDelete)).NotTo(ContainSubstring("utilruntime.Must(crewv1.AddToScheme(scheme))"),
			"AddToScheme should be removed")
		Expect(string(mainContentAfterDelete)).NotTo(ContainSubstring("&controller.SailorReconciler{"),
			"controller setup should be removed")

		By("verifying PROJECT file was updated")
		projectContentAfterDelete, err := os.ReadFile(projectPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectContentAfterDelete)).NotTo(ContainSubstring("kind: Sailor"),
			"resource should be removed from PROJECT")

		By("verifying files were deleted")
		Expect(kbc.HasFile("api/v1/sailor_types.go")).To(BeFalse())
		Expect(kbc.HasFile("internal/controller/sailor_controller.go")).To(BeFalse())
		Expect(kbc.HasFile("config/samples/crew_v1_sailor.yaml")).To(BeFalse())
		Expect(kbc.HasFile("config/rbac/sailor_admin_role.yaml")).To(BeFalse())

		By("verifying main.go matches initial state")
		Expect(string(mainContentAfterDelete)).To(Equal(string(mainContentBefore)),
			"main.go after delete should exactly match before create")

		By("verifying PROJECT file matches initial state (excluding layout version)")
		// PROJECT may have minor formatting differences, but resource list should match
		Expect(string(projectContentAfterDelete)).NotTo(ContainSubstring("resources:"),
			"PROJECT should have no resources after deleting last API")
	})

	It("should completely undo create webhook operation - before equals after delete", func() {
		By("creating API first (required for webhook)")
		err := kbc.CreateAPI(
			"--group", "crew",
			"--version", "v1",
			"--kind", "Pilot",
			"--resource",
			"--controller=false",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("capturing state before creating webhook")
		mainPath := filepath.Join(kbc.Dir, "cmd", "main.go")
		mainContentBefore, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())

		By("creating webhook")
		err = kbc.CreateWebhook(
			"--group", "crew",
			"--version", "v1",
			"--kind", "Pilot",
			"--defaulting",
			"--programmatic-validation",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying webhook code was added to main.go")
		mainContentAfterCreate, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainContentAfterCreate)).To(ContainSubstring("webhookv1 \""),
			"webhook import should be added")
		Expect(string(mainContentAfterCreate)).To(ContainSubstring("webhookv1.SetupPilotWebhookWithManager"),
			"webhook setup should be added")

		By("verifying webhook files were created")
		Expect(kbc.HasFile("internal/webhook/v1/pilot_webhook.go")).To(BeTrue())
		Expect(kbc.HasFile("config/webhook/service.yaml")).To(BeTrue())
		Expect(kbc.HasFile("config/certmanager/kustomization.yaml")).To(BeTrue())

		By("deleting the webhook")
		err = kbc.DeleteWebhook(
			"--group", "crew",
			"--version", "v1",
			"--kind", "Pilot",
			"-y",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying webhook code was REMOVED from main.go")
		mainContentAfterDelete, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainContentAfterDelete)).NotTo(ContainSubstring("webhookv1 \""),
			"webhook import should be removed")
		Expect(string(mainContentAfterDelete)).NotTo(ContainSubstring("webhookv1.SetupPilotWebhookWithManager"),
			"webhook setup should be removed")

		By("verifying main.go matches state before webhook was created")
		Expect(string(mainContentAfterDelete)).To(Equal(string(mainContentBefore)),
			"main.go after delete webhook should exactly match before create webhook")

		By("verifying webhook files were deleted")
		Expect(kbc.HasFile("internal/webhook/v1/pilot_webhook.go")).To(BeFalse())
		Expect(kbc.HasFile("config/webhook/service.yaml")).To(BeFalse())
		Expect(kbc.HasFile("config/certmanager/kustomization.yaml")).To(BeFalse())

		By("verifying can recreate the same webhook (project is clean)")
		err = kbc.CreateWebhook(
			"--group", "crew",
			"--version", "v1",
			"--kind", "Pilot",
			"--defaulting",
			"--programmatic-validation",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred(), "should be able to recreate the same webhook")
		Expect(kbc.HasFile("internal/webhook/v1/pilot_webhook.go")).To(BeTrue())
	})

	It("should handle complete create-delete cycle with multiple APIs", func() {
		By("capturing initial state")
		mainPath := filepath.Join(kbc.Dir, "cmd", "main.go")
		crdKustomizePath := filepath.Join(kbc.Dir, "config", "crd", "kustomization.yaml")
		samplesKustomizePath := filepath.Join(kbc.Dir, "config", "samples", "kustomization.yaml")
		rbacKustomizePath := filepath.Join(kbc.Dir, "config", "rbac", "kustomization.yaml")

		mainContentBefore, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())

		By("creating three APIs")
		err = kbc.CreateAPI("--group", "crew", "--version", "v1", "--kind", "First",
			"--resource", "--controller", "--make=false")
		Expect(err).NotTo(HaveOccurred())

		err = kbc.CreateAPI("--group", "crew", "--version", "v1", "--kind", "Second",
			"--resource", "--controller", "--make=false")
		Expect(err).NotTo(HaveOccurred())

		err = kbc.CreateAPI("--group", "crew", "--version", "v1", "--kind", "Third",
			"--resource", "--controller", "--make=false")
		Expect(err).NotTo(HaveOccurred())

		By("verifying all three APIs in kustomization files")
		crdContent, _ := os.ReadFile(crdKustomizePath)
		Expect(string(crdContent)).To(ContainSubstring("crew." + kbc.Domain + "_firsts.yaml"))
		Expect(string(crdContent)).To(ContainSubstring("crew." + kbc.Domain + "_seconds.yaml"))
		Expect(string(crdContent)).To(ContainSubstring("crew." + kbc.Domain + "_thirds.yaml"))

		By("deleting second API (middle)")
		err = kbc.DeleteAPI("--group", "crew", "--version", "v1", "--kind", "Second",
			"-y")
		Expect(err).NotTo(HaveOccurred())

		By("verifying only Second was removed from kustomization")
		crdContent, _ = os.ReadFile(crdKustomizePath)
		Expect(string(crdContent)).To(ContainSubstring("crew." + kbc.Domain + "_firsts.yaml"))
		Expect(string(crdContent)).NotTo(ContainSubstring("crew." + kbc.Domain + "_seconds.yaml"))
		Expect(string(crdContent)).To(ContainSubstring("crew." + kbc.Domain + "_thirds.yaml"))

		samplesContent, _ := os.ReadFile(samplesKustomizePath)
		Expect(string(samplesContent)).To(ContainSubstring("crew_v1_first.yaml"))
		Expect(string(samplesContent)).NotTo(ContainSubstring("crew_v1_second.yaml"))
		Expect(string(samplesContent)).To(ContainSubstring("crew_v1_third.yaml"))

		rbacContent, _ := os.ReadFile(rbacKustomizePath)
		Expect(string(rbacContent)).To(ContainSubstring("first_admin_role.yaml"))
		Expect(string(rbacContent)).NotTo(ContainSubstring("second_admin_role.yaml"))
		Expect(string(rbacContent)).To(ContainSubstring("third_admin_role.yaml"))

		By("verifying Second's code removed from main.go but First and Third remain")
		mainContent, _ := os.ReadFile(mainPath)
		// All three APIs share same version, so import remains (shared)
		Expect(string(mainContent)).To(ContainSubstring("crewv1 \""))
		// But controller setups should be specific
		Expect(string(mainContent)).To(ContainSubstring("FirstReconciler"))
		Expect(string(mainContent)).NotTo(ContainSubstring("SecondReconciler"))
		Expect(string(mainContent)).To(ContainSubstring("ThirdReconciler"))

		By("deleting remaining APIs")
		err = kbc.DeleteAPI("--group", "crew", "--version", "v1", "--kind", "First",
			"-y")
		Expect(err).NotTo(HaveOccurred())

		err = kbc.DeleteAPI("--group", "crew", "--version", "v1", "--kind", "Third",
			"-y")
		Expect(err).NotTo(HaveOccurred())

		By("verifying all API code removed and main.go restored")
		mainContentFinal, _ := os.ReadFile(mainPath)
		Expect(string(mainContentFinal)).NotTo(ContainSubstring("crewv1 \""))
		Expect(string(mainContentFinal)).NotTo(ContainSubstring("FirstReconciler"))
		Expect(string(mainContentFinal)).NotTo(ContainSubstring("ThirdReconciler"))
		Expect(string(mainContentFinal)).To(Equal(string(mainContentBefore)),
			"main.go should match initial state after all APIs deleted")

		By("verifying kustomization files were deleted with last API")
		Expect(kbc.HasFile("config/crd/kustomization.yaml")).To(BeFalse())
		Expect(kbc.HasFile("config/samples/kustomization.yaml")).To(BeFalse())

		By("Scenario 3: Multigroup layout - delete undoes create")
		By("enabling multigroup")
		err = kbc.Edit("--multigroup")
		Expect(err).NotTo(HaveOccurred())

		By("capturing state before create")
		mainPathMulti := filepath.Join(kbc.Dir, "cmd", "main.go")
		mainContentBeforeMulti, err := os.ReadFile(mainPathMulti)
		Expect(err).NotTo(HaveOccurred())

		By("creating multigroup API")
		err = kbc.CreateAPI(
			"--group", "ship",
			"--version", "v1",
			"--kind", "Cruiser",
			"--resource",
			"--controller",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying multigroup paths")
		Expect(kbc.HasFile("api/ship/v1/cruiser_types.go")).To(BeTrue())
		Expect(kbc.HasFile("internal/controller/ship/cruiser_controller.go")).To(BeTrue())

		By("verifying multigroup imports in main.go")
		mainContent, err = os.ReadFile(mainPathMulti)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainContent)).To(ContainSubstring("shipv1 \""))
		Expect(string(mainContent)).To(ContainSubstring("shipcontroller \""))
		Expect(string(mainContent)).To(ContainSubstring("shipcontroller.CruiserReconciler"))

		By("deleting multigroup API")
		err = kbc.DeleteAPI(
			"--group", "ship",
			"--version", "v1",
			"--kind", "Cruiser",
			"-y",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying main.go restored to before state")
		mainContentAfter, err := os.ReadFile(mainPathMulti)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainContentAfter)).NotTo(ContainSubstring("shipv1 \""))
		Expect(string(mainContentAfter)).NotTo(ContainSubstring("shipcontroller \""))
		Expect(string(mainContentAfter)).To(Equal(string(mainContentBeforeMulti)),
			"multigroup main.go should match initial state")

		By("verifying files deleted")
		Expect(kbc.HasFile("api/ship/v1/cruiser_types.go")).To(BeFalse())
		Expect(kbc.HasFile("internal/controller/ship/cruiser_controller.go")).To(BeFalse())
	})

	It("should verify all webhook deletion scenarios with exact state restoration", func() {
		By("Scenario 1: Simple webhook - before equals after delete")
		By("creating API for conversion webhook")
		err := kbc.CreateAPI(
			"--group", "test",
			"--version", "v1",
			"--kind", "Converter",
			"--resource",
			"--controller=false",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("capturing CRD kustomization before conversion webhook")
		crdKustomizePath := filepath.Join(kbc.Dir, "config", "crd", "kustomization.yaml")
		crdContentBefore, err := os.ReadFile(crdKustomizePath)
		Expect(err).NotTo(HaveOccurred())

		By("creating conversion webhook")
		err = kbc.CreateWebhook(
			"--group", "test",
			"--version", "v1",
			"--kind", "Converter",
			"--conversion",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying conversion webhook patch was added")
		crdContentAfterCreate, err := os.ReadFile(crdKustomizePath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(crdContentAfterCreate)).To(ContainSubstring("patches/webhook_in_converters.yaml"),
			"conversion webhook patch should be added to kustomization")
		Expect(kbc.HasFile("config/crd/patches/webhook_in_converters.yaml")).To(BeTrue(),
			"conversion webhook patch file should exist")

		By("deleting conversion webhook")
		err = kbc.DeleteWebhook(
			"--group", "test",
			"--version", "v1",
			"--kind", "Converter",
			"-y",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying conversion webhook patch was removed")
		crdContentAfterDelete, err := os.ReadFile(crdKustomizePath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(crdContentAfterDelete)).NotTo(ContainSubstring("patches/webhook_in_converters.yaml"),
			"conversion webhook patch should be removed from kustomization")
		Expect(kbc.HasFile("config/crd/patches/webhook_in_converters.yaml")).To(BeFalse(),
			"conversion webhook patch file should be deleted")

		By("verifying CRD kustomization matches state before conversion webhook")
		Expect(string(crdContentAfterDelete)).To(Equal(string(crdContentBefore)),
			"CRD kustomization after delete should match before create webhook")

		By("Scenario 2: Conversion removal with other webhooks remaining")
		By("creating API")
		err = kbc.CreateAPI(
			"--group", "test",
			"--version", "v2",
			"--kind", "Ship",
			"--resource",
			"--controller=false",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("creating defaulting and validation webhooks first")
		err = kbc.CreateWebhook(
			"--group", "test",
			"--version", "v2",
			"--kind", "Ship",
			"--defaulting",
			"--programmatic-validation",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("saving state with defaulting and validation webhooks (baseline)")
		crdPath := filepath.Join(kbc.Dir, "config", "crd", "kustomization.yaml")
		mainPath := filepath.Join(kbc.Dir, "cmd", "main.go")
		projectPath := filepath.Join(kbc.Dir, "PROJECT")

		crdBaseline, err := os.ReadFile(crdPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(crdBaseline)).NotTo(ContainSubstring("webhook_in_ships.yaml"),
			"no conversion patch at baseline")

		mainBaseline, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())

		projectBaseline, err := os.ReadFile(projectPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectBaseline)).To(ContainSubstring("defaulting: true"))
		Expect(string(projectBaseline)).To(ContainSubstring("validation: true"))
		Expect(string(projectBaseline)).NotTo(ContainSubstring("conversion: true"))

		By("adding conversion webhook")
		err = kbc.CreateWebhook(
			"--group", "test",
			"--version", "v2",
			"--kind", "Ship",
			"--conversion",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying conversion patch was added")
		crdWithConversion, err := os.ReadFile(crdPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(crdWithConversion)).To(ContainSubstring("webhook_in_ships.yaml"))
		Expect(kbc.HasFile("config/crd/patches/webhook_in_ships.yaml")).To(BeTrue())

		projectWithConversion, err := os.ReadFile(projectPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectWithConversion)).To(ContainSubstring("conversion: true"))

		By("deleting ONLY conversion webhook (defaulting and validation remain)")
		err = kbc.DeleteWebhook(
			"--group", "test",
			"--version", "v2",
			"--kind", "Ship",
			"--conversion",
			"-y",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying conversion patch removed from kustomization")
		crdAfterDelete, err := os.ReadFile(crdPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(crdAfterDelete)).NotTo(ContainSubstring("webhook_in_ships.yaml"),
			"conversion patch must be removed")
		Expect(kbc.HasFile("config/crd/patches/webhook_in_ships.yaml")).To(BeFalse(),
			"patch file must be deleted")

		By("verifying PROJECT reflects only defaulting and validation")
		projectAfterDelete, err := os.ReadFile(projectPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectAfterDelete)).To(ContainSubstring("defaulting: true"))
		Expect(string(projectAfterDelete)).To(ContainSubstring("validation: true"))
		Expect(string(projectAfterDelete)).NotTo(ContainSubstring("conversion: true"))

		By("verifying CRD kustomization matches baseline (with defaulting/validation)")
		Expect(string(crdAfterDelete)).To(Equal(string(crdBaseline)),
			"CRD kustomization must match baseline state with defaulting/validation")

		By("verifying main.go unchanged (webhook import/setup remain for defaulting/validation)")
		mainAfterDelete, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainAfterDelete)).To(Equal(string(mainBaseline)),
			"main.go must match baseline (webhook code remains for other types)")

		By("verifying can recreate conversion webhook (project is clean)")
		err = kbc.CreateWebhook(
			"--group", "test",
			"--version", "v2",
			"--kind", "Ship",
			"--conversion",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred(), "should be able to recreate conversion webhook")
		Expect(kbc.HasFile("config/crd/patches/webhook_in_ships.yaml")).To(BeTrue())
	})

	It("should verify deploy-image plugin chain - baseline restored after deletion", func() {
		By("creating first API (regular, establishes baseline)")
		err := kbc.CreateAPI(
			"--group", "app",
			"--version", "v1",
			"--kind", "Database",
			"--resource",
			"--controller",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("capturing baseline state (with one regular API)")
		mainPath := filepath.Join(kbc.Dir, "cmd", "main.go")
		projectPath := filepath.Join(kbc.Dir, "PROJECT")
		crdKustomizePath := filepath.Join(kbc.Dir, "config", "crd", "kustomization.yaml")

		mainBaseline, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())

		projectBaseline, err := os.ReadFile(projectPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectBaseline)).To(ContainSubstring("kind: Database"))
		Expect(string(projectBaseline)).NotTo(ContainSubstring("deploy-image"))

		By("creating second API with deploy-image plugin")
		err = kbc.CreateAPI(
			"--group", "app",
			"--version", "v1alpha1",
			"--kind", "Cache",
			"--image", "redis:7-alpine",
			"--plugins", "deploy-image/v1-alpha",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying deploy-image API was created")
		Expect(kbc.HasFile("api/v1alpha1/cache_types.go")).To(BeTrue())
		Expect(kbc.HasFile("internal/controller/cache_controller.go")).To(BeTrue())

		projectWithCache, err := os.ReadFile(projectPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectWithCache)).To(ContainSubstring("kind: Cache"))
		Expect(string(projectWithCache)).To(ContainSubstring("deploy-image.go.kubebuilder.io/v1-alpha"))

		mainWithCache, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainWithCache)).To(ContainSubstring("appv1alpha1"))
		Expect(string(mainWithCache)).To(ContainSubstring("CacheReconciler"))

		By("deleting deploy-image API with plugin chain")
		err = kbc.DeleteAPI(
			"--group", "app",
			"--version", "v1alpha1",
			"--kind", "Cache",
			"--plugins", "deploy-image/v1-alpha",
			"-y",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying deploy-image API files deleted")
		Expect(kbc.HasFile("api/v1alpha1/cache_types.go")).To(BeFalse())
		Expect(kbc.HasFile("internal/controller/cache_controller.go")).To(BeFalse())

		By("verifying main.go restored to baseline")
		mainAfterDelete, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainAfterDelete)).To(Equal(string(mainBaseline)),
			"main.go must match baseline (only Database API remains)")

		By("verifying PROJECT restored to baseline")
		projectAfterDeleteCache, err := os.ReadFile(projectPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectAfterDeleteCache)).To(ContainSubstring("kind: Database"))
		Expect(string(projectAfterDeleteCache)).NotTo(ContainSubstring("kind: Cache"))
		Expect(string(projectAfterDeleteCache)).NotTo(ContainSubstring("deploy-image.go.kubebuilder.io/v1-alpha"))

		By("verifying CRD kustomization contains only Database")
		crdAfterDelete, err := os.ReadFile(crdKustomizePath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(crdAfterDelete)).To(ContainSubstring("databases.yaml"))
		Expect(string(crdAfterDelete)).NotTo(ContainSubstring("caches.yaml"))

		By("verifying can recreate the same API with deploy-image (project is clean)")
		err = kbc.CreateAPI(
			"--group", "app",
			"--version", "v1alpha1",
			"--kind", "Cache",
			"--image", "redis:7-alpine",
			"--plugins", "deploy-image/v1-alpha",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred(), "should be able to recreate deploy-image API")
		Expect(kbc.HasFile("api/v1alpha1/cache_types.go")).To(BeTrue())
	})

	It("should verify optional plugins deletion - baseline restored after adding and removing", func() {
		By("creating base API for manifests")
		err := kbc.CreateAPI(
			"--group", "app",
			"--version", "v1",
			"--kind", "Service",
			"--resource",
			"--controller",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("capturing baseline state (before optional plugins)")
		projectPath := filepath.Join(kbc.Dir, "PROJECT")

		projectBaseline, err := os.ReadFile(projectPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectBaseline)).NotTo(ContainSubstring("helm.kubebuilder.io"))
		Expect(string(projectBaseline)).NotTo(ContainSubstring("grafana.kubebuilder.io"))

		By("adding grafana plugin")
		err = kbc.Edit("--plugins=grafana/v1-alpha")
		Expect(err).NotTo(HaveOccurred())

		By("verifying grafana files created")
		Expect(kbc.HasFile("grafana/controller-runtime-metrics.json")).To(BeTrue())

		projectWithGrafana, err := os.ReadFile(projectPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectWithGrafana)).To(ContainSubstring("grafana.kubebuilder.io/v1-alpha"))

		By("adding helm plugin (need install.yaml first)")
		err = kbc.Make("build-installer")
		Expect(err).NotTo(HaveOccurred())

		err = kbc.Edit("--plugins=helm/v2-alpha")
		Expect(err).NotTo(HaveOccurred())

		By("verifying helm files created")
		Expect(kbc.HasFile("dist/chart/Chart.yaml")).To(BeTrue())
		Expect(kbc.HasFile("dist/chart/values.yaml")).To(BeTrue())

		projectWithBoth, err := os.ReadFile(projectPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectWithBoth)).To(ContainSubstring("grafana.kubebuilder.io/v1-alpha"))
		Expect(string(projectWithBoth)).To(ContainSubstring("helm.kubebuilder.io/v2-alpha"))

		By("removing grafana plugin")
		err = kbc.Delete("--plugins=grafana/v1-alpha")
		Expect(err).NotTo(HaveOccurred())

		By("verifying grafana files deleted")
		Expect(kbc.HasFile("grafana/controller-runtime-metrics.json")).To(BeFalse())
		Expect(kbc.HasFile("grafana")).To(BeFalse())

		projectWithoutGrafana, err := os.ReadFile(projectPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectWithoutGrafana)).NotTo(ContainSubstring("grafana.kubebuilder.io/v1-alpha"))
		Expect(string(projectWithoutGrafana)).To(ContainSubstring("helm.kubebuilder.io/v2-alpha"),
			"helm should still be present")

		By("removing helm plugin")
		err = kbc.Delete("--plugins=helm/v2-alpha")
		Expect(err).NotTo(HaveOccurred())

		By("verifying helm files deleted")
		Expect(kbc.HasFile("dist/chart/Chart.yaml")).To(BeFalse())
		Expect(kbc.HasFile("dist/chart")).To(BeFalse())
		Expect(kbc.HasFile(".github/workflows/test-chart.yml")).To(BeFalse())

		By("verifying PROJECT restored to baseline")
		projectFinal, err := os.ReadFile(projectPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectFinal)).NotTo(ContainSubstring("grafana.kubebuilder.io"))
		Expect(string(projectFinal)).NotTo(ContainSubstring("helm.kubebuilder.io"))
		Expect(string(projectFinal)).To(ContainSubstring("kind: Service"),
			"base API should remain")

		By("verifying can re-add plugins (project is clean)")
		err = kbc.Edit("--plugins=grafana/v1-alpha")
		Expect(err).NotTo(HaveOccurred(), "should be able to re-add grafana")
		Expect(kbc.HasFile("grafana/controller-runtime-metrics.json")).To(BeTrue())
	})

	It("should handle API without controller and controller without API scenarios", func() {
		By("Scenario 1: Delete resource-only API (no controller)")
		mainPath := filepath.Join(kbc.Dir, "cmd", "main.go")
		mainBefore, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())

		err = kbc.CreateAPI(
			"--group", "data",
			"--version", "v1",
			"--kind", "Schema",
			"--resource",
			"--controller=false",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying API files created but no controller")
		Expect(kbc.HasFile("api/v1/schema_types.go")).To(BeTrue())
		Expect(kbc.HasFile("internal/controller/schema_controller.go")).To(BeFalse())

		mainAfterCreate, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainAfterCreate)).To(ContainSubstring("datav1"))
		Expect(string(mainAfterCreate)).NotTo(ContainSubstring("SchemaReconciler"))

		By("deleting resource-only API")
		err = kbc.DeleteAPI("--group", "data", "--version", "v1", "--kind", "Schema", "-y")
		Expect(err).NotTo(HaveOccurred())

		By("verifying complete restoration")
		mainAfterDelete, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainAfterDelete)).To(Equal(string(mainBefore)),
			"main.go should match initial state")
		Expect(kbc.HasFile("api/v1/schema_types.go")).To(BeFalse())

		By("Scenario 2: Delete controller-only API (external resource)")
		err = kbc.CreateAPI(
			"--group", "cert-manager.io",
			"--version", "v1",
			"--kind", "Certificate",
			"--resource=false",
			"--controller",
			"--external-api-domain=cert-manager.io",
			"--external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying controller created without API types")
		Expect(kbc.HasFile("internal/controller/certificate_controller.go")).To(BeTrue())
		Expect(kbc.HasFile("api/v1/certificate_types.go")).To(BeFalse())

		mainWithExternal, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainWithExternal)).To(ContainSubstring("CertificateReconciler"))

		By("deleting external controller")
		err = kbc.DeleteAPI("--group", "cert-manager.io", "--version", "v1", "--kind", "Certificate", "-y")
		Expect(err).NotTo(HaveOccurred())

		By("verifying external controller removed")
		Expect(kbc.HasFile("internal/controller/certificate_controller.go")).To(BeFalse())
		mainFinal, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainFinal)).NotTo(ContainSubstring("CertificateReconciler"))
	})

	It("should handle multigroup scenarios correctly", func() {
		By("enabling multigroup")
		err := kbc.Edit("--multigroup")
		Expect(err).NotTo(HaveOccurred())

		mainPath := filepath.Join(kbc.Dir, "cmd", "main.go")
		projectPath := filepath.Join(kbc.Dir, "PROJECT")

		By("creating APIs in different groups")
		err = kbc.CreateAPI(
			"--group", "crew",
			"--version", "v1",
			"--kind", "Captain",
			"--resource",
			"--controller",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		err = kbc.CreateAPI(
			"--group", "ship",
			"--version", "v1",
			"--kind", "Frigate",
			"--resource",
			"--controller",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying multigroup structure")
		Expect(kbc.HasFile("api/crew/v1/captain_types.go")).To(BeTrue())
		Expect(kbc.HasFile("api/ship/v1/frigate_types.go")).To(BeTrue())
		Expect(kbc.HasFile("internal/controller/crew/captain_controller.go")).To(BeTrue())
		Expect(kbc.HasFile("internal/controller/ship/frigate_controller.go")).To(BeTrue())

		mainWithBoth, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainWithBoth)).To(ContainSubstring("crewv1"))
		Expect(string(mainWithBoth)).To(ContainSubstring("shipv1"))
		Expect(string(mainWithBoth)).To(ContainSubstring("crewcontroller"))
		Expect(string(mainWithBoth)).To(ContainSubstring("shipcontroller"))

		By("adding second API in crew group (shared import test)")
		err = kbc.CreateAPI(
			"--group", "crew",
			"--version", "v1",
			"--kind", "Sailor",
			"--resource",
			"--controller",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("deleting one crew API (import should remain for other)")
		err = kbc.DeleteAPI("--group", "crew", "--version", "v1", "--kind", "Sailor", "-y")
		Expect(err).NotTo(HaveOccurred())

		mainAfterDelete, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainAfterDelete)).To(ContainSubstring("crewv1"),
			"shared import should remain for Captain")
		Expect(string(mainAfterDelete)).To(ContainSubstring("crewcontroller"),
			"shared controller import should remain")
		Expect(string(mainAfterDelete)).To(ContainSubstring("CaptainReconciler"))
		Expect(string(mainAfterDelete)).NotTo(ContainSubstring("SailorReconciler"))

		By("deleting remaining crew API")
		err = kbc.DeleteAPI("--group", "crew", "--version", "v1", "--kind", "Captain", "-y")
		Expect(err).NotTo(HaveOccurred())

		mainAfterAllCrew, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainAfterAllCrew)).NotTo(ContainSubstring("crewv1"))
		Expect(string(mainAfterAllCrew)).NotTo(ContainSubstring("crewcontroller"))
		Expect(string(mainAfterAllCrew)).To(ContainSubstring("shipv1"),
			"ship group should remain")

		By("verifying group directories cleaned up")
		Expect(kbc.HasFile("api/crew")).To(BeFalse())
		Expect(kbc.HasFile("internal/controller/crew")).To(BeFalse())
		Expect(kbc.HasFile("api/ship/v1/frigate_types.go")).To(BeTrue(),
			"ship group should remain")

		By("deleting last API")
		err = kbc.DeleteAPI("--group", "ship", "--version", "v1", "--kind", "Frigate", "-y")
		Expect(err).NotTo(HaveOccurred())

		projectFinal, err := os.ReadFile(projectPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectFinal)).To(ContainSubstring("multigroup: true"),
			"multigroup setting should remain")
		Expect(string(projectFinal)).NotTo(ContainSubstring("resources:"),
			"all resources should be removed")
	})

	It("should handle webhooks on resource-only APIs correctly", func() {
		mainPath := filepath.Join(kbc.Dir, "cmd", "main.go")
		mainBefore, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())

		By("creating resource without controller")
		err = kbc.CreateAPI(
			"--group", "config",
			"--version", "v1",
			"--kind", "Settings",
			"--resource",
			"--controller=false",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("adding webhooks to resource-only API")
		err = kbc.CreateWebhook(
			"--group", "config",
			"--version", "v1",
			"--kind", "Settings",
			"--defaulting",
			"--programmatic-validation",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying webhook files created without controller")
		Expect(kbc.HasFile("internal/webhook/v1/settings_webhook.go")).To(BeTrue())
		Expect(kbc.HasFile("internal/controller/settings_controller.go")).To(BeFalse())

		By("deleting all webhooks first")
		err = kbc.DeleteWebhook("--group", "config", "--version", "v1", "--kind", "Settings", "-y")
		Expect(err).NotTo(HaveOccurred())

		By("deleting resource-only API")
		err = kbc.DeleteAPI("--group", "config", "--version", "v1", "--kind", "Settings", "-y")
		Expect(err).NotTo(HaveOccurred())

		By("verifying complete restoration to baseline")
		mainAfterDelete, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainAfterDelete)).To(Equal(string(mainBefore)),
			"main.go should match initial baseline")
	})

	It("should handle multiple versions of same group correctly", func() {
		mainPath := filepath.Join(kbc.Dir, "cmd", "main.go")

		By("creating v1 API")
		err := kbc.CreateAPI(
			"--group", "app",
			"--version", "v1",
			"--kind", "Database",
			"--resource",
			"--controller",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("creating v1alpha1 API of same group")
		err = kbc.CreateAPI(
			"--group", "app",
			"--version", "v1alpha1",
			"--kind", "Cache",
			"--resource",
			"--controller",
			"--make=false",
		)
		Expect(err).NotTo(HaveOccurred())

		By("verifying both versions exist")
		Expect(kbc.HasFile("api/v1/database_types.go")).To(BeTrue())
		Expect(kbc.HasFile("api/v1alpha1/cache_types.go")).To(BeTrue())

		mainWithBoth, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainWithBoth)).To(ContainSubstring("appv1 "))
		Expect(string(mainWithBoth)).To(ContainSubstring("appv1alpha1 "))

		By("deleting v1alpha1 (v1 should remain)")
		err = kbc.DeleteAPI("--group", "app", "--version", "v1alpha1", "--kind", "Cache", "-y")
		Expect(err).NotTo(HaveOccurred())

		By("verifying v1 remains, v1alpha1 removed")
		Expect(kbc.HasFile("api/v1/database_types.go")).To(BeTrue())
		Expect(kbc.HasFile("api/v1alpha1/cache_types.go")).To(BeFalse())

		mainAfterDelete, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainAfterDelete)).To(ContainSubstring("appv1 "))
		Expect(string(mainAfterDelete)).NotTo(ContainSubstring("appv1alpha1"))

		By("deleting v1")
		err = kbc.DeleteAPI("--group", "app", "--version", "v1", "--kind", "Database", "-y")
		Expect(err).NotTo(HaveOccurred())

		By("verifying all API types removed")
		Expect(kbc.HasFile("api/v1/database_types.go")).To(BeFalse())
		Expect(kbc.HasFile("api/v1alpha1/cache_types.go")).To(BeFalse())
		Expect(kbc.HasFile("internal/controller/database_controller.go")).To(BeFalse())
		Expect(kbc.HasFile("internal/controller/cache_controller.go")).To(BeFalse())

		mainFinal, err := os.ReadFile(mainPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(mainFinal)).NotTo(ContainSubstring("appv1 "))
		Expect(string(mainFinal)).NotTo(ContainSubstring("appv1alpha1"))
		Expect(string(mainFinal)).NotTo(ContainSubstring("DatabaseReconciler"))
		Expect(string(mainFinal)).NotTo(ContainSubstring("CacheReconciler"))
	})
})
