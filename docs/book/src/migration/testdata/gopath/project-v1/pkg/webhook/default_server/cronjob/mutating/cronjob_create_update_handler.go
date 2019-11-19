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

	batchv1 "project/pkg/apis/batch/v1"
)

func init() {
	webhookName := "mutating-create-update-cronjob"
	if HandlerMap[webhookName] == nil {
		HandlerMap[webhookName] = []admission.Handler{}
	}
	HandlerMap[webhookName] = append(HandlerMap[webhookName], &CronJobCreateUpdateHandler{})
}

// +kubebuilder:webhook:groups=batch.tutorial.kubebuilder.io,versions=v1,resources=cronjobs,verbs=create;update
// +kubebuilder:webhook:name=mutating-create-update-cronjob.tutorial.kubebuilder.io
// +kubebuilder:webhook:path=/mutating-create-update-cronjob
// +kubebuilder:webhook:type=mutating,failure-policy=fail

// CronJobCreateUpdateHandler handles CronJob
type CronJobCreateUpdateHandler struct {
	// To use the client, you need to do the following:
	// - uncomment it
	// - import sigs.k8s.io/controller-runtime/pkg/client
	// - uncomment the InjectClient method at the bottom of this file.
	// Client  client.Client

	// Decoder decodes objects
	Decoder types.Decoder
}

func (h *CronJobCreateUpdateHandler) mutatingCronJobFn(ctx context.Context, obj *batchv1.CronJob) error {
	if obj.Spec.ConcurrencyPolicy == "" {
		obj.Spec.ConcurrencyPolicy = batchv1.AllowConcurrent
	}
	if obj.Spec.Suspend == nil {
		obj.Spec.Suspend = new(bool)
	}
	if obj.Spec.SuccessfulJobsHistoryLimit == nil {
		obj.Spec.SuccessfulJobsHistoryLimit = new(int32)
		*obj.Spec.SuccessfulJobsHistoryLimit = 3
	}
	if obj.Spec.FailedJobsHistoryLimit == nil {
		obj.Spec.FailedJobsHistoryLimit = new(int32)
		*obj.Spec.FailedJobsHistoryLimit = 1
	}
	return nil
}

var _ admission.Handler = &CronJobCreateUpdateHandler{}

// Handle handles admission requests.
func (h *CronJobCreateUpdateHandler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &batchv1.CronJob{}

	err := h.Decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}
	copy := obj.DeepCopy()

	err = h.mutatingCronJobFn(ctx, copy)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.PatchResponse(obj, copy)
}

//var _ inject.Client = &CronJobCreateUpdateHandler{}
//
//// InjectClient injects the client into the CronJobCreateUpdateHandler
//func (h *CronJobCreateUpdateHandler) InjectClient(c client.Client) error {
//	h.Client = c
//	return nil
//}

var _ inject.Decoder = &CronJobCreateUpdateHandler{}

// InjectDecoder injects the decoder into the CronJobCreateUpdateHandler
func (h *CronJobCreateUpdateHandler) InjectDecoder(d types.Decoder) error {
	h.Decoder = d
	return nil
}
