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
	creaturesv2alpha1 "sigs.k8s.io/controller-tools/test/pkg/apis/creatures/v2alpha1"
)

func init() {
	webhookName := "validating-create-krakens"
	if HandlerMap[webhookName] == nil {
		HandlerMap[webhookName] = []admission.Handler{}
	}
	HandlerMap[webhookName] = append(HandlerMap[webhookName], &KrakenCreateHandler{})
}

// KrakenCreateHandler handles Kraken
type KrakenCreateHandler struct {
	// Client  client.Client

	// Decoder decodes objects
	Decoder types.Decoder
}

func (h *KrakenCreateHandler) validatingKrakenFn(ctx context.Context, obj *creaturesv2alpha1.Kraken) (bool, string, error) {
	// TODO(user): implement your admission logic
	return true, "allowed to be admitted", nil
}

var _ admission.Handler = &KrakenCreateHandler{}

// Handle handles admission requests.
func (h *KrakenCreateHandler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &creaturesv2alpha1.Kraken{}

	err := h.Decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	allowed, reason, err := h.validatingKrakenFn(ctx, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

//var _ inject.Client = &KrakenCreateHandler{}
//
//// InjectClient injects the client into the KrakenCreateHandler
//func (h *KrakenCreateHandler) InjectClient(c client.Client) error {
//	h.Client = c
//	return nil
//}

var _ inject.Decoder = &KrakenCreateHandler{}

// InjectDecoder injects the decoder into the KrakenCreateHandler
func (h *KrakenCreateHandler) InjectDecoder(d types.Decoder) error {
	h.Decoder = d
	return nil
}
