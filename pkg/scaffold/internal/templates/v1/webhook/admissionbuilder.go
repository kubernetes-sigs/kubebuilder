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

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &AdmissionWebhookBuilder{}

// AdmissionWebhookBuilder scaffolds adds a new webhook server.
type AdmissionWebhookBuilder struct {
	file.Input
	file.ResourceMixin

	Config

	BuilderName string

	OperationsParameterString string

	Mutating bool
}

// GetInput implements input.Template
func (f *AdmissionWebhookBuilder) GetInput() (file.Input, error) {
	if f.Type == "mutating" {
		f.Mutating = true
	}
	f.Type = strings.ToLower(f.Type)
	f.BuilderName = builderName(f.Config, strings.ToLower(f.Resource.Kind))
	ops := make([]string, len(f.Operations))
	for i, op := range f.Operations {
		ops[i] = "admissionregistrationv1beta1." + strings.Title(op)
	}
	f.OperationsParameterString = strings.Join(ops, ", ")

	if f.Path == "" {
		f.Path = filepath.Join("pkg", "webhook",
			fmt.Sprintf("%s_server", f.Server),
			strings.ToLower(f.Resource.Kind),
			f.Type,
			fmt.Sprintf("%s_webhook.go", strings.Join(f.Operations, "_")))
	}
	f.TemplateBody = admissionWebhookBuilderTemplate
	return f.Input, nil
}

const admissionWebhookBuilderTemplate = `{{ .Boilerplate }}

package {{ .Type }}

import (
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
	{{ .Resource.ImportAlias }} "{{ .Resource.Package }}"
)

func init() {
	builderName := "{{ .BuilderName }}"
	Builders[builderName] = builder.
		NewWebhookBuilder().
		Name(builderName + ".{{ .Domain }}").
		Path("/" + builderName).
{{ if .Mutating }}	Mutating().
{{ else }}	Validating().
{{ end }}
		Operations({{ .OperationsParameterString }}).
		FailurePolicy(admissionregistrationv1beta1.Fail).
		ForType(&{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{})
}
`
