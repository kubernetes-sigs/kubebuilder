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
	log "log/slog"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Webhook{}

// Webhook scaffolds the file that defines a webhook for a CRD or a builtin resource
type Webhook struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	// Is the Group domain for the Resource replacing '.' with '-'
	QualifiedGroupWithDash string

	// Define value for AdmissionReviewVersions marker
	AdmissionReviewVersions string

	Force bool

	// Deprecated - The flag should be removed from go/v5
	// IsLegacyPath indicates if webhooks should be scaffolded under the API.
	// Webhooks are now decoupled from APIs based on controller-runtime updates and community feedback.
	// This flag ensures backward compatibility by allowing scaffolding in the legacy/deprecated path.
	IsLegacyPath bool
}

// SetTemplateDefaults implements machinery.Template
func (f *Webhook) SetTemplateDefaults() error {
	if f.Path == "" {
		// Deprecated: Remove me when remove go/v4
		baseDir := "api"
		if !f.IsLegacyPath {
			baseDir = filepath.Join("internal", "webhook")
		}

		if f.MultiGroup && f.Resource.Group != "" {
			f.Path = filepath.Join(baseDir, "%[group]", "%[version]", "%[kind]_webhook.go")
		} else {
			f.Path = filepath.Join(baseDir, "%[version]", "%[kind]_webhook.go")
		}
	}

	f.Path = f.Resource.Replacer().Replace(f.Path)
	log.Info(f.Path)

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

	f.AdmissionReviewVersions = "v1"
	f.QualifiedGroupWithDash = strings.ReplaceAll(f.Resource.QualifiedGroup(), ".", "-")

	return nil
}

const (
	webhookTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	{{- if or .Resource.HasValidationWebhook .Resource.HasDefaultingWebhook }}
	"context"
	{{- end }}

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	{{- if .Resource.HasValidationWebhook }}
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	{{- end }}
	{{ if not .IsLegacyPath -}}
	{{ if not (isEmptyStr .Resource.Path) -}}
	{{ .Resource.ImportAlias }} "{{ .Resource.Path }}"
	{{- end }}
	{{- end }}
)

// nolint:unused
// log is for logging in this package.
var {{ lower .Resource.Kind }}log = logf.Log.WithName("{{ lower .Resource.Kind }}-resource")

{{- if .IsLegacyPath -}}
// SetupWebhookWithManager will setup the manager to manage the webhooks.
func (r *{{ .Resource.Kind }}) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		{{- if .Resource.HasValidationWebhook }}
		WithValidator(&{{ .Resource.Kind }}CustomValidator{}).
		{{- if ne .Resource.Webhooks.ValidationPath "" }}
		WithValidatorCustomPath("{{ .Resource.Webhooks.ValidationPath }}").
		{{- end }}
		{{- end }}
		{{- if .Resource.HasDefaultingWebhook }}
		WithDefaulter(&{{ .Resource.Kind }}CustomDefaulter{}).
		{{- if ne .Resource.Webhooks.DefaultingPath "" }}
		WithDefaulterCustomPath("{{ .Resource.Webhooks.DefaultingPath }}").
		{{- end }}
		{{- end }}
		Complete()
}
{{- else }}
// Setup{{ .Resource.Kind }}WebhookWithManager registers the webhook for {{ .Resource.Kind }} in the manager.
func Setup{{ .Resource.Kind }}WebhookWithManager(mgr ctrl.Manager) error {
	{{- if not (isEmptyStr .Resource.ImportAlias) }}
	return ctrl.NewWebhookManagedBy(mgr, &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}).
	{{- else }}
	return ctrl.NewWebhookManagedBy(mgr, &{{ .Resource.Kind }}{}).
	{{- end }}
		{{- if .Resource.HasValidationWebhook }}
		WithValidator(&{{ .Resource.Kind }}CustomValidator{}).
		{{- if ne .Resource.Webhooks.ValidationPath "" }}
		WithValidatorCustomPath("{{ .Resource.Webhooks.ValidationPath }}").
		{{- end }}
		{{- end }}
		{{- if .Resource.HasDefaultingWebhook }}
		WithDefaulter(&{{ .Resource.Kind }}CustomDefaulter{}).
		{{- if ne .Resource.Webhooks.DefaultingPath "" }}
		WithDefaulterCustomPath("{{ .Resource.Webhooks.DefaultingPath }}").
		{{- end }}
		{{- end }}
		Complete()
}
{{- end }}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
`

	//nolint:lll
	defaultingWebhookTemplate = `
// +kubebuilder:webhook:{{ if ne .Resource.Webhooks.WebhookVersion "v1" }}webhookVersions={{"{"}}{{ .Resource.Webhooks.WebhookVersion }}{{"}"}},{{ end }}{{- if ne .Resource.Webhooks.DefaultingPath "" -}}path={{ .Resource.Webhooks.DefaultingPath }}{{- else -}}path=/mutate-{{ if and .Resource.Core (eq .Resource.QualifiedGroup "core") }}-{{ else }}{{ .QualifiedGroupWithDash }}-{{ end }}{{ .Resource.Version }}-{{ lower .Resource.Kind }}{{- end -}},mutating=true,failurePolicy=fail,sideEffects=None,groups={{ if and .Resource.Core (eq .Resource.QualifiedGroup "core") }}""{{ else }}{{ .Resource.QualifiedGroup }}{{ end }},resources={{ .Resource.Plural }},verbs=create;update,versions={{ .Resource.Version }},name=m{{ lower .Resource.Kind }}-{{ .Resource.Version }}.kb.io,admissionReviewVersions={{ .AdmissionReviewVersions }}

{{ if .IsLegacyPath -}}
// +kubebuilder:object:generate=false
{{- end }}
// {{ .Resource.Kind }}CustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind {{ .Resource.Kind }} when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type {{ .Resource.Kind }}CustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind {{ .Resource.Kind }}.
{{- if .IsLegacyPath }}
func (d *{{ .Resource.Kind }}CustomDefaulter) Default(_ context.Context, obj *{{ .Resource.Kind }}) error {
	{{ lower .Resource.Kind }}log.Info("Defaulting for {{ .Resource.Kind }}", "name", obj.GetName())
{{- else }}
func (d *{{ .Resource.Kind }}CustomDefaulter) Default(_ context.Context, obj *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}) error {
	{{ lower .Resource.Kind }}log.Info("Defaulting for {{ .Resource.Kind }}", "name", obj.GetName())
{{- end }}

	// TODO(user): fill in your defaulting logic.

	return nil
}
`

	//nolint:lll
	validatingWebhookTemplate = `
// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:{{ if ne .Resource.Webhooks.WebhookVersion "v1" }}webhookVersions={{"{"}}{{ .Resource.Webhooks.WebhookVersion }}{{"}"}},{{ end }}{{- if ne .Resource.Webhooks.ValidationPath "" -}}path={{ .Resource.Webhooks.ValidationPath }}{{- else -}}path=/validate-{{ if and .Resource.Core (eq .Resource.QualifiedGroup "core") }}-{{ else }}{{ .QualifiedGroupWithDash }}-{{ end }}{{ .Resource.Version }}-{{ lower .Resource.Kind }}{{- end -}},mutating=false,failurePolicy=fail,sideEffects=None,groups={{ if and .Resource.Core (eq .Resource.QualifiedGroup "core") }}""{{ else }}{{ .Resource.QualifiedGroup }}{{ end }},resources={{ .Resource.Plural }},verbs=create;update,versions={{ .Resource.Version }},name=v{{ lower .Resource.Kind }}-{{ .Resource.Version }}.kb.io,admissionReviewVersions={{ .AdmissionReviewVersions }}

{{ if .IsLegacyPath -}}
// +kubebuilder:object:generate=false
{{- end }}
// {{ .Resource.Kind }}CustomValidator struct is responsible for validating the {{ .Resource.Kind }} resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type {{ .Resource.Kind }}CustomValidator struct{
	// TODO(user): Add more fields as needed for validation
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type {{ .Resource.Kind }}.
{{- if .IsLegacyPath }}
func (v *{{ .Resource.Kind }}CustomValidator) ValidateCreate(_ context.Context, obj *{{ .Resource.Kind }}) (admission.Warnings, error) {
	{{ lower .Resource.Kind }}log.Info("Validation for {{ .Resource.Kind }} upon creation", "name", obj.GetName())
{{- else }}
func (v *{{ .Resource.Kind }}CustomValidator) ValidateCreate(_ context.Context, obj *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}) (admission.Warnings, error) {
	{{ lower .Resource.Kind }}log.Info("Validation for {{ .Resource.Kind }} upon creation", "name", obj.GetName())
{{- end }}

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type {{ .Resource.Kind }}.
{{- if .IsLegacyPath }}
func (v *{{ .Resource.Kind }}CustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *{{ .Resource.Kind }}) (admission.Warnings, error) {
	{{ lower .Resource.Kind }}log.Info("Validation for {{ .Resource.Kind }} upon update", "name", newObj.GetName())
{{- else }}
func (v *{{ .Resource.Kind }}CustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}) (admission.Warnings, error) {
	{{ lower .Resource.Kind }}log.Info("Validation for {{ .Resource.Kind }} upon update", "name", newObj.GetName())
{{- end }}

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type {{ .Resource.Kind }}.
{{- if .IsLegacyPath }}
func (v *{{ .Resource.Kind }}CustomValidator) ValidateDelete(_ context.Context, obj *{{ .Resource.Kind }}) (admission.Warnings, error) {
	{{ lower .Resource.Kind }}log.Info("Validation for {{ .Resource.Kind }} upon deletion", "name", obj.GetName())
{{- else }}
func (v *{{ .Resource.Kind }}CustomValidator) ValidateDelete(_ context.Context, obj *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}) (admission.Warnings, error) {
	{{ lower .Resource.Kind }}log.Info("Validation for {{ .Resource.Kind }} upon deletion", "name", obj.GetName())
{{- end }}

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
`
)
