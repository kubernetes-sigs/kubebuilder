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

package validating

import (
	"context"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
	shipv1beta1 "sigs.k8s.io/controller-tools/test/pkg/apis/ship/v1beta1"
)

func init() {
	webhookName := "validating-update-frigates"
	if HandlerMap[webhookName] == nil {
		HandlerMap[webhookName] = []admission.Handler{}
	}
	HandlerMap[webhookName] = append(HandlerMap[webhookName], &FrigateUpdateHandler{})
}

// FrigateUpdateHandler handles Frigate
type FrigateUpdateHandler struct {
	// Client  client.Client

	// Decoder decodes objects
	Decoder types.Decoder
}

func (h *FrigateUpdateHandler) validatingFrigateFn(ctx context.Context, obj *shipv1beta1.Frigate) (bool, string, error) {
	// TODO(user): implement your admission logic
	return true, "allowed to be admitted", nil
}

var _ admission.Handler = &FrigateUpdateHandler{}

// Handle handles admission requests.
func (h *FrigateUpdateHandler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &shipv1beta1.Frigate{}

	err := h.Decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	allowed, reason, err := h.validatingFrigateFn(ctx, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

//var _ inject.Client = &FrigateUpdateHandler{}
//
//// InjectClient injects the client into the FrigateUpdateHandler
//func (h *FrigateUpdateHandler) InjectClient(c client.Client) error {
//	h.Client = c
//	return nil
//}

var _ inject.Decoder = &FrigateUpdateHandler{}

// InjectDecoder injects the decoder into the FrigateUpdateHandler
func (h *FrigateUpdateHandler) InjectDecoder(d types.Decoder) error {
	h.Decoder = d
	return nil
}
