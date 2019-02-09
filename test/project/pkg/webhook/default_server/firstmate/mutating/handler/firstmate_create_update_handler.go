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
	"encoding/json"

	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
	crewv1 "sigs.k8s.io/kubebuilder/test/project/pkg/apis/crew/v1"
)

var CreateUpdateHandlers []admission.Handler

func init() {
	CreateUpdateHandlers = append(CreateUpdateHandlers, &FirstMateCreateUpdateHandler{})
}

// FirstMateCreateUpdateHandler handles FirstMate
type FirstMateCreateUpdateHandler struct {
	// To use the client, you need to do the following:
	// - uncomment it
	// - import sigs.k8s.io/controller-runtime/pkg/client
	// - uncomment the InjectClient method at the bottom of this file.
	// Client  client.Client

	// Decoder decodes objects
	Decoder types.Decoder
}

func (h *FirstMateCreateUpdateHandler) mutatingFirstMateFn(ctx context.Context, obj *crewv1.FirstMate) error {
	// TODO(user): implement your admission logic
	return nil
}

var _ admission.Handler = &FirstMateCreateUpdateHandler{}

// Handle handles admission requests.
func (h *FirstMateCreateUpdateHandler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &crewv1.FirstMate{}

	err := h.Decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	err = h.mutatingFirstMateFn(ctx, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	marshalled, err := json.Marshal(obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.AdmissionRequest.Object.Raw, marshalled)
}

//var _ inject.Client = &FirstMateCreateUpdateHandler{}
//
//// InjectClient injects the client into the FirstMateCreateUpdateHandler
//func (h *FirstMateCreateUpdateHandler) InjectClient(c client.Client) error {
//	h.Client = c
//	return nil
//}

var _ inject.Decoder = &FirstMateCreateUpdateHandler{}

// InjectDecoder injects the decoder into the FirstMateCreateUpdateHandler
func (h *FirstMateCreateUpdateHandler) InjectDecoder(d types.Decoder) error {
	h.Decoder = d
	return nil
}
