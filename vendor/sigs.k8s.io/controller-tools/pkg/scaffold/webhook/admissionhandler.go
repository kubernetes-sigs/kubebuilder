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

var _ input.File = &AddAdmissionWebhookBuilderHandler{}

// AdmissionHandler scaffolds an admission handler
type AdmissionHandler struct {
	input.Input

	// ResourcePackage is the package of the Resource
	ResourcePackage string

	// GroupDomain is the Group + "." + Domain for the Resource
	GroupDomain string

	// Resource is a resource in the API group
	Resource *resource.Resource

	Config

	BuilderName string

	OperationsString string

	Mutate bool
}

// GetInput implements input.File
func (a *AdmissionHandler) GetInput() (input.Input, error) {
	a.ResourcePackage, a.GroupDomain = getResourceInfo(coreGroups, a.Resource, a.Input)
	a.Type = strings.ToLower(a.Type)
	if a.Type == "mutating" {
		a.Mutate = true
	}
	a.BuilderName = builderName(a.Config, a.Resource.Resource)
	ops := make([]string, len(a.Operations))
	for i, op := range a.Operations {
		ops[i] = strings.Title(op)
	}
	a.OperationsString = strings.Join(ops, "")

	if a.Path == "" {
		a.Path = filepath.Join("pkg", "webhook",
			fmt.Sprintf("%s_server", a.Server),
			a.Resource.Resource,
			a.Type,
			fmt.Sprintf("%s_%s_handler.go", a.Resource.Resource, strings.Join(a.Operations, "_")))
	}
	a.TemplateBody = addAdmissionHandlerTemplate
	return a.Input, nil
}

var addAdmissionHandlerTemplate = `{{ .Boilerplate }}

package {{ .Type }}

import (
	"context"
	"net/http"

	{{ .Resource.Group}}{{ .Resource.Version }} "{{ .ResourcePackage }}/{{ .Resource.Group}}/{{ .Resource.Version }}"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
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
	// Client  client.Client

	// Decoder decodes objects
	Decoder types.Decoder
}
{{ if .Mutate }}
func (h *{{ .Resource.Kind }}{{ .OperationsString }}Handler) {{ .Type }}{{ .Resource.Kind }}Fn(ctx context.Context, obj *{{ .Resource.Group}}{{ .Resource.Version }}.{{ .Resource.Kind }}) error {
	// TODO(user): implement your admission logic
	return nil
}
{{ else }}
func (h *{{ .Resource.Kind }}{{ .OperationsString }}Handler) {{ .Type }}{{ .Resource.Kind }}Fn(ctx context.Context, obj *{{ .Resource.Group}}{{ .Resource.Version }}.{{ .Resource.Kind }}) (bool, string, error) {
	// TODO(user): implement your admission logic
	return true, "allowed to be admitted", nil
}
{{ end }}
var _ admission.Handler = &{{ .Resource.Kind }}{{ .OperationsString }}Handler{}
{{ if .Mutate }}
// Handle handles admission requests.
func (h *{{ .Resource.Kind }}{{ .OperationsString }}Handler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &{{ .Resource.Group}}{{ .Resource.Version }}.{{ .Resource.Kind }}{}

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
	obj := &{{ .Resource.Group}}{{ .Resource.Version }}.{{ .Resource.Kind }}{}

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
