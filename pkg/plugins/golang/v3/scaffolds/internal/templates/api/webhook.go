/*
Copyright 2020 The Kubernetes Authors.

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

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &Webhook{}

// Webhook scaffolds the file that defines a webhook for a CRD or a builtin resource
type Webhook struct { // nolint:maligned
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	// Is the Group domain for the Resource replacing '.' with '-'
	QualifiedGroupWithDash string

	Force bool
}

// SetTemplateDefaults implements file.Template
func (f *Webhook) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup {
			if f.Resource.Group != "" {
				f.Path = filepath.Join("apis", "%[group]", "%[version]", "%[kind]_webhook.go")
			} else {
				f.Path = filepath.Join("apis", "%[version]", "%[kind]_webhook.go")
			}
		} else {
			f.Path = filepath.Join("api", "%[version]", "%[kind]_webhook.go")
		}
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)
	fmt.Println(f.Path)

	webhookTemplate := webhookTemplate
	if f.Resource.HasDefaultingWebhook() {
		webhookTemplate = webhookTemplate + defaultingWebhookTemplate
	}
	if f.Resource.HasValidationWebhook() {
		webhookTemplate = webhookTemplate + validatingWebhookTemplate
	}
	f.TemplateBody = webhookTemplate

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.Error
	}

	f.QualifiedGroupWithDash = strings.Replace(f.Resource.QualifiedGroup(), ".", "-", -1)

	return nil
}

const (
	webhookTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	{{- if .Resource.HasValidationWebhook }}
	"k8s.io/apimachinery/pkg/runtime"
	{{- end }}
	{{- if or .Resource.HasValidationWebhook .Resource.HasDefaultingWebhook }}
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	{{- end }}
)

// log is for logging in this package.
var {{ lower .Resource.Kind }}log = logf.Log.WithName("{{ lower .Resource.Kind }}-resource")

func (r *{{ .Resource.Kind }}) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
`

	// TODO(estroz): update admissionReviewVersions to include v1 when controller-runtime supports that version.
	//nolint:lll
	defaultingWebhookTemplate = `
//+kubebuilder:webhook:{{ if ne .Resource.Webhooks.WebhookVersion "v1" }}webhookVersions={{"{"}}{{ .Resource.Webhooks.WebhookVersion }}{{"}"}},{{ end }}path=/mutate-{{ .QualifiedGroupWithDash }}-{{ .Resource.Version }}-{{ lower .Resource.Kind }},mutating=true,failurePolicy=fail,sideEffects=None,groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }},verbs=create;update,versions={{ .Resource.Version }},name=m{{ lower .Resource.Kind }}.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &{{ .Resource.Kind }}{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *{{ .Resource.Kind }}) Default() {
	{{ lower .Resource.Kind }}log.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}
`

	// TODO(estroz): update admissionReviewVersions to include v1 when controller-runtime supports that version.
	//nolint:lll
	validatingWebhookTemplate = `
// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:{{ if ne .Resource.Webhooks.WebhookVersion "v1" }}webhookVersions={{"{"}}{{ .Resource.Webhooks.WebhookVersion }}{{"}"}},{{ end }}path=/validate-{{ .QualifiedGroupWithDash }}-{{ .Resource.Version }}-{{ lower .Resource.Kind }},mutating=false,failurePolicy=fail,sideEffects=None,groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }},verbs=create;update,versions={{ .Resource.Version }},name=v{{ lower .Resource.Kind }}.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &{{ .Resource.Kind }}{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *{{ .Resource.Kind }}) ValidateCreate() error {
	{{ lower .Resource.Kind }}log.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *{{ .Resource.Kind }}) ValidateUpdate(old runtime.Object) error {
	{{ lower .Resource.Kind }}log.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *{{ .Resource.Kind }}) ValidateDelete() error {
	{{ lower .Resource.Kind }}log.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
`
)
