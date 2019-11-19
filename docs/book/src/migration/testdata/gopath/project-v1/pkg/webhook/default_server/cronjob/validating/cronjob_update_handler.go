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

package validating

import (
	"context"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"

	batchv1 "project/pkg/apis/batch/v1"
)

func init() {
	webhookName := "validating-update-cronjob"
	if HandlerMap[webhookName] == nil {
		HandlerMap[webhookName] = []admission.Handler{}
	}
	HandlerMap[webhookName] = append(HandlerMap[webhookName], &CronJobUpdateHandler{})
}

// +kubebuilder:webhook:groups=batch.tutorial.kubebuilder.io,versions=v1,resources=cronjobs,verbs=update
// +kubebuilder:webhook:name=validating-update-cronjob.tutorial.kubebuilder.io
// +kubebuilder:webhook:path=/validating-update-cronjob
// +kubebuilder:webhook:type=validating,failure-policy=fail

// CronJobUpdateHandler handles CronJob
type CronJobUpdateHandler struct {
	// To use the client, you need to do the following:
	// - uncomment it
	// - import sigs.k8s.io/controller-runtime/pkg/client
	// - uncomment the InjectClient method at the bottom of this file.
	// Client  client.Client

	// Decoder decodes objects
	Decoder types.Decoder
}

func (h *CronJobUpdateHandler) validatingCronJobFn(ctx context.Context, obj *batchv1.CronJob) (bool, string, error) {
	if err := validateCronJob(obj); err != nil {
		return false, "not allowed to be admitted", err
	}
	return true, "allowed to be admitted", nil
}

var _ admission.Handler = &CronJobUpdateHandler{}

// Handle handles admission requests.
func (h *CronJobUpdateHandler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &batchv1.CronJob{}

	err := h.Decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	allowed, reason, err := h.validatingCronJobFn(ctx, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

//var _ inject.Client = &CronJobUpdateHandler{}
//
//// InjectClient injects the client into the CronJobUpdateHandler
//func (h *CronJobUpdateHandler) InjectClient(c client.Client) error {
//	h.Client = c
//	return nil
//}

var _ inject.Decoder = &CronJobUpdateHandler{}

// InjectDecoder injects the decoder into the CronJobUpdateHandler
func (h *CronJobUpdateHandler) InjectDecoder(d types.Decoder) error {
	h.Decoder = d
	return nil
}
