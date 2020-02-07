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

var _ file.Template = &AddAdmissionWebhookBuilderHandler{}

// AdmissionHandler scaffolds an admission handler
type AdmissionHandler struct {
	file.Input
	file.ResourceMixin

	Config

	BuilderName string

	OperationsString string

	Mutate bool
}

// GetInput implements input.Template
func (f *AdmissionHandler) GetInput() (file.Input, error) {
	f.Type = strings.ToLower(f.Type)
	if f.Type == "mutating" {
		f.Mutate = true
	}
	f.BuilderName = builderName(f.Config, strings.ToLower(f.Resource.Kind))
	ops := make([]string, len(f.Operations))
	for i, op := range f.Operations {
		ops[i] = strings.Title(op)
	}
	f.OperationsString = strings.Join(ops, "")

	if f.Path == "" {
		f.Path = filepath.Join("pkg", "webhook",
			fmt.Sprintf("%s_server", f.Server),
			strings.ToLower(f.Resource.Kind),
			f.Type,
			fmt.Sprintf("%s_%s_handler.go", strings.ToLower(f.Resource.Kind), strings.Join(f.Operations, "_")))
	}
	f.TemplateBody = addAdmissionHandlerTemplate
	return f.Input, nil
}

// nolint:lll
const addAdmissionHandlerTemplate = `{{ .Boilerplate }}

package {{ .Type }}

import (
	"context"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
	{{ .Resource.ImportAlias }} "{{ .Resource.Package }}"
)

func init() {
	webhookName := "{{ .BuilderName }}"
	if HandlerMap[webhookName] == nil {
		HandlerMap[webhookName] = []admission.Handler{}
	}
	HandlerMap[webhookName] = append(HandlerMap[webhookName], &{{ .Resource.Kind }}{{ .OperationsString }}Handler{})
}

// {{ .Resource.Kind }}{{ .OperationsString }}Handler handles {{ .Resource.Kind }}
type {{ .Resource.Kind }}{{ .OperationsString }}Handler struct {
	// To use the client, you need to do the following:
	// - uncomment it
	// - import sigs.k8s.io/controller-runtime/pkg/client
	// - uncomment the InjectClient method at the bottom of this file.
	// Client  client.Client

	// Decoder decodes objects
	Decoder types.Decoder
}
{{ if .Mutate }}
func (h *{{ .Resource.Kind }}{{ .OperationsString }}Handler) {{ .Type }}{{ .Resource.Kind }}Fn(ctx context.Context, obj *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}) error {
	// TODO(user): implement your admission logic
	return nil
}
{{ else }}
func (h *{{ .Resource.Kind }}{{ .OperationsString }}Handler) {{ .Type }}{{ .Resource.Kind }}Fn(ctx context.Context, obj *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}) (bool, string, error) {
	// TODO(user): implement your admission logic
	return true, "allowed to be admitted", nil
}
{{ end }}
var _ admission.Handler = &{{ .Resource.Kind }}{{ .OperationsString }}Handler{}
{{ if .Mutate }}
// Handle handles admission requests.
func (h *{{ .Resource.Kind }}{{ .OperationsString }}Handler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}

	err := h.Decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}
	copy := obj.DeepCopy()

	err = h.{{ .Type }}{{ .Resource.Kind }}Fn(ctx, copy)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.PatchResponse(obj, copy)
}
{{ else }}
// Handle handles admission requests.
func (h *{{ .Resource.Kind }}{{ .OperationsString }}Handler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}

	err := h.Decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	allowed, reason, err := h.{{ .Type }}{{ .Resource.Kind }}Fn(ctx, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}
{{ end }}
//var _ inject.Client = &{{ .Resource.Kind }}{{ .OperationsString }}Handler{}
//
//// InjectClient injects the client into the {{ .Resource.Kind }}{{ .OperationsString }}Handler
//func (h *{{ .Resource.Kind }}{{ .OperationsString }}Handler) InjectClient(c client.Client) error {
//	h.Client = c
//	return nil
//}

var _ inject.Decoder = &{{ .Resource.Kind }}{{ .OperationsString }}Handler{}

// InjectDecoder injects the decoder into the {{ .Resource.Kind }}{{ .OperationsString }}Handler
func (h *{{ .Resource.Kind }}{{ .OperationsString }}Handler) InjectDecoder(d types.Decoder) error {
	h.Decoder = d
	return nil
}
`
