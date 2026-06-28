package webhooks

import (
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

var _ machinery.Template = &StandaloneWebhook{}

// StandaloneWebhook scaffolds a multi-GVK webhook that intercepts multiple resource types
type StandaloneWebhook struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin

	Force bool

	// Webhook holds the standalone webhook configuration
	Webhook resource.StandaloneWebhook

	// GroupValue is the escaped group string for the webhook marker
	GroupValue string

	// ResourceValue is the resource string for the webhook marker
	ResourceValue string

	// VersionValue is the version string for the webhook marker
	VersionValue string

	// HandlerName is the Go struct name for the handler
	HandlerName string

	// GroupsMarker is the formatted groups for the +kubebuilder:webhook marker
	GroupsMarker string

	// ResourcesMarker is the formatted resources for the +kubebuilder:webhook marker
	ResourcesMarker string

	// VersionsMarker is the formatted versions for the +kubebuilder:webhook marker
	VersionsMarker string
}

// SetTemplateDefaults implements machinery.Template
func (f *StandaloneWebhook) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("internal", "webhook", "%[webhook-name]_webhook.go")
	}

	// Replace template placeholders
	f.Path = strings.ReplaceAll(f.Path, "%[webhook-name]", strings.ToLower(f.Webhook.Name))

	// Build marker values
	f.GroupsMarker = f.buildMarkerGroups()
	f.ResourcesMarker = f.buildMarkerResources()
	f.VersionsMarker = f.buildMarkerVersions()
	f.HandlerName = f.buildHandlerName()

	// Defaulting path
	defaultingPath := f.Webhook.DefaultingPath
	if defaultingPath == "" {
		defaultingPath = "/mutate-" + strings.ToLower(f.Webhook.Name)
	}

	// Validation path
	validationPath := f.Webhook.ValidationPath
	if validationPath == "" {
		validationPath = "/validate-" + strings.ToLower(f.Webhook.Name)
	}

	var templateBody string
	if f.Webhook.Defaulting && f.Webhook.Validation {
		templateBody = standaloneWebhookDefValTemplate
	} else if f.Webhook.Defaulting {
		templateBody = standaloneWebhookDefaultingTemplate
	} else {
		templateBody = standaloneWebhookValidationTemplate
	}

	f.TemplateBody = templateBody

	// Replace path markers in the template body
	f.TemplateBody = strings.ReplaceAll(f.TemplateBody, "%[def-path]", defaultingPath)
	f.TemplateBody = strings.ReplaceAll(f.TemplateBody, "%[val-path]", validationPath)

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.Error
	}

	return nil
}

func (f *StandaloneWebhook) buildHandlerName() string {
	parts := strings.Split(f.Webhook.Name, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "") + "Webhook"
}

func (f *StandaloneWebhook) buildMarkerGroups() string {
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

func (f *StandaloneWebhook) buildMarkerResources() string {
	return strings.Join(f.Webhook.Resources, ";")
}

func (f *StandaloneWebhook) buildMarkerVersions() string {
	return strings.Join(f.Webhook.Versions, ";")
}

//nolint:lll
const standaloneWebhookDefaultingTemplate = `{{ .Boilerplate }}

package webhook

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=%[def-path],mutating=true,failurePolicy=fail,sideEffects=None,groups={{ .GroupsMarker }},resources={{ .ResourcesMarker }},verbs=create;update,versions={{ .VersionsMarker }},name=m{{ lower .Webhook.Name }}.kb.io,admissionReviewVersions=v1

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
const standaloneWebhookValidationTemplate = `{{ .Boilerplate }}

package webhook

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=%[val-path],mutating=false,failurePolicy=fail,sideEffects=None,groups={{ .GroupsMarker }},resources={{ .ResourcesMarker }},verbs=create;update,versions={{ .VersionsMarker }},name=v{{ lower .Webhook.Name }}.kb.io,admissionReviewVersions=v1

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
const standaloneWebhookDefValTemplate = `{{ .Boilerplate }}

package webhook

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=%[def-path],mutating=true,failurePolicy=fail,sideEffects=None,groups={{ .GroupsMarker }},resources={{ .ResourcesMarker }},verbs=create;update,versions={{ .VersionsMarker }},name=m{{ lower .Webhook.Name }}.kb.io,admissionReviewVersions=v1

// +kubebuilder:webhook:path=%[val-path],mutating=false,failurePolicy=fail,sideEffects=None,groups={{ .GroupsMarker }},resources={{ .ResourcesMarker }},verbs=create;update,versions={{ .VersionsMarker }},name=v{{ lower .Webhook.Name }}.kb.io,admissionReviewVersions=v1

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
