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

	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
	crewv1 "sigs.k8s.io/controller-tools/test/pkg/apis/crew/v1"
)

func init() {
	webhookName := "mutating-delete-firstmates"
	if HandlerMap[webhookName] == nil {
		HandlerMap[webhookName] = []admission.Handler{}
	}
	HandlerMap[webhookName] = append(HandlerMap[webhookName], &FirstMateDeleteHandler{})
}

// FirstMateDeleteHandler handles FirstMate
type FirstMateDeleteHandler struct {
	// Client  client.Client

	// Decoder decodes objects
	Decoder types.Decoder
}

func (h *FirstMateDeleteHandler) mutatingFirstMateFn(ctx context.Context, obj *crewv1.FirstMate) error {
	// TODO(user): implement your admission logic
	return nil
}

var _ admission.Handler = &FirstMateDeleteHandler{}

// Handle handles admission requests.
func (h *FirstMateDeleteHandler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &crewv1.FirstMate{}

	err := h.Decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}
	copy := obj.DeepCopy()

	err = h.mutatingFirstMateFn(ctx, copy)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.PatchResponse(obj, copy)
}

//var _ inject.Client = &FirstMateDeleteHandler{}
//
//// InjectClient injects the client into the FirstMateDeleteHandler
//func (h *FirstMateDeleteHandler) InjectClient(c client.Client) error {
//	h.Client = c
//	return nil
//}

var _ inject.Decoder = &FirstMateDeleteHandler{}

// InjectDecoder injects the decoder into the FirstMateDeleteHandler
func (h *FirstMateDeleteHandler) InjectDecoder(d types.Decoder) error {
	h.Decoder = d
	return nil
}
