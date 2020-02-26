/*
Copyright 2019 The Kubernetes Authors.

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

package webhook

import (
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &Webhook{}

// Webhook scaffolds a Webhook for a Resource
type Webhook struct { // nolint:maligned
	file.TemplateMixin
	file.MultiGroupMixin
	file.BoilerplateMixin
	file.ResourceMixin

	// Is the Group domain for the Resource replacing '.' with '-'
	GroupDomainWithDash string

	// If scaffold the defaulting webhook
	Defaulting bool
	// If scaffold the validating webhook
	Validating bool
}

// SetTemplateDefaults implements input.Template
func (f *Webhook) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup {
			f.Path = filepath.Join("apis", "%[group]", "%[version]", "%[kind]_webhook.go")
		} else {
			f.Path = filepath.Join("api", "%[version]", "%[kind]_webhook.go")
		}
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)
	fmt.Println(f.Path)

	webhookTemplate := webhookTemplate
	if f.Defaulting {
		webhookTemplate = webhookTemplate + defaultingWebhookTemplate
	}
	if f.Validating {
		webhookTemplate = webhookTemplate + validatingWebhookTemplate
	}
	f.TemplateBody = webhookTemplate

	f.IfExistsAction = file.Error

	f.GroupDomainWithDash = strings.Replace(f.Resource.Domain, ".", "-", -1)

	return nil
}

const (
	webhookTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	{{- if or .Validating .Defaulting }}
	"k8s.io/apimachinery/pkg/runtime"
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

	//nolint:lll
	defaultingWebhookTemplate = `
// +kubebuilder:webhook:path=/mutate-{{ .GroupDomainWithDash }}-{{ .Resource.Version }}-{{ lower .Resource.Kind }},mutating=true,failurePolicy=fail,groups={{ .Resource.Domain }},resources={{ .Resource.Plural }},verbs=create;update,versions={{ .Resource.Version }},name=m{{ lower .Resource.Kind }}.kb.io

var _ webhook.Defaulter = &{{ .Resource.Kind }}{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *{{ .Resource.Kind }}) Default() {
	{{ lower .Resource.Kind }}log.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}
`
	//nolint:lll
	validatingWebhookTemplate = `
// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-{{ .GroupDomainWithDash }}-{{ .Resource.Version }}-{{ lower .Resource.Kind }},mutating=false,failurePolicy=fail,groups={{ .Resource.Domain }},resources={{ .Resource.Plural }},versions={{ .Resource.Version }},name=v{{ lower .Resource.Kind }}.kb.io

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
