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

	"github.com/robfig/cron"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	validationutils "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"

	batchv1 "project/pkg/apis/batch/v1"
)

func init() {
	webhookName := "validating-create-cronjob"
	if HandlerMap[webhookName] == nil {
		HandlerMap[webhookName] = []admission.Handler{}
	}
	HandlerMap[webhookName] = append(HandlerMap[webhookName], &CronJobCreateHandler{})
}

// +kubebuilder:webhook:groups=batch.tutorial.kubebuilder.io,versions=v1,resources=cronjobs,verbs=create
// +kubebuilder:webhook:name=validating-create-cronjob.tutorial.kubebuilder.io
// +kubebuilder:webhook:path=/validating-create-cronjob
// +kubebuilder:webhook:type=validating,failure-policy=fail

// CronJobCreateHandler handles CronJob
type CronJobCreateHandler struct {
	// To use the client, you need to do the following:
	// - uncomment it
	// - import sigs.k8s.io/controller-runtime/pkg/client
	// - uncomment the InjectClient method at the bottom of this file.
	// Client  client.Client

	// Decoder decodes objects
	Decoder types.Decoder
}

func (h *CronJobCreateHandler) validatingCronJobFn(ctx context.Context, obj *batchv1.CronJob) (bool, string, error) {
	if err := validateCronJob(obj); err != nil {
		return false, "not allowed to be admitted", err
	}
	return true, "allowed to be admitted", nil
}

var _ admission.Handler = &CronJobCreateHandler{}

// Handle handles admission requests.
func (h *CronJobCreateHandler) Handle(ctx context.Context, req types.Request) types.Response {
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

//var _ inject.Client = &CronJobCreateHandler{}
//
//// InjectClient injects the client into the CronJobCreateHandler
//func (h *CronJobCreateHandler) InjectClient(c client.Client) error {
//	h.Client = c
//	return nil
//}

var _ inject.Decoder = &CronJobCreateHandler{}

// InjectDecoder injects the decoder into the CronJobCreateHandler
func (h *CronJobCreateHandler) InjectDecoder(d types.Decoder) error {
	h.Decoder = d
	return nil
}

func validateCronJob(r *batchv1.CronJob) error {
	var allErrs field.ErrorList
	if err := validateCronJobName(r); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateCronJobSpec(r); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "batch.tutorial.kubebuilder.io", Kind: "CronJob"},
		r.Name, allErrs)
}

func validateCronJobSpec(r *batchv1.CronJob) *field.Error {
	// The field helpers from the kubernetes API machinery help us return nicely
	// structured validation errors.
	return validateScheduleFormat(
		r.Spec.Schedule,
		field.NewPath("spec").Child("schedule"))
}

func validateScheduleFormat(schedule string, fldPath *field.Path) *field.Error {
	if _, err := cron.ParseStandard(schedule); err != nil {
		return field.Invalid(fldPath, schedule, err.Error())
	}
	return nil
}

func validateCronJobName(r *batchv1.CronJob) *field.Error {
	if len(r.ObjectMeta.Name) > validationutils.DNS1035LabelMaxLength-11 {
		// The job name length is 63 character like all Kubernetes objects
		// (which must fit in a DNS subdomain). The cronjob controller appends
		// a 11-character suffix to the cronjob (`-$TIMESTAMP`) when creating
		// a job. The job name length limit is 63 characters. Therefore cronjob
		// names must have length <= 63-11=52. If we don't validate this here,
		// then job creation will fail later.
		return field.Invalid(field.NewPath("metadata").Child("name"), r.Name, "must be no more than 52 characters")
	}
	return nil
}
