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
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &WebhookTest{}

// WebhookTest scaffolds the file that sets up the webhook unit tests
type WebhookTest struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	Force bool

	// Deprecated - The flag should be removed from go/v5
	// IsLegacyPath indicates if webhooks should be scaffolded under the API.
	// Webhooks are now decoupled from APIs based on controller-runtime updates and community feedback.
	// This flag ensures backward compatibility by allowing scaffolding in the legacy/deprecated path.
	IsLegacyPath bool
}

// SetTemplateDefaults implements machinery.Template
func (f *WebhookTest) SetTemplateDefaults() error {
	if f.Path == "" {
		// Deprecated: Remove me when remove go/v4
		//nolint:goconst
		baseDir := "api"
		if !f.IsLegacyPath {
			baseDir = filepath.Join("internal", "webhook")
		}

		if f.MultiGroup && f.Resource.Group != "" {
			f.Path = filepath.Join(baseDir, "%[group]", "%[version]", "%[kind]_webhook_test.go")
		} else {
			f.Path = filepath.Join(baseDir, "%[version]", "%[kind]_webhook_test.go")
		}
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)
	log.Println(f.Path)

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

	return nil
}

const webhookTestTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	{{ if not .IsLegacyPath -}}
	{{ if not (isEmptyStr .Resource.Path) -}}
	{{ .Resource.ImportAlias }} "{{ .Resource.Path }}"
	{{- end }}
	{{- end }}
	// TODO (user): Add any additional imports if needed
)

var _ = Describe("{{ .Resource.Kind }} Webhook", func() {
	var (
		{{- if .IsLegacyPath -}}
		obj *{{ .Resource.Kind }}
		{{- else }}
		obj *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}
		oldObj *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}
		{{- if .Resource.HasValidationWebhook }}
		validator {{ .Resource.Kind }}CustomValidator
		{{- end }}
		{{- if .Resource.HasDefaultingWebhook }}
		defaulter {{ .Resource.Kind }}CustomDefaulter
		{{- end }}
		{{- end }}
	)

	BeforeEach(func() {
		{{- if .IsLegacyPath -}}
		obj = &{{ .Resource.Kind }}{}
		{{- else }}
		obj = &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}
		oldObj = &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}
		{{- if .Resource.HasValidationWebhook }}
		validator = {{ .Resource.Kind }}CustomValidator{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		{{- end }}
		{{- if .Resource.HasDefaultingWebhook }}
		defaulter = {{ .Resource.Kind }}CustomDefaulter{}
		Expect(defaulter).NotTo(BeNil(), "Expected defaulter to be initialized")
		{{- end }}
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		{{- end }}
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
		// TODO (user): Add any setup logic common to all tests
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
	{{- if .IsLegacyPath -}}
	//     convertedObj := &{{ .Resource.Kind }}{}
	{{- else }}
	//     convertedObj := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}
	{{- end }}
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
	{{- if .IsLegacyPath -}}
	//     Expect(obj.ValidateCreate(ctx)).Error().To(HaveOccurred())
	{{- else }}
	//     Expect(validator.ValidateCreate(ctx, obj)).Error().To(HaveOccurred())
	{{- end }}
	// })
	//
	// It("Should admit creation if all required fields are present", func() {
	//     By("simulating an invalid creation scenario")
	//     obj.SomeRequiredField = "valid_value"
	{{- if .IsLegacyPath -}}
	//     Expect(obj.ValidateCreate(ctx)).To(BeNil())
	{{- else }}
	//     Expect(validator.ValidateCreate(ctx, obj)).To(BeNil())
	{{- end }}
	// })
	//
	// It("Should validate updates correctly", func() {
	//     By("simulating a valid update scenario")
	{{- if .IsLegacyPath -}}
	//     oldObj := &Captain{SomeRequiredField: "valid_value"}
	//     obj.SomeRequiredField = "updated_value"
	//     Expect(obj.ValidateUpdate(ctx, oldObj)).To(BeNil())
	{{- else }}
	//     oldObj.SomeRequiredField = "updated_value"
	//     obj.SomeRequiredField = "updated_value"
	//     Expect(validator.ValidateUpdate(ctx, oldObj, obj)).To(BeNil())
	{{- end }}
	// })
})
`

const defaultWebhookTestTemplate = `
Context("When creating {{ .Resource.Kind }} under Defaulting Webhook", func() {
	// TODO (user): Add logic for defaulting webhooks
	// Example:
	// It("Should apply defaults when a required field is empty", func() {
	//     By("simulating a scenario where defaults should be applied")
	{{- if .IsLegacyPath -}}
	//     obj.SomeFieldWithDefault = ""
	//     Expect(obj.Default(ctx)).To(Succeed())
	//     Expect(obj.SomeFieldWithDefault).To(Equal("default_value"))
	{{- else }}
	//     obj.SomeFieldWithDefault = ""
	//     By("calling the Default method to apply defaults")
	//     defaulter.Default(ctx, obj)
	//     By("checking that the default values are set")
	//     Expect(obj.SomeFieldWithDefault).To(Equal("default_value"))
	{{- end }}
	// })
})
`
