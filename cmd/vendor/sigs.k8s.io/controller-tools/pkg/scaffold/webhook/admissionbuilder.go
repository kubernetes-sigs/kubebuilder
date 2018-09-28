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

	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
	"sigs.k8s.io/controller-tools/pkg/scaffold/resource"
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

	OperationsParameterString string

	Mutating bool
}

// GetInput implements input.File
func (a *AdmissionWebhookBuilder) GetInput() (input.Input, error) {
	a.ResourcePackage, a.GroupDomain = getResourceInfo(coreGroups, a.Resource, a.Input)

	if a.Type == "mutating" {
		a.Mutating = true
	}
	a.Type = strings.ToLower(a.Type)
	a.BuilderName = builderName(a.Config, a.Resource.Resource)
	ops := make([]string, len(a.Operations))
	for i, op := range a.Operations {
		ops[i] = "admissionregistrationv1beta1." + strings.Title(op)
	}
	a.OperationsParameterString = strings.Join(ops, ", ")

	if a.Path == "" {
		a.Path = filepath.Join("pkg", "webhook",
			fmt.Sprintf("%s_server", a.Server),
			a.Resource.Resource,
			a.Type,
			fmt.Sprintf("%s_webhook.go", strings.Join(a.Operations, "_")))
	}
	a.TemplateBody = admissionWebhookBuilderTemplate
	return a.Input, nil
}

var admissionWebhookBuilderTemplate = `{{ .Boilerplate }}

package {{ .Type }}

import (
	{{ .Resource.Group}}{{ .Resource.Version }} "{{ .ResourcePackage }}/{{ .Resource.Group}}/{{ .Resource.Version }}"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
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
		ForType(&{{ .Resource.Group}}{{ .Resource.Version }}.{{ .Resource.Kind }}{})
}
`
