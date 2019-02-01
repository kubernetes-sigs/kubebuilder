/*
Copyright 2018 The Kubernetes Authors.

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

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
)

var _ input.File = &AdmissionWebhookBuilder{}

// AdmissionWebhookBuilder scaffolds adds a new webhook server.
type AdmissionWebhookBuilder struct {
	input.Input

	// Resource is a resource in the API group
	Resource *resource.Resource

	// ResourcePackage is the package of the Resource
	ResourcePackage string

	// GroupDomain is the Group + "." + Domain for the Resource
	GroupDomain string

	Config

	BuilderName string

	OperationsString string

	OperationsUpperString string

	OperationsUpperStringWSemicolon string

	Mutating bool
}

// GetInput implements input.File
func (a *AdmissionWebhookBuilder) GetInput() (input.Input, error) {
	a.ResourcePackage, a.GroupDomain = getResourceInfo(coreGroups, a.Resource, a.Input)

	if a.Type == "mutating" {
		a.Mutating = true
	}
	a.Type = strings.ToLower(a.Type)
	a.BuilderName = builderName(a.Config, strings.ToLower(a.Resource.Kind))
	ops := make([]string, len(a.Operations))
	for i, op := range a.Operations {
		ops[i] = strings.Title(op)
	}
	a.OperationsUpperString = strings.Join(ops, "")
	a.OperationsUpperStringWSemicolon = strings.Join(ops, ";")
	a.OperationsString = strings.Join(a.Operations, "")

	if a.Path == "" {
		a.Path = filepath.Join("pkg", "webhook",
			fmt.Sprintf("%s_server", a.Server),
			strings.ToLower(a.Resource.Kind),
			a.Type,
			fmt.Sprintf("%s_webhook.go", strings.Join(a.Operations, "_")))
	}
	a.TemplateBody = admissionWebhookBuilderTemplate
	return a.Input, nil
}

var admissionWebhookBuilderTemplate = `{{ .Boilerplate }}

package {{ .Type }}

import (
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
	handler {{ if .Mutating }}"{{ .Repo }}/pkg/webhook/{{ .Server }}_server/{{ lower .Resource.Kind }}/mutating/handler"
{{ else }}"{{ .Repo }}/pkg/webhook/{{ .Server }}_server/{{ lower .Resource.Kind }}/validating/handler"
{{ end }}
)

// +kubebuilder:webhook:groups={{ .Resource.Group }},resources={{ .Resource.Resource }}
// +kubebuilder:webhook:versions={{ .Resource.Version }}
// +kubebuilder:webhook:verbs={{ .OperationsUpperStringWSemicolon }}
// +kubebuilder:webhook:name={{ .BuilderName }}.{{ .Domain }},path=/{{ .BuilderName }}
// +kubebuilder:webhook:type={{ if .Mutating }}mutating{{ else }}validating{{ end }}
// +kubebuilder:webhook:failure-policy=Fail
func init() {
	var wh webhook.Webhook
	builderName := "{{ .BuilderName }}"
	wh = builder.
		NewWebhookBuilder().
		Name(builderName + ".{{ .Domain }}").
		Path("/" + builderName).
{{ if .Mutating }}	Mutating().
{{ else }}	Validating().
{{ end }}
		Handlers(handler.{{ .OperationsUpperString }}Handlers...).
		Build()
	{{ .Resource.Kind }}Webhooks = append({{ .Resource.Kind }}Webhooks, wh)
}
`
