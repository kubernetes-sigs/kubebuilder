/*
Copyright 2018 The Kubernetes authors.

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

package mutating

import (
	"context"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

func init() {
	webhookName := "mutating-update-namespaces"
	if HandlerMap[webhookName] == nil {
		HandlerMap[webhookName] = []admission.Handler{}
	}
	HandlerMap[webhookName] = append(HandlerMap[webhookName], &NamespaceUpdateHandler{})
}

// NamespaceUpdateHandler handles Namespace
type NamespaceUpdateHandler struct {
	// Client  client.Client

	// Decoder decodes objects
	Decoder types.Decoder
}

func (h *NamespaceUpdateHandler) mutatingNamespaceFn(ctx context.Context, obj *corev1.Namespace) error {
	// TODO(user): implement your admission logic
	return nil
}

var _ admission.Handler = &NamespaceUpdateHandler{}

// Handle handles admission requests.
func (h *NamespaceUpdateHandler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &corev1.Namespace{}

	err := h.Decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}
	copy := obj.DeepCopy()

	err = h.mutatingNamespaceFn(ctx, copy)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.PatchResponse(obj, copy)
}

//var _ inject.Client = &NamespaceUpdateHandler{}
//
//// InjectClient injects the client into the NamespaceUpdateHandler
//func (h *NamespaceUpdateHandler) InjectClient(c client.Client) error {
//	h.Client = c
//	return nil
//}

var _ inject.Decoder = &NamespaceUpdateHandler{}

// InjectDecoder injects the decoder into the NamespaceUpdateHandler
func (h *NamespaceUpdateHandler) InjectDecoder(d types.Decoder) error {
	h.Decoder = d
	return nil
}
