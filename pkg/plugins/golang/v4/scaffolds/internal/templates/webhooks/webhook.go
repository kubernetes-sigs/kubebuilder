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
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

var _ machinery.Template = &Webhook{}

// Webhook scaffolds the file that defines a webhook for a CRD or a builtin resource,
// or a standalone multi-GVK webhook when MultiGVK is true.
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

	// MultiGVK indicates this is a standalone multi-GVK webhook (not tied to a single CRD Kind).
	MultiGVK bool

	// MultiGVKWebhook holds the webhook configuration for multi-GVK webhooks.
	MultiGVKWebhook resource.Webhook
}

// HandlerName returns the Go struct name for the multi-GVK webhook handler.
func (f *Webhook) HandlerName() string {
	parts := strings.Split(f.MultiGVKWebhook.Name, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "") + "Webhook"
}

// MarkerGroups returns the semicolon-separated group list for the +kubebuilder:webhook marker.
func (f *Webhook) MarkerGroups() string {
	var parts []string
	for _, g := range f.MultiGVKWebhook.Groups {
		if g == "" || g == "core" {
			parts = append(parts, `""`)
		} else {
			parts = append(parts, g)
		}
	}
	return strings.Join(parts, ";")
}

// MarkerKinds returns the semicolon-separated kinds list for the +kubebuilder:webhook marker.
func (f *Webhook) MarkerKinds() string {
	return strings.Join(f.MultiGVKWebhook.Kinds, ";")
}

// MarkerVersions returns the semicolon-separated versions list for the +kubebuilder:webhook marker.
func (f *Webhook) MarkerVersions() string {
	return strings.Join(f.MultiGVKWebhook.Versions, ";")
}

// MultiGVKDefaultingPath returns the defaulting webhook path for multi-GVK webhooks.
func (f *Webhook) MultiGVKDefaultingPath() string {
	if f.MultiGVKWebhook.DefaultingPath != "" {
		return f.MultiGVKWebhook.DefaultingPath
	}
	return "/mutate-" + strings.ToLower(f.MultiGVKWebhook.Name)
}

// MultiGVKValidationPath returns the validation webhook path for multi-GVK webhooks.
func (f *Webhook) MultiGVKValidationPath() string {
	if f.MultiGVKWebhook.ValidationPath != "" {
		return f.MultiGVKWebhook.ValidationPath
	}
	return "/validate-" + strings.ToLower(f.MultiGVKWebhook.Name)
}

// SetTemplateDefaults implements machinery.Template
func (f *Webhook) SetTemplateDefaults() error {
	if f.MultiGVK {
		if f.Path == "" {
			f.Path = filepath.Join("internal", "webhook", "%[webhook-name]_webhook.go")
		}
		f.Path = strings.ReplaceAll(f.Path, "%[webhook-name]", strings.ToLower(f.MultiGVKWebhook.Name))
		f.TemplateBody = multiGVKWebhookTemplate
	} else {
		if f.Path == "" {
			baseDir := filepath.Join("internal", "webhook")

			if f.MultiGroup && f.Resource.Group != "" {
				f.Path = filepath.Join(baseDir, "%[group]", "%[version]", "%[kind]_webhook.go")
			} else {
				f.Path = filepath.Join(baseDir, "%[version]", "%[kind]_webhook.go")
			}
		}

		f.Path = f.Resource.Replacer().Replace(f.Path)
		log.Info(f.Path)

		webhookCore := webhookTemplate
		if f.Resource.HasDefaultingWebhook() {
			webhookCore = webhookCore + defaultingWebhookTemplate
		}
		if f.Resource.HasValidationWebhook() {
			webhookCore = webhookCore + validatingWebhookTemplate
		}
		f.TemplateBody = webhookCore

		f.AdmissionReviewVersions = "v1"
		f.QualifiedGroupWithDash = strings.ReplaceAll(f.Resource.QualifiedGroup(), ".", "-")
	}

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.Error
	}

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
	{{ if not (isEmptyStr .Resource.Path) -}}
	{{ .Resource.ImportAlias }} "{{ .Resource.Path }}"
	{{- end }}
)

// nolint:unused
// log is for logging in this package.
var {{ lower .Resource.Kind }}log = logf.Log.WithName("{{ lower .Resource.Kind }}-resource")


// Setup{{ .Resource.Kind }}WebhookWithManager registers the webhook for {{ .Resource.Kind }} in the manager.
func Setup{{ .Resource.Kind }}WebhookWithManager(mgr ctrl.Manager) error {
	{{- if not (isEmptyStr .Resource.ImportAlias) }}
	return ctrl.NewWebhookManagedBy(mgr, &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}).
	{{- else }}
	return ctrl.NewWebhookManagedBy(mgr, &{{ .Resource.Kind }}{}).
	{{- end }}
		{{- if .Resource.HasValidationWebhook }}
		WithValidator(&{{ .Resource.Kind }}CustomValidator{}).
		{{- if ne .Resource.Webhook.ValidationPath "" }}
		WithValidatorCustomPath("{{ .Resource.Webhook.ValidationPath }}").
		{{- end }}
		{{- end }}
		{{- if .Resource.HasDefaultingWebhook }}
		WithDefaulter(&{{ .Resource.Kind }}CustomDefaulter{}).
		{{- if ne .Resource.Webhook.DefaultingPath "" }}
		WithDefaulterCustomPath("{{ .Resource.Webhook.DefaultingPath }}").
		{{- end }}
		{{- end }}
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
`

	//nolint:lll
	defaultingWebhookTemplate = `
// +kubebuilder:webhook:{{ if ne .Resource.Webhook.WebhookVersion "v1" }}webhookVersions={{"{"}}{{ .Resource.Webhook.WebhookVersion }}{{"}"}},{{ end }}{{- if ne .Resource.Webhook.DefaultingPath "" -}}path={{ .Resource.Webhook.DefaultingPath }}{{- else -}}path=/mutate-{{ if and .Resource.Core (eq .Resource.QualifiedGroup "core") }}-{{ else }}{{ .QualifiedGroupWithDash }}-{{ end }}{{ .Resource.Version }}-{{ lower .Resource.Kind }}{{- end -}},mutating=true,failurePolicy=fail,sideEffects=None,groups={{ if and .Resource.Core (eq .Resource.QualifiedGroup "core") }}""{{ else }}{{ .Resource.QualifiedGroup }}{{ end }},resources={{ .Resource.Plural }},verbs=create;update,versions={{ .Resource.Version }},name=m{{ lower .Resource.Kind }}-{{ .Resource.Version }}.kb.io,admissionReviewVersions={{ .AdmissionReviewVersions }}

// {{ .Resource.Kind }}CustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind {{ .Resource.Kind }} when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type {{ .Resource.Kind }}CustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind {{ .Resource.Kind }}.
func (d *{{ .Resource.Kind }}CustomDefaulter) Default(_ context.Context, obj *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}) error {
	{{ lower .Resource.Kind }}log.Info("Defaulting for {{ .Resource.Kind }}", "name", obj.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}
`

	//nolint:lll
	validatingWebhookTemplate = `
// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:{{ if ne .Resource.Webhook.WebhookVersion "v1" }}webhookVersions={{"{"}}{{ .Resource.Webhook.WebhookVersion }}{{"}"}},{{ end }}{{- if ne .Resource.Webhook.ValidationPath "" -}}path={{ .Resource.Webhook.ValidationPath }}{{- else -}}path=/validate-{{ if and .Resource.Core (eq .Resource.QualifiedGroup "core") }}-{{ else }}{{ .QualifiedGroupWithDash }}-{{ end }}{{ .Resource.Version }}-{{ lower .Resource.Kind }}{{- end -}},mutating=false,failurePolicy=fail,sideEffects=None,groups={{ if and .Resource.Core (eq .Resource.QualifiedGroup "core") }}""{{ else }}{{ .Resource.QualifiedGroup }}{{ end }},resources={{ .Resource.Plural }},verbs=create;update,versions={{ .Resource.Version }},name=v{{ lower .Resource.Kind }}-{{ .Resource.Version }}.kb.io,admissionReviewVersions={{ .AdmissionReviewVersions }}

// {{ .Resource.Kind }}CustomValidator struct is responsible for validating the {{ .Resource.Kind }} resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type {{ .Resource.Kind }}CustomValidator struct{
	// TODO(user): Add more fields as needed for validation
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type {{ .Resource.Kind }}.
func (v *{{ .Resource.Kind }}CustomValidator) ValidateCreate(_ context.Context, obj *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}) (admission.Warnings, error) {
	{{ lower .Resource.Kind }}log.Info("Validation for {{ .Resource.Kind }} upon creation", "name", obj.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type {{ .Resource.Kind }}.
func (v *{{ .Resource.Kind }}CustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}) (admission.Warnings, error) {
	{{ lower .Resource.Kind }}log.Info("Validation for {{ .Resource.Kind }} upon update", "name", newObj.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type {{ .Resource.Kind }}.
func (v *{{ .Resource.Kind }}CustomValidator) ValidateDelete(_ context.Context, obj *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}) (admission.Warnings, error) {
	{{ lower .Resource.Kind }}log.Info("Validation for {{ .Resource.Kind }} upon deletion", "name", obj.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
`

	//nolint:lll
	multiGVKWebhookTemplate = `{{ .Boilerplate }}

package webhook

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

{{ if .MultiGVKWebhook.Defaulting }}
// +kubebuilder:webhook:path={{ .MultiGVKDefaultingPath }},mutating=true,failurePolicy=fail,sideEffects=None,groups={{ .MarkerGroups }},resources={{ .MarkerKinds }},verbs=create;update,versions={{ .MarkerVersions }},name=m{{ lower .MultiGVKWebhook.Name }}.kb.io,admissionReviewVersions=v1
{{ end }}
{{ if .MultiGVKWebhook.Validation }}
// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path={{ .MultiGVKValidationPath }},mutating=false,failurePolicy=fail,sideEffects=None,groups={{ .MarkerGroups }},resources={{ .MarkerKinds }},verbs=create;update,versions={{ .MarkerVersions }},name=v{{ lower .MultiGVKWebhook.Name }}.kb.io,admissionReviewVersions=v1
{{ end }}

{{ if and .MultiGVKWebhook.Defaulting .MultiGVKWebhook.Validation }}
// {{ .HandlerName }} mutates and validates intercepted resources.
{{ else if .MultiGVKWebhook.Defaulting }}
// {{ .HandlerName }} mutates intercepted resources.
{{ else }}
// {{ .HandlerName }} validates intercepted resources.
{{ end }}
type {{ .HandlerName }} struct {
}

// {{ .HandlerName }} implements admission.Handler.
var _ admission.Handler = &{{ .HandlerName }}{}

// Handle implements admission.Handler.
func (h *{{ .HandlerName }}) Handle(ctx context.Context, req admission.Request) admission.Response {
	{{ if and .MultiGVKWebhook.Defaulting .MultiGVKWebhook.Validation }}
	// TODO(user): fill in your webhook logic.
	{{ else if .MultiGVKWebhook.Defaulting }}
	// TODO(user): fill in your defaulting logic.
	{{ else }}
	// TODO(user): fill in your validation logic.
	{{ end }}
	// Use admission.Decoder to decode req.Object.Raw.
	// Use req.Resource.Group, req.Resource.Version, and req.Resource.Resource
	// to identify which resource type triggered this webhook.
	return admission.Allowed("")
}
`
)
