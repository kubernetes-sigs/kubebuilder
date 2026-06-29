package webhooks

import (
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

var _ machinery.Template = &MultiGVKWebhook{}

// MultiGVKWebhook scaffolds a multi-GVK webhook that intercepts multiple resource types
type MultiGVKWebhook struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin

	Force bool

	// Webhook holds the multi-GVK webhook configuration
	Webhook resource.Webhook
}

// SetTemplateDefaults implements machinery.Template
func (f *MultiGVKWebhook) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("internal", "webhook", "%[webhook-name]_webhook.go")
	}

	f.Path = strings.ReplaceAll(f.Path, "%[webhook-name]", strings.ToLower(f.Webhook.Name))

	var templateBody string
	if f.Webhook.Defaulting && f.Webhook.Validation {
		templateBody = multiGVKWebhookDefValTemplate
	} else if f.Webhook.Defaulting {
		templateBody = multiGVKWebhookDefaultingTemplate
	} else {
		templateBody = multiGVKWebhookValidationTemplate
	}

	f.TemplateBody = templateBody

	// Replace path markers in the template body
	defaultingPath := f.Webhook.DefaultingPath
	if defaultingPath == "" {
		defaultingPath = "/mutate-" + strings.ToLower(f.Webhook.Name)
	}
	validationPath := f.Webhook.ValidationPath
	if validationPath == "" {
		validationPath = "/validate-" + strings.ToLower(f.Webhook.Name)
	}
	f.TemplateBody = strings.ReplaceAll(f.TemplateBody, "%[def-path]", defaultingPath)
	f.TemplateBody = strings.ReplaceAll(f.TemplateBody, "%[val-path]", validationPath)

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.Error
	}

	return nil
}

// HandlerName returns the Go struct name for the webhook handler.
func (f *MultiGVKWebhook) HandlerName() string {
	parts := strings.Split(f.Webhook.Name, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "") + "Webhook"
}

// MarkerGroups returns the semicolon-separated group list for the +kubebuilder:webhook marker.
func (f *MultiGVKWebhook) MarkerGroups() string {
	var parts []string
	for _, g := range f.Webhook.Groups {
		if g == "" {
			parts = append(parts, `""`)
		} else {
			parts = append(parts, g)
		}
	}
	return strings.Join(parts, ";")
}

// MarkerKinds returns the semicolon-separated kinds list for the +kubebuilder:webhook marker.
func (f *MultiGVKWebhook) MarkerKinds() string {
	return strings.Join(f.Webhook.Kinds, ";")
}

// MarkerVersions returns the semicolon-separated versions list for the +kubebuilder:webhook marker.
func (f *MultiGVKWebhook) MarkerVersions() string {
	return strings.Join(f.Webhook.Versions, ";")
}

//nolint:lll
const multiGVKWebhookDefaultingTemplate = `{{ .Boilerplate }}

package webhook

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=%[def-path],mutating=true,failurePolicy=fail,sideEffects=None,groups={{ .MarkerGroups }},resources={{ .MarkerKinds }},verbs=create;update,versions={{ .MarkerVersions }},name=m{{ lower .Webhook.Name }}.kb.io,admissionReviewVersions=v1

// {{ .HandlerName }} mutates intercepted resources.
type {{ .HandlerName }} struct {
	Decoder admission.Decoder
}

// {{ .HandlerName }} implements admission.Handler.
var _ admission.Handler = &{{ .HandlerName }}{}

// Handle implements admission.Handler.
func (h *{{ .HandlerName }}) Handle(ctx context.Context, req admission.Request) admission.Response {
	// TODO(user): fill in your defaulting logic.
	// Use req.Object.Raw to access the raw object, then decode it with h.Decoder.
	// Use req.Resource.Group, req.Resource.Version, and req.Resource.Resource
	// to identify which resource type triggered this webhook.
	return admission.Allowed("")
}
`

//nolint:lll
const multiGVKWebhookValidationTemplate = `{{ .Boilerplate }}

package webhook

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=%[val-path],mutating=false,failurePolicy=fail,sideEffects=None,groups={{ .MarkerGroups }},resources={{ .MarkerKinds }},verbs=create;update,versions={{ .MarkerVersions }},name=v{{ lower .Webhook.Name }}.kb.io,admissionReviewVersions=v1

// {{ .HandlerName }} validates intercepted resources.
type {{ .HandlerName }} struct {
	Decoder admission.Decoder
}

// {{ .HandlerName }} implements admission.Handler.
var _ admission.Handler = &{{ .HandlerName }}{}

// Handle implements admission.Handler.
func (h *{{ .HandlerName }}) Handle(ctx context.Context, req admission.Request) admission.Response {
	// TODO(user): fill in your validation logic.
	// Use req.Object.Raw to access the raw object, then decode it with h.Decoder.
	// Use req.Resource.Group, req.Resource.Version, and req.Resource.Resource
	// to identify which resource type triggered this webhook.
	return admission.Allowed("")
}
`

//nolint:lll
const multiGVKWebhookDefValTemplate = `{{ .Boilerplate }}

package webhook

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=%[def-path],mutating=true,failurePolicy=fail,sideEffects=None,groups={{ .MarkerGroups }},resources={{ .MarkerKinds }},verbs=create;update,versions={{ .MarkerVersions }},name=m{{ lower .Webhook.Name }}.kb.io,admissionReviewVersions=v1

// +kubebuilder:webhook:path=%[val-path],mutating=false,failurePolicy=fail,sideEffects=None,groups={{ .MarkerGroups }},resources={{ .MarkerKinds }},verbs=create;update,versions={{ .MarkerVersions }},name=v{{ lower .Webhook.Name }}.kb.io,admissionReviewVersions=v1

// {{ .HandlerName }} mutates and validates intercepted resources.
type {{ .HandlerName }} struct {
	Decoder admission.Decoder
}

// {{ .HandlerName }} implements admission.Handler.
var _ admission.Handler = &{{ .HandlerName }}{}

// Handle implements admission.Handler.
func (h *{{ .HandlerName }}) Handle(ctx context.Context, req admission.Request) admission.Response {
	// TODO(user): fill in your webhook logic.
	// Use req.Object.Raw to access the raw object, then decode it with h.Decoder.
	// Use req.Resource.Group, req.Resource.Version, and req.Resource.Resource
	// to identify which resource type triggered this webhook.
	return admission.Allowed("")
}
`
