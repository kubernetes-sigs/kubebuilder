/*
Copyright 2021 The Kubernetes authors.

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

package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,sideEffects=None,groups=core,resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io,admissionReviewVersions={v1,v1beta1}

type PodDefaulter struct {
	client.Client
	Log     logr.Logger
	decoder *admission.Decoder
}

var _ admission.DecoderInjector = &PodDefaulter{}

func (d *PodDefaulter) Handle(ctx context.Context, req admission.Request) admission.Response {
	_ = d.Log.WithValues("pod", client.ObjectKey{Namespace: req.Namespace, Name: req.Name})
	r := &corev1.Pod{}
	err := d.decoder.Decode(req, r)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Option 1. Extract .Default() to make it resemble
	// admission.Defaulter. This probably makes a bit more sense in
	// validator. I tried to go with .default() but it doesn't compiles
	// `default` is a keyword.

	d.Default(r)

	// End of option 1.

	// Option 2. Simply add TODO here and remove .default().

	// TODO(user): fill in your defaulting logic.

	// End of option 2.

	marshaled, err := json.Marshal(r)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaled)
}

func (d *PodDefaulter) Default(r *corev1.Pod) {
	// TODO(user): fill in your defaulting logic.
}

func (d *PodDefaulter) SetupWebhookWithManager(mgr ctrl.Manager) error {
	hookServer := mgr.GetWebhookServer()
	hookServer.Register("/mutate-v1-pod", &webhook.Admission{Handler: d})
	return nil
}

func (d *PodDefaulter) InjectDecoder(decoder *admission.Decoder) error {
	d.decoder = decoder
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-v1-pod,mutating=false,failurePolicy=fail,sideEffects=None,groups=core,resources=pods,verbs=create;update,versions=v1,name=vpod.kb.io,admissionReviewVersions={v1,v1beta1}

type PodValidator struct {
	client.Client
	Log     logr.Logger
	decoder *admission.Decoder
}

var _ admission.DecoderInjector = &PodValidator{}

func (v *PodValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	r := &corev1.Pod{}
	if err := v.decoder.Decode(req, r); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Option 1. Extract .Validate*() methods to make it resemble
	// admission.Validator to make controller-runtime users feel at home.
	// Initially I wanted them to be private methods (i.e. .validate*())
	// but since .default() is not possible (see option 1. in defaulter)
	// I went with this.
	//
	// Two questions that come to mind with this approach:
	//
	// 1. Should admission.Denied be supported?
	// 2. Should there be something in admission.Allowed message?

	var validateErr error
	switch req.Operation {
	case admissionv1.Create:
		validateErr = v.ValidateCreate(r)
	case admissionv1.Update:
		old := &corev1.Pod{}
		if err := v.decoder.DecodeRaw(req.OldObject, old); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		validateErr = v.ValidateUpdate(r, old)
	case admissionv1.Delete:
		validateErr = v.ValidateDelete(r)
	}

	if validateErr != nil {
		return admission.Errored(http.StatusBadRequest, validateErr)
	}
	return admission.Allowed("")

	// End of option 1.

	// Option 2. Simply add TODO here and remove .validate**() methods.

	// TODO(user): fill in your validation logic upon object creation.
	return admission.Errored(http.StatusBadRequest, errors.New("not implemented"))

	// End of option 2.
}

func (v *PodValidator) ValidateCreate(r *corev1.Pod) error {
	log := v.Log.WithValues("pod", client.ObjectKeyFromObject(r))
	log.Info("validate create")

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

func (v *PodValidator) ValidateUpdate(r, old *corev1.Pod) error {
	log := v.Log.WithValues("pod", client.ObjectKeyFromObject(r))
	log.Info("validate udpate")

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

func (v *PodValidator) ValidateDelete(r *corev1.Pod) error {
	log := v.Log.WithValues("pod", client.ObjectKeyFromObject(r))
	log.Info("validate delete")

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
func (v *PodValidator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	hookServer := mgr.GetWebhookServer()
	hookServer.Register("/validate-v1-pod", &webhook.Admission{Handler: v})
	return nil
}

func (v *PodValidator) InjectDecoder(decoder *admission.Decoder) error {
	v.decoder = decoder
	return nil
}
