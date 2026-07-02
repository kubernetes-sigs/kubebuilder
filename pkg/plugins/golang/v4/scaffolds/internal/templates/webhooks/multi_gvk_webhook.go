package webhooks

import (
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

var _ machinery.Template = &MultiGVKWebhook{}

// MultiGVKWebhook scaffolds a webhook that intercepts multiple resource types (multi-GVK).
// Unlike Webhook (which is tied to a single CRD Kind), this template generates an
// admission.Handler that switches on req.Resource at runtime. A separate file is used
// because the generated code structure (package, imports, handler pattern, marker format)
// differs fundamentally from the single-GVK webhook template.
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

	f.TemplateBody = multiGVKWebhookTemplate

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

// DefaultingPath returns the path for the defaulting webhook.
func (f *MultiGVKWebhook) DefaultingPath() string {
	if f.Webhook.DefaultingPath != "" {
		return f.Webhook.DefaultingPath
	}
	return "/mutate-" + strings.ToLower(f.Webhook.Name)
}

// ValidationPath returns the path for the validation webhook.
func (f *MultiGVKWebhook) ValidationPath() string {
	if f.Webhook.ValidationPath != "" {
		return f.Webhook.ValidationPath
	}
	return "/validate-" + strings.ToLower(f.Webhook.Name)
}

// MarkerGroups returns the semicolon-separated group list for the +kubebuilder:webhook marker.
func (f *MultiGVKWebhook) MarkerGroups() string {
	var parts []string
	for _, g := range f.Webhook.Groups {
		if g == "" || g == "core" {
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
const multiGVKWebhookTemplate = `{{ .Boilerplate }}

package webhook

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

{{ if .Webhook.Defaulting }}
// +kubebuilder:webhook:path={{ .DefaultingPath }},mutating=true,failurePolicy=fail,sideEffects=None,groups={{ .MarkerGroups }},resources={{ .MarkerKinds }},verbs=create;update,versions={{ .MarkerVersions }},name=m{{ lower .Webhook.Name }}.kb.io,admissionReviewVersions=v1
{{ end }}
{{ if .Webhook.Validation }}
// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path={{ .ValidationPath }},mutating=false,failurePolicy=fail,sideEffects=None,groups={{ .MarkerGroups }},resources={{ .MarkerKinds }},verbs=create;update,versions={{ .MarkerVersions }},name=v{{ lower .Webhook.Name }}.kb.io,admissionReviewVersions=v1
{{ end }}

{{ if and .Webhook.Defaulting .Webhook.Validation }}
// {{ .HandlerName }} mutates and validates intercepted resources.
{{ else if .Webhook.Defaulting }}
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
	{{ if and .Webhook.Defaulting .Webhook.Validation }}
	// TODO(user): fill in your webhook logic.
	{{ else if .Webhook.Defaulting }}
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
