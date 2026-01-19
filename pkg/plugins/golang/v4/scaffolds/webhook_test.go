//go:build integration

/*
Copyright 2026 The Kubernetes Authors.

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

package scaffolds

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("Webhook Incremental Scaffolding", func() {
	var (
		kbc *utils.TestContext
	)

	BeforeEach(func() {
		var err error
		kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(kbc.Prepare()).To(Succeed())

		By("initializing a project")
		err = kbc.Init(
			"--domain", "test.io",
			"--repo", "test.io/webhooktest",
		)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		By("removing working directory")
		kbc.Destroy()
	})

	Context("When creating webhooks incrementally", func() {
		It("should support adding validation to existing defaulting webhook", func() {
			By("creating an API")
			err := kbc.CreateAPI(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestIncremental",
				"--resource", "--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating defaulting webhook")
			err = kbc.CreateWebhook(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestIncremental",
				"--defaulting",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("adding validation webhook WITHOUT --force")
			err = kbc.CreateWebhook(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestIncremental",
				"--programmatic-validation",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying both webhooks are present in test file")
			testFile := filepath.Join(kbc.Dir, "internal/webhook/v1/testincremental_webhook_test.go")
			content, err := os.ReadFile(testFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring("defaulter TestIncrementalCustomDefaulter"))
			Expect(string(content)).To(ContainSubstring("validator TestIncrementalCustomValidator"))
			Expect(string(content)).To(ContainSubstring("Context(\"When creating TestIncremental under Defaulting Webhook\""))
			Expect(string(content)).To(ContainSubstring("Context(\"When creating or updating TestIncremental under Validating Webhook\""))
		})

		It("should support adding defaulting to existing validation webhook", func() {
			By("creating an API")
			err := kbc.CreateAPI(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestReverse",
				"--resource", "--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating validation webhook")
			err = kbc.CreateWebhook(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestReverse",
				"--programmatic-validation",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("adding defaulting webhook WITHOUT --force")
			err = kbc.CreateWebhook(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestReverse",
				"--defaulting",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying both webhooks are present")
			testFile := filepath.Join(kbc.Dir, "internal/webhook/v1/testreverse_webhook_test.go")
			content, err := os.ReadFile(testFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring("validator TestReverseCustomValidator"))
			Expect(string(content)).To(ContainSubstring("defaulter TestReverseCustomDefaulter"))
		})

		It("should support conversion-only webhooks without defaulting/validation", func() {
			By("creating API v1")
			err := kbc.CreateAPI(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestConversion",
				"--resource", "--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating API v2")
			err = kbc.CreateAPI(
				"--group", "test",
				"--version", "v2",
				"--kind", "TestConversion",
				"--resource=false", "--controller=false",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating conversion webhook")
			err = kbc.CreateWebhook(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestConversion",
				"--conversion",
				"--spoke", "v2",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying conversion test context is created")
			testFile := filepath.Join(kbc.Dir, "internal/webhook/v1/testconversion_webhook_test.go")
			content, err := os.ReadFile(testFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring("Context(\"When creating TestConversion under Conversion Webhook\""))
			Expect(string(content)).NotTo(ContainSubstring("defaulter"))
			Expect(string(content)).NotTo(ContainSubstring("validator"))

		By("verifying webhook file was created with minimal setup")
		webhookFile := filepath.Join(kbc.Dir, "internal/webhook/v1/testconversion_webhook.go")
		webhookContent, err := os.ReadFile(webhookFile)
		Expect(err).NotTo(HaveOccurred())

		Expect(string(webhookContent)).To(ContainSubstring("SetupTestConversionWebhookWithManager"))
		Expect(string(webhookContent)).To(ContainSubstring("NewWebhookManagedBy(mgr, &testv1.TestConversion{})"))
		Expect(string(webhookContent)).To(ContainSubstring("Complete()"))
		Expect(string(webhookContent)).NotTo(ContainSubstring("CustomDefaulter"))
		Expect(string(webhookContent)).NotTo(ContainSubstring("CustomValidator"))

			By("verifying conversion webhook IS wired in main.go")
			mainFile := filepath.Join(kbc.Dir, "cmd/main.go")
			mainContent, err := os.ReadFile(mainFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(mainContent)).To(ContainSubstring("SetupTestConversionWebhookWithManager"))

			By("verifying e2e test has conversion CA injection check")
			e2eFile := filepath.Join(kbc.Dir, "test/e2e/e2e_test.go")
			e2eContent, err := os.ReadFile(e2eFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(e2eContent)).To(ContainSubstring("CA injection for TestConversion conversion webhook"))

			By("verifying webhook suite test was created")
			suiteFile := filepath.Join(kbc.Dir, "internal/webhook/v1/webhook_suite_test.go")
			_, err = os.Stat(suiteFile)
			Expect(err).NotTo(HaveOccurred(), "Webhook suite test file should exist")
		})

		It("should support multiversion scenario: conversion then defaulting/validation", func() {
			By("creating API v1")
			err := kbc.CreateAPI(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestMulti",
				"--resource", "--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating API v2")
			err = kbc.CreateAPI(
				"--group", "test",
				"--version", "v2",
				"--kind", "TestMulti",
				"--resource=false", "--controller=false",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating conversion webhook first")
			err = kbc.CreateWebhook(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestMulti",
				"--conversion",
				"--spoke", "v2",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("adding defaulting and validation webhooks WITHOUT --force")
			err = kbc.CreateWebhook(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestMulti",
				"--defaulting",
				"--programmatic-validation",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying all three webhook types are present")
			testFile := filepath.Join(kbc.Dir, "internal/webhook/v1/testmulti_webhook_test.go")
			content, err := os.ReadFile(testFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring("defaulter TestMultiCustomDefaulter"))
			Expect(string(content)).To(ContainSubstring("validator TestMultiCustomValidator"))
			Expect(string(content)).To(ContainSubstring("Context(\"When creating TestMulti under Conversion Webhook\""))
			Expect(string(content)).To(ContainSubstring("Context(\"When creating TestMulti under Defaulting Webhook\""))
			Expect(string(content)).To(ContainSubstring("Context(\"When creating or updating TestMulti under Validating Webhook\""))

			By("verifying defaulting/validation webhooks are wired in main.go")
			mainFile := filepath.Join(kbc.Dir, "cmd/main.go")
			mainContent, err := os.ReadFile(mainFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(mainContent)).To(ContainSubstring("SetupTestMultiWebhookWithManager"))
		})
	})

	Context("When user customizes webhook files", func() {
		It("should preserve customizations when adding new webhook types", func() {
			By("creating an API")
			err := kbc.CreateAPI(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestCustom",
				"--resource", "--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating defaulting webhook")
			err = kbc.CreateWebhook(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestCustom",
				"--defaulting",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("simulating user customizations to test file")
			testFile := filepath.Join(kbc.Dir, "internal/webhook/v1/testcustom_webhook_test.go")
			testContent, err := os.ReadFile(testFile)
			Expect(err).NotTo(HaveOccurred())

			By("customizing: renaming variables obj->myObj, oldObj->oldMyObj")
			modified := strings.ReplaceAll(string(testContent), "obj       *testv1.TestCustom", "myObj     *testv1.TestCustom")
			modified = strings.ReplaceAll(modified, "oldObj    *testv1.TestCustom", "oldMyObj  *testv1.TestCustom")
			modified = strings.ReplaceAll(modified, "obj = &testv1.TestCustom{}", "myObj = &testv1.TestCustom{}")
			modified = strings.ReplaceAll(modified, "oldObj = &testv1.TestCustom{}", "oldMyObj = &testv1.TestCustom{}")
			modified = strings.ReplaceAll(modified, "Expect(oldObj)", "Expect(oldMyObj)")
			modified = strings.ReplaceAll(modified, "Expect(obj)", "Expect(myObj)")

			By("customizing: adding custom setup code to BeforeEach")
			customCode := `		// Custom test setup
		myObj.Name = "my-test-object"
		myObj.Namespace = "test-ns"
		oldMyObj.Name = "old-object"
	})`
			modified = strings.Replace(modified, "	})\n\n	AfterEach(func() {", customCode+"\n\n	AfterEach(func() {", 1)

			err = os.WriteFile(testFile, []byte(modified), 0644)
			Expect(err).NotTo(HaveOccurred())

			By("customizing: adding custom implementation to webhook")
			webhookFile := filepath.Join(kbc.Dir, "internal/webhook/v1/testcustom_webhook.go")
			webhookContent, err := os.ReadFile(webhookFile)
			Expect(err).NotTo(HaveOccurred())

			modifiedWebhook := strings.ReplaceAll(string(webhookContent),
				"// TODO(user): fill in your defaulting logic.",
				`// My custom defaulting logic
	if testcustom.Spec.Replicas == nil {
		replicas := int32(1)
		testcustom.Spec.Replicas = &replicas
	}`)

			err = os.WriteFile(webhookFile, []byte(modifiedWebhook), 0644)
			Expect(err).NotTo(HaveOccurred())

			By("adding validation webhook WITHOUT --force")
			err = kbc.CreateWebhook(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestCustom",
				"--programmatic-validation",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying validator was added to test file")
			testContent, err = os.ReadFile(testFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(testContent)).To(ContainSubstring("validator TestCustomCustomValidator"))
			Expect(string(testContent)).To(ContainSubstring("Context(\"When creating or updating TestCustom under Validating Webhook\""))

			By("verifying user's renamed variables are preserved")
			Expect(string(testContent)).To(ContainSubstring("myObj     *testv1.TestCustom"))
			Expect(string(testContent)).To(ContainSubstring("oldMyObj  *testv1.TestCustom"))

			By("verifying user's custom BeforeEach code is preserved")
			Expect(string(testContent)).To(ContainSubstring("myObj.Name = \"my-test-object\""))
			Expect(string(testContent)).To(ContainSubstring("myObj.Namespace = \"test-ns\""))
			Expect(string(testContent)).To(ContainSubstring("oldMyObj.Name = \"old-object\""))

			By("verifying user's webhook implementation is preserved")
			webhookContent, err = os.ReadFile(webhookFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(webhookContent)).To(ContainSubstring("// My custom defaulting logic"))
			Expect(string(webhookContent)).To(ContainSubstring("testcustom.Spec.Replicas = &replicas"))

			By("verifying validator implementation was added")
			Expect(string(webhookContent)).To(ContainSubstring("type TestCustomCustomValidator struct"))
			Expect(string(webhookContent)).To(ContainSubstring("func (v *TestCustomCustomValidator) ValidateCreate"))
		})

		It("should work when user removes TODO comments", func() {
			By("creating an API")
			err := kbc.CreateAPI(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestNoTODO",
				"--resource", "--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating validation webhook")
			err = kbc.CreateWebhook(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestNoTODO",
				"--programmatic-validation",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("simulating user removing TODO comments")
			testFile := filepath.Join(kbc.Dir, "internal/webhook/v1/testnotodo_webhook_test.go")
			content, err := os.ReadFile(testFile)
			Expect(err).NotTo(HaveOccurred())

			modified := strings.ReplaceAll(string(content), "// TODO (user): Add any setup logic common to all tests\n", "")
			modified = strings.ReplaceAll(modified, "// TODO (user): Add any teardown logic common to all tests\n", "")

			err = os.WriteFile(testFile, []byte(modified), 0644)
			Expect(err).NotTo(HaveOccurred())

			By("adding defaulting webhook WITHOUT --force")
			err = kbc.CreateWebhook(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestNoTODO",
				"--defaulting",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying defaulter was added despite missing TODO comments")
			content, err = os.ReadFile(testFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring("defaulter TestNoTODOCustomDefaulter"))
			Expect(string(content)).To(ContainSubstring("Context(\"When creating TestNoTODO under Defaulting Webhook\""))
		})

		It("should support adding defaulting/validation to existing conversion webhook", func() {
			By("creating API v1")
			err := kbc.CreateAPI(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestMultiversion",
				"--resource", "--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating API v2")
			err = kbc.CreateAPI(
				"--group", "test",
				"--version", "v2",
				"--kind", "TestMultiversion",
				"--resource=false", "--controller=false",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating conversion webhook")
			err = kbc.CreateWebhook(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestMultiversion",
				"--conversion",
				"--spoke", "v2",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying only conversion test context exists initially")
			testFile := filepath.Join(kbc.Dir, "internal/webhook/v1/testmultiversion_webhook_test.go")
			content, err := os.ReadFile(testFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring("Context(\"When creating TestMultiversion under Conversion Webhook\""))
			Expect(string(content)).NotTo(ContainSubstring("defaulter"))
			Expect(string(content)).NotTo(ContainSubstring("validator"))

			By("adding defaulting and validation webhooks WITHOUT --force")
			err = kbc.CreateWebhook(
				"--group", "test",
				"--version", "v1",
				"--kind", "TestMultiversion",
				"--defaulting",
				"--programmatic-validation",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying all three webhook types are now present")
			content, err = os.ReadFile(testFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring("defaulter TestMultiversionCustomDefaulter"))
			Expect(string(content)).To(ContainSubstring("validator TestMultiversionCustomValidator"))
			Expect(string(content)).To(ContainSubstring("Context(\"When creating TestMultiversion under Conversion Webhook\""))
			Expect(string(content)).To(ContainSubstring("Context(\"When creating TestMultiversion under Defaulting Webhook\""))
			Expect(string(content)).To(ContainSubstring("Context(\"When creating or updating TestMultiversion under Validating Webhook\""))
		})

		It("should correctly scaffold conversion webhook and storage version marker", func() {
			By("creating API v1")
			err := kbc.CreateAPI(
				"--group", "batch",
				"--version", "v1",
				"--kind", "CronJob",
				"--resource", "--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating defaulting and validation webhooks for v1")
			err = kbc.CreateWebhook(
				"--group", "batch",
				"--version", "v1",
				"--kind", "CronJob",
				"--defaulting",
				"--programmatic-validation",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating API v2 without controller")
			err = kbc.CreateAPI(
				"--group", "batch",
				"--version", "v2",
				"--kind", "CronJob",
				"--resource=false", "--controller=false",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("adding conversion webhook to v1 WITHOUT --force")
			err = kbc.CreateWebhook(
				"--group", "batch",
				"--version", "v1",
				"--kind", "CronJob",
				"--conversion",
				"--spoke", "v2",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying v1 has all three webhook test contexts")
			testFile := filepath.Join(kbc.Dir, "internal/webhook/v1/cronjob_webhook_test.go")
			content, err := os.ReadFile(testFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring("Context(\"When creating CronJob under Defaulting Webhook\""))
			Expect(string(content)).To(ContainSubstring("Context(\"When creating or updating CronJob under Validating Webhook\""))
			Expect(string(content)).To(ContainSubstring("Context(\"When creating CronJob under Conversion Webhook\""))

			By("verifying hub and spoke files were created")
			hubFile := filepath.Join(kbc.Dir, "api/v1/cronjob_conversion.go")
			spokeFile := filepath.Join(kbc.Dir, "api/v2/cronjob_conversion.go")

			_, err = os.Stat(hubFile)
			Expect(err).NotTo(HaveOccurred(), "Hub file should exist")

			_, err = os.Stat(spokeFile)
			Expect(err).NotTo(HaveOccurred(), "Spoke file should exist")

			By("verifying storage version marker was added")
			typesFile := filepath.Join(kbc.Dir, "api/v1/cronjob_types.go")
			typesContent, err := os.ReadFile(typesFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(typesContent)).To(ContainSubstring("// +kubebuilder:storageversion"))
		})
	})
})
