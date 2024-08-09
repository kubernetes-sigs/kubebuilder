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
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
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

	// Define value for AdmissionReviewVersions marker
	AdmissionReviewVersions string

	Force bool
}

// SetTemplateDefaults implements file.Template
func (f *Webhook) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup && f.Resource.Group != "" {
			f.Path = filepath.Join("api", "%[group]", "%[version]", "%[kind]_webhook.go")
		} else {
			f.Path = filepath.Join("api", "%[version]", "%[kind]_webhook.go")
		}
	}

	f.Path = f.Resource.Replacer().Replace(f.Path)
	log.Println(f.Path)

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
	f.QualifiedGroupWithDash = strings.Replace(f.Resource.QualifiedGroup(), ".", "-", -1)

	return nil
}

const (
	webhookTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	{{- if or .Resource.HasValidationWebhook .Resource.HasDefaultingWebhook }}
	"context"
	"fmt"
	{{- end }}

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	{{- if or .Resource.HasValidationWebhook .Resource.HasDefaultingWebhook }}
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	{{- end }}
	{{- if .Resource.HasValidationWebhook }}
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	{{- end }}
)

// nolint:unused
// log is for logging in this package.
var {{ lower .Resource.Kind }}log = logf.Log.WithName("{{ lower .Resource.Kind }}-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *{{ .Resource.Kind }}) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		{{- if .Resource.HasValidationWebhook }}
		WithValidator(&{{ .Resource.Kind }}CustomValidator{}).
		{{- end }}
		{{- if .Resource.HasDefaultingWebhook }}
		WithDefaulter(&{{ .Resource.Kind }}CustomDefaulter{}).
		{{- end }}
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
`

	//nolint:lll
	defaultingWebhookTemplate = `
// +kubebuilder:webhook:{{ if ne .Resource.Webhooks.WebhookVersion "v1" }}webhookVersions={{"{"}}{{ .Resource.Webhooks.WebhookVersion }}{{"}"}},{{ end }}path=/mutate-{{ .QualifiedGroupWithDash }}-{{ .Resource.Version }}-{{ lower .Resource.Kind }},mutating=true,failurePolicy=fail,sideEffects=None,groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }},verbs=create;update,versions={{ .Resource.Version }},name=m{{ lower .Resource.Kind }}.kb.io,admissionReviewVersions={{ .AdmissionReviewVersions }}

// +kubebuilder:object:generate=false
// {{ .Resource.Kind }}CustomDefaulter struct is responsible for setting default values on the custom resource of the 
// Kind {{ .Resource.Kind }} when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type {{ .Resource.Kind }}CustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &{{ .Resource.Kind }}CustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind {{ .Resource.Kind }}
func (d *{{ .Resource.Kind }}CustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	{{ lower .Resource.Kind }}, ok := obj.(*{{ .Resource.Kind }})
	if !ok {
		return fmt.Errorf("expected an {{ .Resource.Kind }} object but got %T", obj)
	}	
	{{ lower .Resource.Kind }}log.Info("Defaulting for {{ .Resource.Kind }}", "name", {{ lower .Resource.Kind }}.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}
`

	//nolint:lll
	validatingWebhookTemplate = `
// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:{{ if ne .Resource.Webhooks.WebhookVersion "v1" }}webhookVersions={{"{"}}{{ .Resource.Webhooks.WebhookVersion }}{{"}"}},{{ end }}path=/validate-{{ .QualifiedGroupWithDash }}-{{ .Resource.Version }}-{{ lower .Resource.Kind }},mutating=false,failurePolicy=fail,sideEffects=None,groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }},verbs=create;update,versions={{ .Resource.Version }},name=v{{ lower .Resource.Kind }}.kb.io,admissionReviewVersions={{ .AdmissionReviewVersions }}

// +kubebuilder:object:generate=false
// {{ .Resource.Kind }}CustomValidator struct is responsible for validating the {{ .Resource.Kind }} resource 
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type {{ .Resource.Kind }}CustomValidator struct{
	//TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &{{ .Resource.Kind }}CustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type {{ .Resource.Kind }}
func (v *{{ .Resource.Kind }}CustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	{{ lower .Resource.Kind }}, ok := obj.(*{{ .Resource.Kind }})
	if !ok {
		return nil, fmt.Errorf("expected a {{ .Resource.Kind }} object but got %T", obj)
	}
	{{ lower .Resource.Kind }}log.Info("Validation for {{ .Resource.Kind }} upon creation", "name", {{ lower .Resource.Kind }}.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type {{ .Resource.Kind }}
func (v *{{ .Resource.Kind }}CustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	{{ lower .Resource.Kind }}, ok := newObj.(*{{ .Resource.Kind }})
	if !ok {
		return nil, fmt.Errorf("expected a {{ .Resource.Kind }} object but got %T", newObj)
	}
	{{ lower .Resource.Kind }}log.Info("Validation for {{ .Resource.Kind }} upon update", "name", {{ lower .Resource.Kind }}.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type {{ .Resource.Kind }}
func (v *{{ .Resource.Kind }}CustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	{{ lower .Resource.Kind }}, ok := obj.(*{{ .Resource.Kind }})
	if !ok {
		return nil, fmt.Errorf("expected a {{ .Resource.Kind }} object but got %T", obj)
	}
	{{ lower .Resource.Kind }}log.Info("Validation for {{ .Resource.Kind }} upon deletion", "name", {{ lower .Resource.Kind }}.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
`
)
