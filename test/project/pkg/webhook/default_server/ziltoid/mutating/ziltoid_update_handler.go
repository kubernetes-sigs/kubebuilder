/*
Copyright 2019 The Kubernetes authors.

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
	ziltodiav1 "sigs.k8s.io/kubebuilder/test/project/pkg/apis/ziltodia/v1"
)

func init() {
	webhookName := "mutating-update-ziltoid"
	if HandlerMap[webhookName] == nil {
		HandlerMap[webhookName] = []admission.Handler{}
	}
	HandlerMap[webhookName] = append(HandlerMap[webhookName], &ZiltoidUpdateHandler{})
}

// ZiltoidUpdateHandler handles Ziltoid
type ZiltoidUpdateHandler struct {
	// To use the client, you need to do the following:
	// - uncomment it
	// - import sigs.k8s.io/controller-runtime/pkg/client
	// - uncomment the InjectClient method at the bottom of this file.
	// Client  client.Client

	// Decoder decodes objects
	Decoder types.Decoder
}

func (h *ZiltoidUpdateHandler) mutatingZiltoidFn(ctx context.Context, obj *ziltodiav1.Ziltoid) error {
	// TODO(user): implement your admission logic
	return nil
}

var _ admission.Handler = &ZiltoidUpdateHandler{}

// Handle handles admission requests.
func (h *ZiltoidUpdateHandler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &ziltodiav1.Ziltoid{}

	err := h.Decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}
	copy := obj.DeepCopy()

	err = h.mutatingZiltoidFn(ctx, copy)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.PatchResponse(obj, copy)
}

//var _ inject.Client = &ZiltoidUpdateHandler{}
//
//// InjectClient injects the client into the ZiltoidUpdateHandler
//func (h *ZiltoidUpdateHandler) InjectClient(c client.Client) error {
//	h.Client = c
//	return nil
//}

var _ inject.Decoder = &ZiltoidUpdateHandler{}

// InjectDecoder injects the decoder into the ZiltoidUpdateHandler
func (h *ZiltoidUpdateHandler) InjectDecoder(d types.Decoder) error {
	h.Decoder = d
	return nil
}
