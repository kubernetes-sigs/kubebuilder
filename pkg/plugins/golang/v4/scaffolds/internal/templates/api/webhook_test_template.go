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

package api

import (
	"fmt"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &WebhookTest{}

// WebhookTest scaffolds the file that sets up the webhook unit tests
type WebhookTest struct { // nolint:maligned
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	Force bool
}

// SetTemplateDefaults implements file.Template
func (f *WebhookTest) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup && f.Resource.Group != "" {
			f.Path = filepath.Join("api", "%[group]", "%[version]", "%[kind]_webhook_test.go")
		} else {
			f.Path = filepath.Join("api", "%[version]", "%[kind]_webhook_test.go")
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
)

var _ = Describe("{{ .Resource.Kind }} Webhook", func() {
	%s
})
`

const conversionWebhookTestTemplate = `
Context("When creating {{ .Resource.Kind }} under Conversion Webhook", func() {
	It("Should get the converted version of {{ .Resource.Kind }}" , func() {

		// TODO(user): Add your logic here

	})
})
`

const validateWebhookTestTemplate = `
Context("When creating {{ .Resource.Kind }} under Validating Webhook", func() {
	It("Should deny if a required field is empty", func() {

		// TODO(user): Add your logic here

	})

	It("Should admit if all required fields are provided", func() {

		// TODO(user): Add your logic here

	})
})
`

const defaultWebhookTestTemplate = `
Context("When creating {{ .Resource.Kind }} under Defaulting Webhook", func() {
	It("Should fill in the default value if a required field is empty", func() {

		// TODO(user): Add your logic here

	})
})
`
