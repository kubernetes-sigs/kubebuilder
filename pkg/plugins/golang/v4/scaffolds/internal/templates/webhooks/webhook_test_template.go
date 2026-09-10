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

package webhooks

import (
	"fmt"
	log "log/slog"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &WebhookTest{}

// WebhookTest scaffolds the file that sets up the webhook unit tests
type WebhookTest struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin
	machinery.IfNotExistsActionMixin

	Force bool
}

// SetTemplateDefaults implements machinery.Template
func (f *WebhookTest) SetTemplateDefaults() error {
	if f.Path == "" {
		pathAPI := filepath.Join("internal", "webhook")

		if f.MultiGroup && f.Resource.Group != "" {
			f.Path = filepath.Join(pathAPI, "%[group]", "%[version]", "%[kind]_webhook_test.go")
		} else {
			f.Path = filepath.Join(pathAPI, "%[version]", "%[kind]_webhook_test.go")
		}
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)
	log.Info(f.Path)

	webhookTestTemplate := webhookTestTemplate
	templates := make([]string, 0)
	if f.Resource.HasDefaultingWebhook() {
		templates = append(templates, defaultWebhookTestTemplate)
	}
	if f.Resource.HasValidationWebhook() {
		templates = append(templates, validateWebhookTestTemplate)
	}
	if f.Resource.HasConversionWebhook() {
		templates = append(templates, conversionWebhookTestTemplate)
	}
	f.TemplateBody = fmt.Sprintf(webhookTestTemplate, strings.Join(templates, "\n"))

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	}
	f.IfNotExistsAction = machinery.IgnoreFile

	return nil
}

const webhookTestTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	{{ if not (isEmptyStr .Resource.Path) -}}
	{{ .Resource.ImportAlias }} "{{ .Resource.Path }}"
	{{- end }}
	// TODO (user): Add any additional imports if needed
)

var _ = Describe("{{ .Resource.Kind }} Webhook", func() {
	var (
		obj *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}
		oldObj *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}
		{{- if .Resource.HasValidationWebhook }}
		validator {{ .Resource.Kind }}Validator
		{{- end }}
		{{- if .Resource.HasDefaultingWebhook }}
		defaulter {{ .Resource.Kind }}Defaulter
		{{- end }}
	)

	BeforeEach(func() {
		obj = &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}
		oldObj = &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}
		{{- if .Resource.HasValidationWebhook }}
		validator = {{ .Resource.Kind }}Validator{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		{{- end }}
		{{- if .Resource.HasDefaultingWebhook }}
		defaulter = {{ .Resource.Kind }}Defaulter{}
		Expect(defaulter).NotTo(BeNil(), "Expected defaulter to be initialized")
		{{- end }}
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	%s
})
`

const conversionWebhookTestTemplate = `
Context("When creating {{ .Resource.Kind }} under Conversion Webhook", func() {
	// TODO (user): Add logic to convert the object to the desired version and verify the conversion
	// Example:
	// It("Should convert the object correctly", func() {
	//     convertedObj := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}
	//     Expect(obj.ConvertTo(convertedObj)).To(Succeed())
	//     Expect(convertedObj).ToNot(BeNil())
	// })
})
`

const validateWebhookTestTemplate = `
Context("When creating or updating {{ .Resource.Kind }} under Validating Webhook", func() {
	// TODO (user): Add logic for validating webhooks
	// Example:
	// It("Should deny creation if a required field is missing", func() {
	//     By("simulating an invalid creation scenario")
	//     obj.SomeRequiredField = ""
	//     Expect(validator.ValidateCreate(ctx, obj)).Error().To(HaveOccurred())
	// })
	//
	// It("Should admit creation if all required fields are present", func() {
	//     By("simulating an invalid creation scenario")
	//     obj.SomeRequiredField = "valid_value"
	//     Expect(validator.ValidateCreate(ctx, obj)).To(BeNil())
	// })
	//
	// It("Should validate updates correctly", func() {
	//     By("simulating a valid update scenario")
	//     oldObj.SomeRequiredField = "updated_value"
	//     obj.SomeRequiredField = "updated_value"
	//     Expect(validator.ValidateUpdate(ctx, oldObj, obj)).To(BeNil())
	// })
})
`

const defaultWebhookTestTemplate = `
Context("When creating {{ .Resource.Kind }} under Defaulting Webhook", func() {
	// TODO (user): Add logic for defaulting webhooks
	// Example:
	// It("Should apply defaults when a required field is empty", func() {
	//     By("simulating a scenario where defaults should be applied")
	//     obj.SomeFieldWithDefault = ""
	//     By("calling the Default method to apply defaults")
	//     defaulter.Default(ctx, obj)
	//     By("checking that the default values are set")
	//     Expect(obj.SomeFieldWithDefault).To(Equal("default_value"))
	// })
})
`
