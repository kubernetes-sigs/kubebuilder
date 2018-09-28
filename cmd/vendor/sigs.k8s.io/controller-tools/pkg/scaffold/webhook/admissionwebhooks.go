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

var _ input.File = &Server{}

// AdmissionWebhooks scaffolds how to construct a webhook server and register webhooks.
type AdmissionWebhooks struct {
	input.Input

	// Resource is a resource in the API group
	Resource *resource.Resource

	Config
}

// GetInput implements input.File
func (a *AdmissionWebhooks) GetInput() (input.Input, error) {
	a.Server = strings.ToLower(a.Server)
	a.Type = strings.ToLower(a.Type)
	if a.Path == "" {
		a.Path = filepath.Join("pkg", "webhook",
			fmt.Sprintf("%s_server", a.Server),
			strings.ToLower(a.Resource.Resource),
			a.Type, "webhooks.go")
	}
	a.TemplateBody = webhooksTemplate
	return a.Input, nil
}

var webhooksTemplate = `{{ .Boilerplate }}

package {{ .Type }}

import (
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
)

var (
	// Builders contain admission webhook builders
	Builders = map[string]*builder.WebhookBuilder{}
	// HandlerMap contains admission webhook handlers
	HandlerMap = map[string][]admission.Handler{}
)
`
